package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/EmptyCatchCommentSniff.php
//
// EmptyCatchComment reports a catch block holding nothing, not even a comment.
type EmptyCatchComment struct{}

func (EmptyCatchComment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Commenting\EmptyCatchCommentSniff`
}

func (EmptyCatchComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Commenting/EmptyCatchCommentSniff.php"
}

func (EmptyCatchComment) Fix(*tokens.Stream) bool { return false }

func (r EmptyCatchComment) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(s.Tokens())
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if !f.isKeyword(i, "catch") || sc.cond[i] != i || sc.opener[i] < 0 {
			continue
		}
		opener, closer := sc.opener[i], sc.closer[i]
		if f.nextNonWhitespace(opener+1, closer) < 0 {
			errs = append(errs, squizReportError(r.Name(), "Missing", f.toks[opener].line,
				"Empty CATCH statement must have a comment to explain why the exception is not handled"))
		}
	}
	return errs
}
