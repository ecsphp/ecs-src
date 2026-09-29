package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

var memberModifiers = map[string]bool{
	"public": true, "private": true, "protected": true, "static": true,
	"abstract": true, "final": true, "readonly": true, "var": true,
}

var visibilityModifiers = map[string]bool{
	"public": true, "private": true, "protected": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/VisibilityRequiredFixer.php
//
// VisibilityRequired adds an explicit "public" to class methods, properties and
// constants that declare no visibility. "var $x" becomes "public $x". Trait use
// and enum cases are left alone.
type VisibilityRequired struct{}

func (VisibilityRequired) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\VisibilityRequiredFixer`
}

func (VisibilityRequired) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/VisibilityRequiredFixer.php"
}

func (VisibilityRequired) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "{" {
			continue
		}
		if kind, _ := classifyBrace(s, i); kind != braceClassLike {
			continue
		}
		// process back to front so insertions do not shift earlier starts
		for _, m := range slices.Backward(classMemberStarts(s, i)) {
			if addVisibility(s, m) {
				changed = true
			}
		}
	}
	return changed
}

func addVisibility(s *tokens.Stream, m int) bool {
	hasVisibility := false
	varIdx := -1
	for k := m; k < s.Len(); {
		t := s.At(k)
		if t.Kind != token.Keyword {
			break // a type, "?", or "$var": this is a property declaration
		}
		lw := strings.ToLower(t.Value)
		if lw == "use" || lw == "case" {
			return false // trait use / enum case - not a visibility target
		}
		if !memberModifiers[lw] {
			break // "function" or "const": the member declaration begins here
		}
		if visibilityModifiers[lw] {
			hasVisibility = true
		}
		if lw == "var" {
			varIdx = k
		}
		k = skipWhitespace(s, k+1)
	}
	if hasVisibility {
		return false
	}
	if varIdx >= 0 {
		s.SetValue(varIdx, "public")
		return true
	}
	s.InsertAt(m, token.Token{Kind: token.Keyword, Value: "public"})
	s.InsertAt(m+1, token.Token{Kind: token.Whitespace, Value: " "})
	return true
}
