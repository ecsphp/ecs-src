package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Metrics/NestingLevelSniff.php
//
// NestingLevel reports a function nesting deeper than absoluteNestingLevel
// (10). Its "exceeds nesting level" warning is dropped by ECS.
type NestingLevel struct{}

func (NestingLevel) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Metrics\NestingLevelSniff`
}

func (NestingLevel) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Metrics/NestingLevelSniff.php"
}

func (NestingLevel) Fix(*tokens.Stream) bool { return false }

func (n NestingLevel) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, n.Name())
	_, levels := a.sc.levelMap()
	for i := range a.f.toks {
		if a.code(i) != "function" || a.sc.opener[i] < 0 || a.sc.closer[i] < 0 {
			continue
		}
		deepest := 0
		for j := a.sc.opener[i] + 1; j < a.sc.closer[i]; j++ {
			deepest = max(deepest, levels[j])
		}
		if level := deepest - levels[i] - 1; level > 10 {
			a.addError(i, "MaxExceeded", "Function's nesting level (%d) exceeds allowed maximum of %d", level, 10)
		}
	}
	return a.errors
}
