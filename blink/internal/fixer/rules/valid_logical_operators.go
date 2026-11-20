package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Operators/ValidLogicalOperatorsSniff.php
//
// ValidLogicalOperators reports "and" and "or", which must be "&&" and "||".
// It fixes nothing.
type ValidLogicalOperators struct{}

func (ValidLogicalOperators) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Operators\ValidLogicalOperatorsSniff`
}

func (ValidLogicalOperators) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Operators/ValidLogicalOperatorsSniff.php"
}

func (ValidLogicalOperators) Fix(*tokens.Stream) bool { return false }

func (r ValidLogicalOperators) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), func(f *slvFile, sc *phpcsScope, report func(i int, code, msg string)) {
		for i := range f.toks {
			if f.isKeyword(i, "and") || f.isKeyword(i, "or") {
				op := strings.ToLower(f.toks[i].orig)
				alt := map[string]string{"and": "&&", "or": "||"}[op]
				report(i, "NotAllowed", fmt.Sprintf(`Logical operator "%s" is prohibited; use "%s" instead`, op, alt))
			}
		}
	})
}
