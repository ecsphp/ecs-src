package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Classes/ClosingBraceSniff.php
//
// PSR12ClosingBrace reports code or a comment after the closing brace of a
// class, interface, trait, enum or function on the same line. It fixes nothing.
type PSR12ClosingBrace struct{}

func (PSR12ClosingBrace) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Classes\ClosingBraceSniff`
}

func (PSR12ClosingBrace) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Classes/ClosingBraceSniff.php"
}

func (PSR12ClosingBrace) Fix(*tokens.Stream) bool { return false }

func (r PSR12ClosingBrace) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	for i, t := range f.toks {
		if t.kind != token.Keyword || sc.cond[i] != i || sc.closer[i] < 0 {
			continue
		}
		switch sc.code[i] {
		case "function":
			if f.isClosure(i) {
				continue
			}
		case "class", "interface", "trait", "enum":
			if !phpcsIsOOKeyword(f, i) || f.isAnonClass(i) {
				continue
			}
		default:
			continue
		}
		closer := sc.closer[i]
		next := closer + 1
		for next < len(f.toks) && f.toks[next].kind == token.Whitespace {
			next++
		}
		if next == len(f.toks) || f.toks[next].line != f.toks[closer].line {
			continue
		}
		errs = append(errs, fixer.CodingStandardError{
			Line:    f.toks[closer].line,
			Message: "Closing brace must not be followed by any comment or statement on the same line",
			Checker: r.Name() + ".StatementAfter",
		})
	}
	return errs
}
