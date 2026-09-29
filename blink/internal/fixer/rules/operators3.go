package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/ObjectOperatorWithoutWhitespaceFixer.php
//
// ObjectOperatorWithoutWhitespace removes whitespace around "->" and "?->"
// ("$a -> b" -> "$a->b").
type ObjectOperatorWithoutWhitespace struct{}

func (ObjectOperatorWithoutWhitespace) Name() string {
	return `PhpCsFixer\Fixer\Operator\ObjectOperatorWithoutWhitespaceFixer`
}

func (ObjectOperatorWithoutWhitespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/ObjectOperatorWithoutWhitespaceFixer.php"
}

func (ObjectOperatorWithoutWhitespace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || (t.Value != "->" && t.Value != "?->") {
			continue
		}
		if i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) {
			s.RemoveAt(i + 1)
			changed = true
		}
		if i > 0 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
			s.RemoveAt(i - 1)
			i--
			changed = true
		}
	}
	return changed
}
