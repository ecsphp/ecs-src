package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/MultilineWhitespaceBeforeSemicolonsFixer.php
//
// MultilineWhitespaceBeforeSemicolons removes multi-line whitespace before a
// semicolon (the default "no_multi_line" strategy): "$a = foo()\n    ;" becomes
// "$a = foo();". A "const" statement's own semicolon is left alone, and a
// comment sitting between the code and the ";" keeps the ";" attached to the
// code, moved ahead of the comment.
type MultilineWhitespaceBeforeSemicolons struct{}

func (MultilineWhitespaceBeforeSemicolons) Name() string {
	return `PhpCsFixer\Fixer\Semicolon\MultilineWhitespaceBeforeSemicolonsFixer`
}

func (MultilineWhitespaceBeforeSemicolons) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/MultilineWhitespaceBeforeSemicolonsFixer.php"
}

func (MultilineWhitespaceBeforeSemicolons) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind == token.Keyword && strings.ToLower(t.Value) == "const" {
			// skip a const statement entirely, up to its terminating ";"
			for j := i + 1; j < s.Len(); j++ {
				if s.At(j).Kind == token.Punct && s.At(j).Value == ";" {
					i = j
					break
				}
			}
			continue
		}
		if t.Kind != token.Punct || t.Value != ";" {
			continue
		}
		prevIdx := i - 1
		if prevIdx < 0 || s.At(prevIdx).Kind != token.Whitespace || !hasNewline(s.At(prevIdx).Value) {
			continue
		}
		// a comment between the code and the ";": keep the ";" with the code by
		// moving it ahead of the comment
		if i-2 >= 0 && isComment(s.At(i-2)) && startsWithNewline(s.At(prevIdx).Value) {
			sig := prevValueTokenIndex(s, i)
			s.RemoveAt(i)       // drop the ";"
			s.RemoveAt(prevIdx) // drop the whitespace
			s.InsertAt(sig+1, token.Token{Kind: token.Punct, Value: ";"})
			changed = true
			continue
		}
		// plain case: drop the multi-line whitespace so ";" follows the code
		s.RemoveAt(prevIdx)
		i--
		changed = true
	}
	return changed
}

func startsWithNewline(v string) bool {
	return strings.HasPrefix(v, "\n") || strings.HasPrefix(v, "\r")
}

// prevValueTokenIndex scans back from a ";" for the token the ";" belongs to,
// stopping at a number, name, variable, string or ")" (PHP-CS-Fixer's
// getPreviousSignificantTokenIndex).
func prevValueTokenIndex(s *tokens.Stream, i int) int {
	for j := i - 1; j > 0; j-- {
		t := s.At(j)
		if t.Kind == token.Number || t.Kind == token.Ident || t.Kind == token.Variable ||
			t.Kind == token.String || (t.Kind == token.Punct && t.Value == ")") {
			return j
		}
	}
	return i
}
