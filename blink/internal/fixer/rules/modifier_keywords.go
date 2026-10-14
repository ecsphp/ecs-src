package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ModifierKeywordsFixer.php
//
// ModifierKeywords gives class constants, properties and methods an explicit
// visibility and orders the modifier keywords canonically:
// inheritance (abstract/final), visibility, scope (static), mutation (readonly),
// then the type declaration and name.
//
// This handles class-body const/property/method members. Constructor promoted
// properties and PHP 8.4 asymmetric set-visibility (public(set)) are left
// untouched; members whose modifier run contains a comment are skipped.
//
// The `elements` option limits which member kinds (const, property, method) are
// fixed; nil fixes all three, matching the PHP-CS-Fixer default.
type ModifierKeywords struct {
	// elements is the set of enabled member kinds; nil means all three.
	elements map[string]bool
}

func (f ModifierKeywords) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := arrayNotationStringList(config["elements"]); ok {
		set := make(map[string]bool, len(list))
		for _, item := range list {
			set[item] = true
		}
		f.elements = set
	}
	return f
}

func (f ModifierKeywords) enabled(kind string) bool {
	if f.elements == nil {
		return true
	}
	return f.elements[kind]
}

func (ModifierKeywords) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ModifierKeywordsFixer`
}

func (ModifierKeywords) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ModifierKeywordsFixer.php"
}

func (f ModifierKeywords) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		if kind, _ := classifyBrace(s, i); kind != braceClassLike {
			continue
		}
		for _, m := range slices.Backward(classMemberStarts(s, i)) {
			if f.orderModifierKeywords(s, m) {
				changed = true
			}
		}
	}
	return changed
}

// modifierMemberKind classifies the member whose modifier run ends at index
// after: "const", "method" (a function) or "property".
func modifierMemberKind(s *tokens.Stream, after int) string {
	if after < s.Len() && s.At(after).Kind == token.Keyword {
		switch strings.ToLower(s.At(after).Value) {
		case "const":
			return "const"
		case "function":
			return "method"
		}
	}
	return "property"
}

func (f ModifierKeywords) orderModifierKeywords(s *tokens.Stream, m int) bool {
	var absFinal []string
	visibility := ""
	static := ""
	readOnly := ""
	hasVisibility := false

	k := m
	for k < s.Len() {
		t := s.At(k)
		if t.Kind == token.Whitespace {
			k++
			continue
		}
		if t.Kind == token.Comment || t.Kind == token.DocComment {
			return false // cannot safely reorder across a comment
		}
		if t.Kind != token.Keyword {
			break
		}
		lw := strings.ToLower(t.Value)
		if lw == "use" || lw == "case" {
			return false // trait use / enum case is not a member with modifiers
		}
		if !memberModifiers[lw] {
			break // "const", "function" or a type keyword: modifier run ends
		}
		// set-visibility "public(set)" is not representable on the flat lexer
		if next := skipWhitespace(s, k+1); next < s.Len() && s.At(next).Kind == token.Punct && s.At(next).Value == "(" {
			return false
		}
		switch {
		case lw == "abstract" || lw == "final":
			absFinal = append(absFinal, t.Value)
		case lw == "static":
			static = t.Value
		case lw == "readonly":
			readOnly = t.Value
		case lw == "var":
			visibility = "public"
			hasVisibility = true
		case visibilityModifiers[lw]:
			visibility = t.Value
			hasVisibility = true
		}
		k++
	}
	after := k
	if after < s.Len() && (s.At(after).Kind == token.Comment || s.At(after).Kind == token.DocComment) {
		return false
	}
	if !f.enabled(modifierMemberKind(s, after)) {
		return false
	}
	if visibility == "" {
		visibility = "public"
	}

	ordered := make([]string, 0, len(absFinal)+3)
	ordered = append(ordered, absFinal...)
	ordered = append(ordered, visibility)
	if static != "" {
		ordered = append(ordered, static)
	}
	if readOnly != "" {
		ordered = append(ordered, readOnly)
	}

	// desired rendering: modifiers single-spaced, one trailing space before the type/name
	var want strings.Builder
	for _, mod := range ordered {
		want.WriteString(mod)
		want.WriteString(" ")
	}

	// current rendering of the run [m, after)
	var cur strings.Builder
	for x := m; x < after; x++ {
		cur.WriteString(s.At(x).Value)
	}
	if hasVisibility && cur.String() == want.String() {
		return false
	}

	repl := make([]token.Token, 0, len(ordered)*2)
	for _, mod := range ordered {
		repl = append(repl, token.Token{Kind: token.Keyword, Value: mod})
		repl = append(repl, token.Token{Kind: token.Whitespace, Value: " "})
	}
	if after == m {
		s.InsertSliceAt(m, repl)
		return true
	}
	s.ReplaceRange(m, after-1, repl)
	return true
}
