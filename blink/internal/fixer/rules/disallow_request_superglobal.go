package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DisallowRequestSuperglobalSniff.php
//
// DisallowRequestSuperglobal reports every $_REQUEST variable outside a
// double-quoted string or heredoc.
type DisallowRequestSuperglobal struct{}

func (DisallowRequestSuperglobal) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\DisallowRequestSuperglobalSniff`
}

func (DisallowRequestSuperglobal) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DisallowRequestSuperglobalSniff.php"
}

func (DisallowRequestSuperglobal) Fix(*tokens.Stream) bool { return false }

func (d DisallowRequestSuperglobal) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	var lines []int
	var encapsed []bool
	for i := range s.Len() {
		if t := s.At(i); t.Kind != token.Variable || t.Value != "$_REQUEST" {
			continue
		}
		if encapsed == nil {
			encapsed = encapsedTokens(s)
			lines = phpcsTokenLines(s)
		}
		if encapsed[i] {
			continue
		}
		errs = append(errs, fixer.CodingStandardError{
			Line:    lines[i],
			Message: "The $_REQUEST superglobal should not be used; use $_GET, $_POST, or $_COOKIE instead",
			Checker: d.Name() + ".Found",
		})
	}
	return errs
}
