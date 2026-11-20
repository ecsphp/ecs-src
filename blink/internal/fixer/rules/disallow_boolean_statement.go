package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowBooleanStatementSniff.php
//
// DisallowBooleanStatement reports a boolean operator outside the parentheses
// of a control structure or other parenthesis owner. It fixes nothing.
type DisallowBooleanStatement struct{}

func (DisallowBooleanStatement) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\DisallowBooleanStatementSniff`
}

func (DisallowBooleanStatement) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowBooleanStatementSniff.php"
}

func (DisallowBooleanStatement) Fix(*tokens.Stream) bool { return false }

func (r DisallowBooleanStatement) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), func(f *slvFile, sc *phpcsScope, report func(i int, code, msg string)) {
		for i := range f.toks {
			if !squizReportIsBooleanOperator(f, i) {
				continue
			}
			owned := false
			for _, open := range squizReportNested(sc, i) {
				owned = owned || squizReportParenOwner(f, sc, open) >= 0
			}
			if !owned {
				report(i, "Found", "Boolean operators are not allowed outside of control structure conditions")
			}
		}
	})
}
