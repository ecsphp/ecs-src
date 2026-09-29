package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
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
//
// elements mirrors the "elements" option (subset of property/method/const); a
// nil value (the zero value) means all three, matching the default.
type VisibilityRequired struct {
	elements []string
}

func (f VisibilityRequired) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["elements"]; ok {
		f.elements = visibilityConfigStrings(raw)
	}
	return f
}

// visibilityConfigStrings converts a config list ([]any or []string) to []string.
func visibilityConfigStrings(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if sv, ok := e.(string); ok {
				out = append(out, sv)
			}
		}
		return out
	}
	return nil
}

// visibilityEnabled reports whether the given member kind is targeted. An empty
// kind (unknown member) is always processed to preserve default behavior.
func (f VisibilityRequired) visibilityEnabled(kind string) bool {
	if kind == "" || f.elements == nil {
		return true
	}
	return slices.Contains(f.elements, kind)
}

func (VisibilityRequired) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\VisibilityRequiredFixer`
}

func (VisibilityRequired) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/VisibilityRequiredFixer.php"
}

func (f VisibilityRequired) Fix(s *tokens.Stream) bool {
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
			if f.addVisibility(s, m) {
				changed = true
			}
		}
	}
	return changed
}

func (f VisibilityRequired) addVisibility(s *tokens.Stream, m int) bool {
	hasVisibility := false
	varIdx := -1
	kind := "property"
	for k := m; k < s.Len(); {
		t := s.At(k)
		if t.Kind != token.Keyword {
			kind = "property" // a type, "?", or "$var": this is a property declaration
			break
		}
		lw := strings.ToLower(t.Value)
		if lw == "use" || lw == "case" {
			return false // trait use / enum case - not a visibility target
		}
		if !memberModifiers[lw] {
			switch lw {
			case "function":
				kind = "method"
			case "const":
				kind = "const"
			default:
				kind = "" // unknown member start: keep default handling
			}
			break
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
	if !f.visibilityEnabled(kind) {
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
