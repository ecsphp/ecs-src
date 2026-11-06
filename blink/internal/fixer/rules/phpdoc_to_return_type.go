package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/PhpdocToReturnTypeFixer.php
//
// PhpdocToReturnType takes a single `@return` annotation of a non-mixed type and
// adds the matching native return type to the signature. Risky/experimental.
type PhpdocToReturnType struct {
	phpdocTypeOptions
}

func (f PhpdocToReturnType) WithConfig(config map[string]any) fixer.Fixer {
	f.phpdocTypeOptions = f.withConfig(config)
	return f
}

func (PhpdocToReturnType) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\PhpdocToReturnTypeFixer`
}

func (PhpdocToReturnType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/PhpdocToReturnTypeFixer.php"
}

func (f PhpdocToReturnType) Fix(s *tokens.Stream) bool {
	cfg := f.config()
	hasNamespace := phpdocFileHasNamespace(s)
	var excludes []string
	inserts := map[int][]token.Token{}

	for index := 0; index < s.Len(); index++ {
		t := s.At(index)
		if t.Kind == token.DocComment {
			excludes = append(excludes, phpdocTypesToExclude(t.Value)...)
			continue
		}
		if !kwIs(t, "function") && !kwIs(t, "fn") {
			continue
		}
		if name := sigNext(s, index); name >= 0 && taIsTString(s, name) {
			switch strings.ToLower(s.At(name).Value) {
			case "__construct", "__destruct", "__clone":
				continue
			}
		}
		docIndex := phpdocFindFunctionDocComment(s, index)
		if docIndex < 0 {
			continue
		}
		anns := phpdocAnnotationsOfType(s, docIndex, "return")
		if len(anns) != 1 || !anns[0].hasTypes {
			continue
		}
		members, glue := docTypeMembers(anns[0].typesContent)

		returnType, isNullable, ok := phpdocResolveType(members, glue, true, phpdocReturnSkipped, cfg, hasNamespace)
		if !ok {
			continue
		}
		if slices.Contains(excludes, returnType) {
			continue
		}

		open := nextPunctOfKind(s, index, "(")
		if open < 0 {
			continue
		}
		paramsEnd := s.MatchForward(open)
		if paramsEnd < 0 {
			continue
		}
		bodyStart := nextPunctOfKind(s, paramsEnd, "{", ";", "=>")
		if bodyStart < 0 {
			continue
		}
		if phpdocHasReturnTypeHint(s, bodyStart) {
			continue
		}
		if !phpdocValidTypeSyntax(returnType) {
			continue
		}
		toks := []token.Token{{Kind: token.Punct, Value: ":"}, {Kind: token.Whitespace, Value: " "}}
		inserts[paramsEnd+1] = append(toks, phpdocTypeTokens(returnType, isNullable)...)
	}
	return phpdocApplyInserts(s, inserts)
}

// phpdocHasReturnTypeHint mirrors hasReturnTypeHint: a ":" right after the ")"
// that precedes the function body marks an existing return type.
func phpdocHasReturnTypeHint(s *tokens.Stream, bodyStart int) bool {
	closeParen := phpdocPrevPunct(s, bodyStart, ")")
	if closeParen < 0 {
		return false
	}
	return isPunctVal(s, sigNext(s, closeParen), ":")
}

func phpdocPrevPunct(s *tokens.Stream, idx int, val string) int {
	for j := idx - 1; j >= 0; j-- {
		if s.At(j).Kind == token.Punct && s.At(j).Value == val {
			return j
		}
	}
	return -1
}
