package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/PhpdocToPropertyTypeFixer.php
//
// PhpdocToPropertyType takes a `@var` annotation of a non-mixed type and adds
// the matching native type to the property. Risky/experimental.
type PhpdocToPropertyType struct {
	phpdocTypeOptions
}

func (f PhpdocToPropertyType) WithConfig(config map[string]any) fixer.Fixer {
	f.phpdocTypeOptions = f.withConfig(config)
	return f
}

func (PhpdocToPropertyType) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\PhpdocToPropertyTypeFixer`
}

func (PhpdocToPropertyType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/PhpdocToPropertyTypeFixer.php"
}

func phpdocPropertySkipped(t string) bool { return t == "resource" || t == "null" }

func (f PhpdocToPropertyType) Fix(s *tokens.Stream) bool {
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
		if taIsClassKw(s, index) || taKw(s, index, "trait") {
			phpdocFixClass(s, index, excludes, cfg, hasNamespace, inserts)
		}
	}
	return phpdocApplyInserts(s, inserts)
}

func phpdocFixClass(s *tokens.Stream, classIndex int, excludes []string, cfg phpdocTypeConfig, hasNamespace bool, inserts map[int][]token.Token) {
	open := nextPunctOfKind(s, classIndex, "{")
	if open < 0 {
		return
	}
	classEnd := s.MatchForward(open)
	if classEnd < 0 {
		return
	}
	for index := open; index < classEnd; index++ {
		if kwIs(s.At(index), "function") {
			body := nextPunctOfKind(s, index, "{", ";")
			if body < 0 {
				return
			}
			if isPunctVal(s, body, "{") {
				end := s.MatchForward(body)
				if end < 0 {
					return
				}
				index = end
			} else {
				index = body
			}
			continue
		}
		if s.At(index).Kind != token.DocComment {
			continue
		}
		propIndices, order := phpdocUntypedProperties(s, index)
		if len(order) == 0 {
			continue
		}
		info, ok := phpdocResolvePropertyType(order, propIndices, phpdocAnnotationsOfType(s, index, "var"), cfg, hasNamespace)
		if !ok {
			continue
		}
		propertyType := info.commonType
		if propertyType == "callable" || propertyType == "never" || propertyType == "void" {
			continue
		}
		if slices.Contains(excludes, propertyType) {
			continue
		}
		if !phpdocValidTypeSyntax(propertyType) {
			continue
		}
		inserts[propIndices[order[0]]] = append(phpdocTypeTokens(propertyType, info.isNullable), token.Token{Kind: token.Whitespace, Value: " "})

		maxIndex := 0
		for _, i := range propIndices {
			maxIndex = max(maxIndex, i)
		}
		index = maxIndex + 1
	}
}

// phpdocUntypedProperties mirrors findNextUntypedPropertiesDeclaration: the
// variable names (keyed to their indices, plus their declaration order) of an
// untyped property declaration following the docblock, or empty.
func phpdocUntypedProperties(s *tokens.Stream, docIndex int) (map[string]int, []string) {
	index := docIndex
	for {
		index = sigNext(s, index)
		if index < 0 {
			return nil, nil
		}
		if s.At(index).Kind == token.Keyword {
			switch strings.ToLower(s.At(index).Value) {
			case "private", "protected", "public", "static", "var":
				continue
			}
		}
		break
	}
	if s.At(index).Kind != token.Variable {
		return nil, nil
	}
	props := map[string]int{}
	var order []string
	for index >= 0 && !isPunctVal(s, index, ";") {
		if s.At(index).Kind == token.Variable {
			name := s.At(index).Value
			if _, seen := props[name]; !seen {
				order = append(order, name)
			}
			props[name] = index
		}
		index = sigNext(s, index)
	}
	return props, order
}

// phpdocResolvePropertyType mirrors resolveApplicableType: the common type of
// every documented property, or false when they are not all the same.
func phpdocResolvePropertyType(order []string, propIndices map[string]int, anns []phpdocAnnotationInfo, cfg phpdocTypeConfig, hasNamespace bool) (phpdocTypeInfo, bool) {
	propertyTypes := map[string]phpdocTypeInfo{}
	for _, ann := range anns {
		propName := ann.variableName
		if propName == "" {
			if len(order) != 1 {
				continue
			}
			propName = order[0]
		}
		if _, ok := propIndices[propName]; !ok {
			continue
		}
		if !ann.hasTypes {
			continue
		}
		members, glue := docTypeMembers(ann.typesContent)
		commonType, isNullable, ok := phpdocResolveType(members, glue, false, phpdocPropertySkipped, cfg, hasNamespace)
		if !ok {
			continue
		}
		info := phpdocTypeInfo{commonType: commonType, isNullable: isNullable}
		if existing, ok := propertyTypes[propName]; ok && existing != info {
			return phpdocTypeInfo{}, false
		}
		propertyTypes[propName] = info
	}
	if len(propertyTypes) != len(propIndices) {
		return phpdocTypeInfo{}, false
	}
	var first phpdocTypeInfo
	set := false
	for _, info := range propertyTypes {
		if !set {
			first, set = info, true
			continue
		}
		if info != first {
			return phpdocTypeInfo{}, false
		}
	}
	return first, true
}
