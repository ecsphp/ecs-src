package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/BinaryOperatorSpacesFixer.php
//
// binaryOperators are unambiguously binary operators (never unary), safe to
// space on a flat token stream. Ambiguous ones (+ - & * used as unary/reference/
// splat) are intentionally excluded.
var binaryOperators = map[string]bool{
	"==": true, "===": true, "!=": true, "!==": true, "<>": true,
	"<=": true, ">=": true, "<=>": true, "<": true, ">": true,
	"&&": true, "||": true, "??": true, "=>": true,
}

// BinaryOperatorSpaces normalizes spacing to a single space around binary
// operators (assignment "=", arrow "=>", comparison and logical operators).
// Whitespace spanning a newline is left alone to preserve alignment. Reference
// assignment ("=& $x") and declare() headers are skipped.
type BinaryOperatorSpaces struct{}

func (BinaryOperatorSpaces) Name() string {
	return `PhpCsFixer\Fixer\Operator\BinaryOperatorSpacesFixer`
}

func (BinaryOperatorSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/BinaryOperatorSpacesFixer.php"
}

func (BinaryOperatorSpaces) Fix(s *tokens.Stream) bool {
	return normalizeSpaceAround(s, func(s *tokens.Stream, i int) bool {
		t := s.At(i)
		if t.Kind != token.Punct {
			return false
		}
		if t.Value == "=" {
			// leave reference assignment ("=& $x") and declare(...) headers alone
			return nextSignificantValue(s, i) != "&" && !insideDeclareArgs(s, i)
		}
		return binaryOperators[t.Value]
	})
}
