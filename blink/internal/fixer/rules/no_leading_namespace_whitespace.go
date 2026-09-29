package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/NoLeadingNamespaceWhitespaceFixer.php
//
// NoLeadingNamespaceWhitespace removes indentation before a "namespace"
// declaration: "  namespace App;" becomes "namespace App;".
type NoLeadingNamespaceWhitespace struct{}

func (NoLeadingNamespaceWhitespace) Name() string {
	return `PhpCsFixer\Fixer\NamespaceNotation\NoLeadingNamespaceWhitespaceFixer`
}

func (NoLeadingNamespaceWhitespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/NamespaceNotation/NoLeadingNamespaceWhitespaceFixer.php"
}

func (NoLeadingNamespaceWhitespace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 1; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || t.Value != "namespace" {
			continue
		}
		prev := s.At(i - 1)
		if prev.Kind != token.Whitespace {
			continue
		}
		idx := strings.LastIndexByte(prev.Value, '\n')
		if idx < 0 {
			continue
		}
		trimmed := prev.Value[:idx+1] // drop spaces/tabs after the last newline
		if trimmed != prev.Value {
			s.SetValue(i-1, trimmed)
			changed = true
		}
	}
	return changed
}
