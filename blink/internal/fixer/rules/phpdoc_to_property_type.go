package rules

import (
	"maps"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/PhpdocToPropertyTypeFixer.php
//
// PhpdocToPropertyType adds property type declarations taken from "@var"
// annotations (risky).
type PhpdocToPropertyType struct{ phpdocToType }

func (PhpdocToPropertyType) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\PhpdocToPropertyTypeFixer`
}

func (PhpdocToPropertyType) SourceURL() string {
	return fixer.SourceURLFor(PhpdocToPropertyType{}.Name())
}

func (f PhpdocToPropertyType) WithConfig(config map[string]any) fixer.Fixer {
	f.phpdocToType = f.withConfig(config)
	return f
}

func (PhpdocToPropertyType) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasKind(token.DocComment)
}

const ptprTemplate = "<?php class A { private %s $b; }"

func ptprIsSkipped(t string) bool { return t == "resource" || t == "null" }

func (f PhpdocToPropertyType) Fix(s *tokens.Stream) bool {
	inserts := map[int][]token.Token{}
	var typesToExclude []string
	for index := 0; index < s.Len(); index++ {
		t := s.At(index)
		if t.Kind == token.DocComment {
			typesToExclude = append(typesToExclude, ptdTypesToExclude(t.Value)...)
			continue
		}
		if !kwIs(t, "class") && !kwIs(t, "trait") {
			continue
		}
		if prev := sigPrev(s, index); isPunctVal(s, prev, "::") || isPunctVal(s, prev, "->") || isPunctVal(s, prev, "?->") {
			continue
		}
		for i, ts := range f.fixClass(s, index, typesToExclude) {
			if _, ok := inserts[i]; !ok {
				inserts[i] = ts
			}
		}
	}
	return ptdInsertSlices(s, inserts)
}

func (f PhpdocToPropertyType) fixClass(s *tokens.Stream, index int, typesToExclude []string) map[int][]token.Token {
	inserts := map[int][]token.Token{}
	index = nextPunctOfKind(s, index, "{")
	if index < 0 {
		return inserts
	}
	classEnd := s.MatchForward(index)
	if classEnd < 0 {
		return inserts
	}
	for ; index < classEnd; index++ {
		if kwIs(s.At(index), "function") {
			index = nextPunctOfKind(s, index, "{", ";")
			if index < 0 {
				return inserts
			}
			if isPunctVal(s, index, "{") {
				if index = s.MatchForward(index); index < 0 {
					return inserts
				}
			}
			continue
		}
		if s.At(index).Kind != token.DocComment {
			continue
		}
		names, indexes := ptprUntypedProperties(s, index)
		if len(names) == 0 {
			continue
		}
		info, ok := f.resolveApplicableType(names, ptdAnnotationsOfType(s, "var", index))
		if !ok {
			continue
		}
		if slices.Contains([]string{"callable", "never", "void"}, info.commonType) || slices.Contains(typesToExclude, info.commonType) {
			continue
		}
		typeTokens, ok := ptdTypeTokens(ptprTemplate, info.commonType)
		if !ok {
			continue
		}
		inserts[indexes[names[0]]] = append(
			ptdTypeDeclarationTokens(typeTokens, info.commonType, info.isNullable),
			token.Token{Kind: token.Whitespace, Value: " "},
		)
		index = slices.Max(slices.Collect(maps.Values(indexes))) + 1
	}
	return inserts
}

// ptprUntypedProperties ports findNextUntypedPropertiesDeclaration: the
// properties declared right after the docblock at index, in order, when
// they have no type.
func ptprUntypedProperties(s *tokens.Stream, index int) ([]string, map[string]int) {
	for {
		index = sigNext(s, index)
		if index < 0 {
			return nil, nil
		}
		t := s.At(index)
		if t.Kind != token.Keyword || !slices.Contains([]string{"private", "protected", "public", "static", "var"}, strings.ToLower(t.Value)) {
			break
		}
	}
	if s.At(index).Kind != token.Variable {
		return nil, nil
	}
	var names []string
	indexes := map[string]int{}
	for index >= 0 && !isPunctVal(s, index, ";") {
		if t := s.At(index); t.Kind == token.Variable {
			if _, seen := indexes[t.Value]; !seen {
				names = append(names, t.Value)
			}
			indexes[t.Value] = index
		}
		index = sigNext(s, index)
	}
	return names, indexes
}

// resolveApplicableType ports resolveApplicableType: every property gets a
// type from the annotations, and they all agree.
func (f PhpdocToPropertyType) resolveApplicableType(names []string, annotations []ptdAnnotation) (ptdTypeInfo, bool) {
	propertyTypes := map[string]ptdTypeInfo{}
	var order []string
	for _, annotation := range annotations {
		name, ok := annotation.variableName()
		if !ok {
			if len(names) != 1 {
				continue
			}
			name = names[0]
		}
		if !slices.Contains(names, name) {
			continue
		}
		expr, ok := annotation.typeExpression()
		if !ok {
			continue
		}
		info, ok := f.resolveType(expr, false, ptprIsSkipped)
		if !ok {
			continue
		}
		if prev, seen := propertyTypes[name]; seen {
			if prev != info {
				return ptdTypeInfo{}, false
			}
			continue
		}
		propertyTypes[name] = info
		order = append(order, name)
	}
	if len(propertyTypes) != len(names) {
		return ptdTypeInfo{}, false
	}
	typ := propertyTypes[order[0]]
	for _, name := range order[1:] {
		if propertyTypes[name] != typ {
			return ptdTypeInfo{}, false
		}
	}
	return typ, true
}
