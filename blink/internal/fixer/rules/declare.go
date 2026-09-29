package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/DeclareEqualNormalizeFixer.php
//
// DeclareEqualNormalize removes the spaces around "=" inside a declare header:
// "declare(strict_types = 1)" -> "declare(strict_types=1)".
type DeclareEqualNormalize struct{}

func (DeclareEqualNormalize) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\DeclareEqualNormalizeFixer`
}

func (DeclareEqualNormalize) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/DeclareEqualNormalizeFixer.php"
}

func (DeclareEqualNormalize) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || t.Value != "=" || !insideDeclareArgs(s, i) {
			continue
		}
		if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace {
			s.RemoveAt(i + 1)
			changed = true
		}
		if i >= 1 && s.At(i-1).Kind == token.Whitespace {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
	}
	return changed
}
