package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Metrics/CyclomaticComplexitySniff.php
//
// CyclomaticComplexity reports a function whose cyclomatic complexity exceeds
// absoluteComplexity (20). Its "exceeds complexity" warning is dropped by ECS.
type CyclomaticComplexity struct{}

func (CyclomaticComplexity) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Metrics\CyclomaticComplexitySniff`
}

func (CyclomaticComplexity) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Metrics/CyclomaticComplexitySniff.php"
}

func (CyclomaticComplexity) Fix(*tokens.Stream) bool { return false }

// cyclomaticComplexityPredicates is the sniff's PREDICATE_NODES.
var cyclomaticComplexityPredicates = phpcsSet("case", "default", "catch", "if", "for", "foreach", "while", "elseif",
	"??", "??=", "match_arrow", "?->")

func (c CyclomaticComplexity) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, c.Name())
	for i := range a.f.toks {
		if a.code(i) != "function" || a.sc.opener[i] < 0 || a.sc.closer[i] < 0 {
			continue
		}
		complexity := 1
		for j := a.sc.opener[i] + 1; j < a.sc.closer[i]; j++ {
			if cyclomaticComplexityPredicates[a.sc.code[j]] || (a.sc.code[j] == "?" && a.sc.isInlineThen(j)) {
				complexity++
			}
		}
		if complexity > 20 {
			a.addError(i, "MaxExceeded", "Function's cyclomatic complexity (%d) exceeds allowed maximum of %d", complexity, 20)
		}
	}
	return a.errors
}
