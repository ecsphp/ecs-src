package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/NoSinglelineWhitespaceBeforeSemicolonsFixer.php
//
// NoSinglelineWhitespaceBeforeSemicolons removes single-line whitespace before
// a semicolon: "$x = 1 ;" becomes "$x = 1;".
type NoSinglelineWhitespaceBeforeSemicolons struct{}

func (NoSinglelineWhitespaceBeforeSemicolons) Name() string {
	return `PhpCsFixer\Fixer\Semicolon\NoSinglelineWhitespaceBeforeSemicolonsFixer`
}

func (NoSinglelineWhitespaceBeforeSemicolons) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/NoSinglelineWhitespaceBeforeSemicolonsFixer.php"
}

func (NoSinglelineWhitespaceBeforeSemicolons) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 1; i-- {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != ";" {
			continue
		}
		prev := s.At(i - 1)
		if prev.Kind == token.Whitespace && !strings.ContainsAny(prev.Value, "\n\r") {
			// keep the space before a ";" that follows another ";" - the empty
			// condition of a for() header ("for ($i = 0; ; ++$i)")
			if p, ok := prevSignificant(s, i-1); ok && p.Kind == token.Punct && p.Value == ";" {
				continue
			}
			s.RemoveAt(i - 1)
			changed = true
		}
	}
	return changed
}
