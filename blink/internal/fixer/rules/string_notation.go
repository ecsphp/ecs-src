package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/SingleQuoteFixer.php
//
// SingleQuote converts a double-quoted string to single quotes when it is safe:
// no variables ($), no escape sequences (\), and no single quote in the content.
type SingleQuote struct{}

func (SingleQuote) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\SingleQuoteFixer`
}

func (SingleQuote) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/SingleQuoteFixer.php"
}

func (SingleQuote) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.String || len(t.Value) < 2 {
			continue
		}
		if t.Value[0] != '"' || t.Value[len(t.Value)-1] != '"' {
			continue // not a plain double-quoted string (e.g. heredoc)
		}
		content := t.Value[1 : len(t.Value)-1]
		if strings.ContainsAny(content, "$\\'") {
			continue // interpolation, escape, or a quote that would need escaping
		}
		s.SetValue(i, "'"+content+"'")
		changed = true
	}
	return changed
}
