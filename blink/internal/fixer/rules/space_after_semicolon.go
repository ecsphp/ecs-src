package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/SpaceAfterSemicolonFixer.php
//
// SpaceAfterSemicolon inserts a single space after a semicolon when it is
// directly followed by code on the same line: "$a=1;$b=2;" becomes
// "$a=1; $b=2;". A ";" before ")" (empty for loop) or a newline is left alone.
// With RemoveInEmptyFor set, the whitespace inside an empty for expression
// ("for ($i = 0; ; ++$i)") is removed instead ("for ($i = 0;; ++$i)"), as is
// the space before the closing parenthesis ("for (;; )").
type SpaceAfterSemicolon struct {
	RemoveInEmptyFor bool
}

func (SpaceAfterSemicolon) Name() string {
	return `PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`
}

func (SpaceAfterSemicolon) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/SpaceAfterSemicolonFixer.php"
}

func (f SpaceAfterSemicolon) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["remove_in_empty_for_expressions"].(bool); ok {
		f.RemoveInEmptyFor = v
	}
	return f
}

func (f SpaceAfterSemicolon) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != ";" {
			continue
		}
		if i+1 >= s.Len() {
			continue
		}
		next := s.At(i + 1)
		if f.RemoveInEmptyFor && next.Kind == token.Whitespace && !hasNewline(next.Value) &&
			i+2 < s.Len() && s.At(i+2).Kind == token.Punct && (s.At(i+2).Value == ";" || s.At(i+2).Value == ")") &&
			spaceAfterSemicolonInForHeader(s, i) {
			s.RemoveAt(i + 1)
			changed = true
			continue
		}
		if next.Kind == token.Whitespace || next.Kind == token.CloseTag {
			continue
		}
		// a space is still inserted before a following ";" (empty for condition:
		// "for (;; )" -> "for (; ; )"); only ")" suppresses it
		if next.Kind == token.Punct && next.Value == ")" {
			continue
		}
		s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
		i++ // skip inserted whitespace
	}
	return changed
}

// spaceAfterSemicolonInForHeader reports whether the ";" at i sits directly in
// the parentheses of a for( ... ) header.
func spaceAfterSemicolonInForHeader(s *tokens.Stream, i int) bool {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")":
			depth++
		case "(":
			if depth > 0 {
				depth--
				continue
			}
			prev, ok := prevSignificant(s, j)
			return ok && prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "for")
		case ";", "{", "}":
			if depth == 0 && t.Value != ";" {
				return false
			}
		}
	}
	return false
}
