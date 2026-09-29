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
		// a keyword-spelled class-constant name is left as written, EXCEPT when the
		// same keyword reappears after :: in that constant's value (e.g.
		// "const int PUBLIC = Modifiers::PUBLIC" -> "public"). PHP-CS-Fixer's
		// LowercaseKeywordsFixer lowercases it only in that case (a transformer
		// misfire); mirroring it keeps blink 1:1 with ECS.
		if isClassConstName(s, i) {
			if !sameKeywordAfterDoubleColon(s, i, lowerASCII(t.Value)) {
				continue
			}
		} else {
			// a keyword-spelled constant name in assignment position (e.g.
			// "const string RETURN = ...") must not be lowercased
			if nextSignificantValue(s, i) == "=" {
				continue
			}
			// a keyword-spelled class/member name (e.g. "Enum" in "extends Enum",
			// "Spatie\Enum\Enum", "class Enum", or a named argument "instanceOf:") is
			// an identifier, not a keyword
			if keywordUsedAsIdentifier(s, i) {
				continue
			}
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

// keywordUsedAsIdentifier reports whether a keyword-spelled token is actually a
// class or member name (the lexer marks context-sensitive words like "enum" as
// keywords even when used as identifiers).
func keywordUsedAsIdentifier(s *tokens.Stream, i int) bool {
	if prev, ok := prevSignificant(s, i); ok {
		if prev.Kind == token.Punct {
			switch prev.Value {
			case `\`, "->", "?->", "::":
				return true
			}
		}
		if prev.Kind == token.Keyword {
			switch strings.ToLower(prev.Value) {
			case "extends", "implements", "new", "instanceof",
				"class", "interface", "trait", "enum", "function", "const",
				"namespace", "use", "as", "goto", "insteadof":
				return true
			}
		}
	}
	if n := nextSignificantIndex(s, i); n >= 0 && s.At(n).Kind == token.Punct {
		switch s.At(n).Value {
		case `\`, "::", ":":
			return true
		}
	}
	return false
}

// isClassConstName reports whether token i is the name in a class-constant
// declaration - "const NAME" or a typed "const <type> NAME".
func isClassConstName(s *tokens.Stream, i int) bool {
	j := prevSignificantIndex(s, i)
	for steps := 0; j >= 0 && steps < 8; steps++ {
		prev := s.At(j)
		if prev.Kind == token.Keyword && strings.ToLower(prev.Value) == "const" {
			return true
		}
		// walk back over the type between "const" and the name
		if prev.Kind == token.Ident || prev.Kind == token.Keyword ||
			(prev.Kind == token.Punct && (prev.Value == "?" || prev.Value == "|" || prev.Value == "&" || prev.Value == `\`)) {
			j = prevSignificantIndex(s, j)
			continue
		}
		return false
	}
	return false
}

// sameKeywordAfterDoubleColon reports whether the same keyword (name, already
// lowercased) reappears after :: within the constant's value, up to the ; or {
// that ends the declaration.
func sameKeywordAfterDoubleColon(s *tokens.Stream, i int, name string) bool {
	for j := i + 1; j < s.Len(); j++ {
		v := s.At(j)
		if v.Kind == token.Punct && (v.Value == ";" || v.Value == "{") {
			return false
		}
		if v.Kind == token.Punct && v.Value == "::" {
			if n := nextSignificantIndex(s, j); n >= 0 && lowerASCII(s.At(n).Value) == name {
				return true
			}
		}
	}
	return false
}
