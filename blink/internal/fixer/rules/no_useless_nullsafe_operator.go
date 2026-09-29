package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NoUselessNullsafeOperatorFixer.php
//
// NoUselessNullsafeOperator turns "$this?->" into "$this->": a nullsafe call on
// $this can never be null, so the operator is useless.
type NoUselessNullsafeOperator struct{}

func (NoUselessNullsafeOperator) Name() string {
	return `PhpCsFixer\Fixer\Operator\NoUselessNullsafeOperatorFixer`
}

func (NoUselessNullsafeOperator) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NoUselessNullsafeOperatorFixer.php"
}

func (NoUselessNullsafeOperator) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "?->" {
			continue
		}
		p := prevSignificantIndex(s, i)
		if p < 0 || s.At(p).Kind != token.Variable || !strings.EqualFold(s.At(p).Value, "$this") {
			continue
		}
		s.SetValue(i, "->")
		changed = true
	}
	return changed
}
