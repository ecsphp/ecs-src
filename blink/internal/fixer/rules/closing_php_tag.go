package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/ClosingPHPTagSniff.php
//
// ClosingPHPTag reports every PHP open tag no close tag follows.
type ClosingPHPTag struct{}

func (ClosingPHPTag) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\ClosingPHPTagSniff`
}

func (ClosingPHPTag) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/ClosingPHPTagSniff.php"
}

func (ClosingPHPTag) Fix(*tokens.Stream) bool { return false }

func (c ClosingPHPTag) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	var errs []fixer.CodingStandardError
	var lines []int
	for i := s.Len() - 1; i >= 0; i-- {
		switch s.At(i).Kind {
		case token.CloseTag:
			return errs
		case token.OpenTag:
			if lines == nil {
				lines = phpcsTokenLines(s)
			}
			errs = append(errs, fixer.CodingStandardError{
				Line:    lines[i],
				Message: "The PHP open tag does not have a corresponding PHP close tag",
				Checker: c.Name() + ".NotFound",
			})
		}
	}
	return errs
}
