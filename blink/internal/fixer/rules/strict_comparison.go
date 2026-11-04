package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Strict/StrictComparisonFixer.php
//
// StrictComparison rewrites the loose comparison operators to their strict
// counterparts: "==" to "===" and both "!=" and "<>" (the same T_IS_NOT_EQUAL
// token in PHP) to "!==". Risky: it can change behaviour.
type StrictComparison struct{}

func (StrictComparison) Name() string {
	return `PhpCsFixer\Fixer\Strict\StrictComparisonFixer`
}

func (StrictComparison) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Strict/StrictComparisonFixer.php"
}

func (StrictComparison) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct {
			continue
		}
		switch s.At(i).Value {
		case "==":
			s.SetValue(i, "===")
			changed = true
		case "!=", "<>":
			s.SetValue(i, "!==")
			changed = true
		}
	}
	return changed
}
