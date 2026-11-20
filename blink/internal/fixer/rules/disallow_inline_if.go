package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowInlineIfSniff.php
//
// DisallowInlineIf reports every ternary. It fixes nothing.
type DisallowInlineIf struct{}

func (DisallowInlineIf) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\DisallowInlineIfSniff`
}

func (DisallowInlineIf) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowInlineIfSniff.php"
}

func (DisallowInlineIf) Fix(*tokens.Stream) bool { return false }

func (r DisallowInlineIf) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), func(f *slvFile, sc *phpcsScope, report func(i int, code, msg string)) {
		for i := range f.toks {
			if squizReportIsInlineThen(f, sc, i) {
				report(i, "Found", "Inline IF statements are not allowed")
			}
		}
	})
}
