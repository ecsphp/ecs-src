package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// lowerASCII lowercases s like strings.ToLower, but returns s unchanged (no
// allocation) when it is already lowercase ASCII - the common case for keywords
// and constants. Non-ASCII or uppercase input falls back to strings.ToLower so
// the result stays byte-identical.
func lowerASCII(s string) string {
	for i := 0; i < len(s); i++ {
		if c := s[i]; (c >= 'A' && c <= 'Z') || c >= 0x80 {
			return strings.ToLower(s)
		}
	}
	return s
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/LowercaseKeywordsFixer.php
//
// LowercaseKeywords lowercases PHP keywords (FUNCTION -> function).
type LowercaseKeywords struct{}

func (LowercaseKeywords) Name() string {
	return `PhpCsFixer\Fixer\Casing\LowercaseKeywordsFixer`
}

func (LowercaseKeywords) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/LowercaseKeywordsFixer.php"
}

func (LowercaseKeywords) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		if lower := lowerASCII(t.Value); lower != t.Value {
			s.SetValue(i, lower)
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/ConstantCaseFixer.php
//
// ConstantCase lowercases the true, false and null constants.
type ConstantCase struct {
	Upper bool
}

func (f ConstantCase) WithConfig(config map[string]any) fixer.Fixer {
	if c, ok := config["case"].(string); ok {
		f.Upper = c == "upper"
	}
	return f
}

func (ConstantCase) Name() string {
	return `PhpCsFixer\Fixer\Casing\ConstantCaseFixer`
}

func (ConstantCase) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/ConstantCaseFixer.php"
}

func (f ConstantCase) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Ident {
			continue
		}
		lower := lowerASCII(t.Value)
		if lower != "true" && lower != "false" && lower != "null" {
			continue
		}
		if memberPrev(s, i) {
			continue // $obj->true, Foo::null - a member name, not the constant
		}
		want := lower
		if f.Upper {
			want = strings.ToUpper(lower)
		}
		if want != t.Value {
			s.SetValue(i, want)
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/LowercaseStaticReferenceFixer.php
//
// LowercaseStaticReference lowercases self, static and parent.
type LowercaseStaticReference struct{}

func (LowercaseStaticReference) Name() string {
	return `PhpCsFixer\Fixer\Casing\LowercaseStaticReferenceFixer`
}

func (LowercaseStaticReference) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/LowercaseStaticReferenceFixer.php"
}

func (LowercaseStaticReference) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Ident && t.Kind != token.Keyword {
			continue
		}
		lower := lowerASCII(t.Value)
		if lower != "self" && lower != "static" && lower != "parent" {
			continue
		}
		// SELF/PARENT/STATIC can be constant names: "ObjectReference::SELF" (member
		// access) or "const string PARENT = ..." (declaration) - not the keyword
		if memberPrev(s, i) || nextSignificantValue(s, i) == "=" {
			continue
		}
		if lower != t.Value {
			s.SetValue(i, lower)
			changed = true
		}
	}
	return changed
}
