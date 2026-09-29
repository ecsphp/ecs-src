package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/ConcatSpaceFixer.php
//
// ConcatSpace forces a single space around the "." concatenation operator
// (spacing: one), as configured in the ECS spaces set: "'a'.'b'" becomes
// "'a' . 'b'". Whitespace across a newline is preserved.
type ConcatSpace struct{}

func (ConcatSpace) Name() string {
	return `PhpCsFixer\Fixer\Operator\ConcatSpaceFixer`
}

func (ConcatSpace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/ConcatSpaceFixer.php"
}

func (ConcatSpace) Fix(s *tokens.Stream) bool {
	return normalizeSpaceAround(s, func(s *tokens.Stream, i int) bool {
		t := s.At(i)
		return t.Kind == token.Punct && t.Value == "."
	})
}
