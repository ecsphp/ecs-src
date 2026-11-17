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
// PhpdocToReturnType adds a return type declaration taken from a single
// "@return" annotation (risky).
type PhpdocToReturnType struct{ phpdocToType }

func (PhpdocToReturnType) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\PhpdocToReturnTypeFixer`
}

func (PhpdocToReturnType) SourceURL() string {
	return fixer.SourceURLFor(PhpdocToReturnType{}.Name())
}

func (f PhpdocToReturnType) WithConfig(config map[string]any) fixer.Fixer {
	f.phpdocToType = f.withConfig(config)
	return f
}

func (PhpdocToReturnType) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasAnyWord("function", "fn")
}

const ptrTemplate = "<?php function f(): %s {}"

func ptrIsSkipped(t string) bool { return t == "resource" || t == "null" }

func (f PhpdocToReturnType) Fix(s *tokens.Stream) bool {
	inserts := map[int][]token.Token{}
	var typesToExclude []string
	for index := 0; index < s.Len(); index++ {
		t := s.At(index)
		if t.Kind == token.DocComment {
			typesToExclude = append(typesToExclude, ptdTypesToExclude(t.Value)...)
			continue
		}
		if !kwIs(t, "function") && !kwIs(t, "fn") {
			continue
		}
		if name := sigNext(s, index); name >= 0 && s.At(name).Kind == token.Ident {
			switch strings.ToLower(s.At(name).Value) {
			case "__construct", "__destruct", "__clone":
				continue
			}
		}
		doc := ptdFindFunctionDocComment(s, index)
		if doc < 0 {
			continue
		}
		annotations := ptdAnnotationsOfType(s, "return", doc)
		if len(annotations) != 1 {
			continue
		}
		expr, ok := annotations[0].typeExpression()
		if !ok {
			continue
		}
		info, ok := f.resolveType(expr, true, ptrIsSkipped)
		if !ok || slices.Contains(typesToExclude, info.commonType) {
			continue
		}
		paramsStart := nextPunctOfKind(s, index, "(")
		if paramsStart < 0 {
			continue
		}
		paramsEnd := s.MatchForward(paramsStart)
		if paramsEnd < 0 || ptrHasReturnTypeHint(s, paramsEnd) {
			continue
		}
		typeTokens, ok := ptdTypeTokens(ptrTemplate, info.commonType)
		if !ok {
			continue
		}
		inserts[paramsEnd+1] = append([]token.Token{
			{Kind: token.Punct, Value: ":"},
			{Kind: token.Whitespace, Value: " "},
		}, ptdTypeDeclarationTokens(typeTokens, info.commonType, info.isNullable)...)
	}
	return ptdInsertSlices(s, inserts)
}

// ptrHasReturnTypeHint ports hasReturnTypeHint: the last ")" before the body
// (a closure's "use (...)" one when present) is followed by a ":".
func ptrHasReturnTypeHint(s *tokens.Stream, paramsEnd int) bool {
	end := paramsEnd
	if use := sigNext(s, end); kwIs(s.At(max(use, 0)), "use") {
		if open := sigNext(s, use); isPunctVal(s, open, "(") {
			if closeIdx := s.MatchForward(open); closeIdx >= 0 {
				end = closeIdx
			}
		}
	}
	return isPunctVal(s, sigNext(s, end), ":")
}
