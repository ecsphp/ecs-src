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
// PhpdocToParamType adds parameter type declarations taken from "@param"
// annotations (risky).
type PhpdocToParamType struct{ phpdocToType }

func (PhpdocToParamType) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\PhpdocToParamTypeFixer`
}

func (PhpdocToParamType) SourceURL() string {
	return fixer.SourceURLFor(PhpdocToParamType{}.Name())
}

func (f PhpdocToParamType) WithConfig(config map[string]any) fixer.Fixer {
	f.phpdocToType = f.withConfig(config)
	return f
}

func (PhpdocToParamType) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasAnyWord("function", "fn")
}

const ptpTemplate = "<?php function f(%s $x) {}"

func ptpIsSkipped(t string) bool { return t == "resource" || t == "static" || t == "void" }

func (f PhpdocToParamType) Fix(s *tokens.Stream) bool {
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
			case "__clone", "__destruct":
				continue
			}
		}
		doc := ptdFindFunctionDocComment(s, index)
		if doc < 0 {
			continue
		}
		for _, annotation := range ptdAnnotationsOfType(s, "param", doc) {
			expr, ok := annotation.typeExpression()
			if !ok {
				continue
			}
			info, ok := f.resolveType(expr, false, ptpIsSkipped)
			if !ok || slices.Contains(typesToExclude, info.commonType) {
				continue
			}
			open := nextPunctOfKind(s, index, "(")
			if open < 0 {
				continue
			}
			variable := ptpFindVariable(s, open, annotation)
			if variable < 0 {
				continue
			}
			if byRef := sigPrev(s, variable); isPunctVal(s, byRef, "&") {
				variable = byRef
			}
			if prev := sigPrev(s, variable); !isPunctVal(s, prev, ",") && !isPunctVal(s, prev, "(") {
				continue // has a type already
			}
			typeTokens, ok := ptdTypeTokens(ptpTemplate, info.commonType)
			if !ok {
				continue
			}
			inserts[variable] = append(
				ptdTypeDeclarationTokens(typeTokens, info.commonType, info.isNullable),
				token.Token{Kind: token.Whitespace, Value: " "},
			)
		}
	}
	return ptdInsertSlices(s, inserts)
}

// ptpFindVariable ports findCorrectVariable.
func ptpFindVariable(s *tokens.Stream, open int, annotation ptdAnnotation) int {
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return -1
	}
	name, hasName := annotation.variableName()
	for i := open + 1; i < closeIdx; i++ {
		if s.At(i).Kind == token.Variable && hasName && s.At(i).Value == name {
			return i
		}
	}
	return -1
}
