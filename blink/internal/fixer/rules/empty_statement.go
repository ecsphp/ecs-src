package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/EmptyStatementSniff.php
//
// EmptyStatement reports a control structure (if, else, for, try, catch, match,
// ...) whose body holds nothing but whitespace and comments. It fixes nothing.
type EmptyStatement struct{}

func (EmptyStatement) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\EmptyStatementSniff`
}

func (EmptyStatement) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/EmptyStatementSniff.php"
}

func (EmptyStatement) Fix(*tokens.Stream) bool { return false }

var emptyStatementKeywords = phpcsSet("try", "catch", "finally", "do", "else", "elseif", "for", "foreach",
	"if", "switch", "while", "match")

func (r EmptyStatement) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	sc := newPhpcsScope(newSlvFile(s.Tokens()))
	f := sc.f
	var errs []fixer.CodingStandardError
	for i, t := range f.toks {
		if !emptyStatementKeywords[sc.code[i]] || sc.opener[i] < 0 {
			continue
		}
		// findNext stops before its end, so the token right before the closer is never looked at
		if f.nextEffective(sc.opener[i]+1, sc.closer[i]-1) >= 0 {
			continue
		}
		name := strings.ToUpper(t.orig)
		errs = append(errs, fixer.CodingStandardError{
			Line:    t.line,
			Message: "Empty " + name + " statement detected",
			Checker: r.Name() + ".Detected" + name[:1] + strings.ToLower(name[1:]),
		})
	}
	return errs
}
