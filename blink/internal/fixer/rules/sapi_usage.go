package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/SAPIUsageSniff.php
//
// SAPIUsage reports every php_sapi_name() call, asking for PHP_SAPI instead.
type SAPIUsage struct{}

func (SAPIUsage) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\SAPIUsageSniff`
}

func (SAPIUsage) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/SAPIUsageSniff.php"
}

func (SAPIUsage) Fix(*tokens.Stream) bool { return false }

func (u SAPIUsage) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	var lines []int
	var encapsed []bool
	for i := range s.Len() {
		if t := s.At(i); t.Kind != token.Ident || !strings.EqualFold(t.Value, "php_sapi_name") {
			continue
		}
		start, ok := forbiddenFunctionsNameStart(s, i)
		if !ok || (start != i && start > 0 && kwIs(s.At(start-1), "namespace")) {
			continue
		}
		if encapsed == nil {
			encapsed = encapsedTokens(s)
			lines = phpcsTokenLines(s)
		}
		if encapsed[i] || sapiUsageIgnoredBefore(s, start) {
			continue
		}
		errs = append(errs, fixer.CodingStandardError{
			Line:    lines[start],
			Message: "Use the PHP_SAPI constant instead of calling php_sapi_name()",
			Checker: u.Name() + ".FunctionFound",
		})
	}
	return errs
}

// sapiUsageIgnoredBefore reports whether the token before the name at i, comments
// included, makes it no call to the PHP function: a member, function or constant.
func sapiUsageIgnoredBefore(s *tokens.Stream, i int) bool {
	p := i - 1
	for p >= 0 && s.At(p).Kind == token.Whitespace {
		p--
	}
	if p < 0 {
		return false
	}
	t := s.At(p)
	if kwIs(t, "function") || kwIs(t, "const") {
		// "use function" / "use const" import the name: no T_FUNCTION or T_CONST
		u := sigPrev(s, p)
		return u < 0 || !kwIs(s.At(u), "use")
	}
	return isPunctVal(s, p, "::") || isPunctVal(s, p, "->") || isPunctVal(s, p, "?->")
}
