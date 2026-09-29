package rules

import (
	"slices"
	"strings"

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
type ModifierKeywords struct{}

func (ModifierKeywords) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\ModifierKeywordsFixer`
}

func (ModifierKeywords) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/ModifierKeywordsFixer.php"
}

func (ModifierKeywords) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		if kind, _ := classifyBrace(s, i); kind != braceClassLike {
			continue
		}
		for _, m := range slices.Backward(classMemberStarts(s, i)) {
			if orderModifierKeywords(s, m) {
				changed = true
			}
		}
	}
	return changed
}

func orderModifierKeywords(s *tokens.Stream, m int) bool {
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
		for _, tk := range slices.Backward(repl) {
			s.InsertAt(m, tk)
		}
		return true
	}
	s.ReplaceRange(m, after-1, repl)
	return true
}
