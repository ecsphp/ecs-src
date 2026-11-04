package rules

import (
	"regexp"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/OctalNotationFixer.php
//
// OctalNotation rewrites a legacy octal literal to the explicit 0o notation
// ("0755" -> "0o755"). PHP 8.1+ only, which this project requires.
type OctalNotation struct{}

func (OctalNotation) Name() string {
	return `PhpCsFixer\Fixer\Basic\OctalNotationFixer`
}

func (OctalNotation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/OctalNotationFixer.php"
}

var legacyOctal = regexp.MustCompile(`^0_*([0-7_]+)$`)

func (OctalNotation) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Number {
			continue
		}
		v := s.At(i).Value
		if nv := legacyOctal.ReplaceAllString(v, "0o$1"); nv != v {
			s.SetValue(i, nv)
			changed = true
		}
	}
	return changed
}
