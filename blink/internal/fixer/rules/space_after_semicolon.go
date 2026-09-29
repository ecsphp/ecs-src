package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/SpaceAfterSemicolonFixer.php
//
// SpaceAfterSemicolon inserts a single space after a semicolon when it is
// directly followed by code on the same line: "$a=1;$b=2;" becomes
// "$a=1; $b=2;". A ";" before ")" (empty for loop) or a newline is left alone.
type SpaceAfterSemicolon struct{}

func (SpaceAfterSemicolon) Name() string {
	return `PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer`
}

func (SpaceAfterSemicolon) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/SpaceAfterSemicolonFixer.php"
}

func (SpaceAfterSemicolon) Fix(s *tokens.Stream) bool {
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
		if next.Kind == token.Whitespace || next.Kind == token.CloseTag {
			continue
		}
		if next.Kind == token.Punct && (next.Value == ")" || next.Value == ";") {
			continue
		}
		s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
		i++ // skip inserted whitespace
	}
	return changed
}
