package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/PhpdocToParamTypeFixer.php
//
// PhpdocToParamType takes a `@param` annotation of a non-mixed type and adds the
// matching native type hint to the parameter. Risky/experimental.
type PhpdocToParamType struct {
	phpdocTypeOptions
}

func (f PhpdocToParamType) WithConfig(config map[string]any) fixer.Fixer {
	f.phpdocTypeOptions = f.withConfig(config)
	return f
}

func (PhpdocToParamType) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\PhpdocToParamTypeFixer`
}

func (PhpdocToParamType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/PhpdocToParamTypeFixer.php"
}

func (f PhpdocToParamType) Fix(s *tokens.Stream) bool {
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
			case "__clone", "__destruct":
				continue
			}
		}
		docIndex := phpdocFindFunctionDocComment(s, index)
		if docIndex < 0 {
			continue
		}
		open := nextPunctOfKind(s, index, "(")
		if open < 0 {
			continue
		}
		for _, ann := range phpdocAnnotationsOfType(s, docIndex, "param") {
			if !ann.hasTypes {
				continue
			}
			members, glue := docTypeMembers(ann.typesContent)

			paramType, isNullable, ok := phpdocResolveType(members, glue, false, phpdocParamSkipped, cfg, hasNamespace)
			if !ok {
				continue
			}
			if slices.Contains(excludes, paramType) {
				continue
			}

			variableIndex := phpdocFindVariable(s, open, ann.variableName)
			if variableIndex < 0 {
				continue
			}
			if byRef := sigPrev(s, variableIndex); isPunctVal(s, byRef, "&") {
				variableIndex = byRef
			}
			if phpdocHasParamTypeHint(s, variableIndex) {
				continue
			}
			if !phpdocValidTypeSyntax(paramType) {
				continue
			}
			inserts[variableIndex] = append(phpdocTypeTokens(paramType, isNullable), token.Token{Kind: token.Whitespace, Value: " "})
		}
	}
	return phpdocApplyInserts(s, inserts)
}

// phpdocResolveType returns the native type a type expression maps onto: the
// common type, else a union type; ok is false when neither applies.
func phpdocResolveType(members []string, glue string, isReturnType bool, isSkipped func(string) bool, cfg phpdocTypeConfig, hasNamespace bool) (string, bool, bool) {
	if info, ok := phpdocCommonTypeInfo(members, isReturnType, isSkipped, cfg, hasNamespace); ok {
		return info.commonType, info.isNullable, true
	}
	if union, ok := phpdocUnionTypes(members, glue, isReturnType, isSkipped, cfg, hasNamespace); ok {
		return union, false, true
	}
	return "", false, false
}

// phpdocFindVariable mirrors findCorrectVariable: the "$name" parameter between
// the "(" at open and its matching ")", or -1.
func phpdocFindVariable(s *tokens.Stream, open int, name string) int {
	if name == "" {
		return -1
	}
	end := s.MatchForward(open)
	if end < 0 {
		return -1
	}
	for i := open + 1; i < end; i++ {
		if s.At(i).Kind == token.Variable && s.At(i).Value == name {
			return i
		}
	}
	return -1
}

// phpdocHasParamTypeHint mirrors hasParamTypeHint: a parameter already has a
// type unless the token before it is "," or "(".
func phpdocHasParamTypeHint(s *tokens.Stream, index int) bool {
	prev := sigPrev(s, index)
	return !isPunctVal(s, prev, ",") && !isPunctVal(s, prev, "(")
}
