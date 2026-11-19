package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Strings/UnnecessaryStringConcatSniff.php
//
// UnnecessaryStringConcat reports a concat of two string literals of the same
// quote type, which could be one string. It fixes nothing.
type UnnecessaryStringConcat struct {
	AllowMultiline bool
}

func (UnnecessaryStringConcat) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Strings\UnnecessaryStringConcatSniff`
}

func (UnnecessaryStringConcat) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Strings/UnnecessaryStringConcatSniff.php"
}

func (u UnnecessaryStringConcat) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["allowMultiline"].(bool); ok {
		u.AllowMultiline = v
	}
	return u
}

func (UnnecessaryStringConcat) Fix(*tokens.Stream) bool { return false }

func (u UnnecessaryStringConcat) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	// Tokens::STRING_TOKENS: T_CONSTANT_ENCAPSED_STRING and T_DOUBLE_QUOTED_STRING
	isString := func(i int) bool { return i >= 0 && f.toks[i].kind == token.String && !f.toks[i].heredoc }
	var errs []fixer.CodingStandardError
	for i, t := range f.toks {
		if t.kind != token.Punct || t.orig != "." {
			continue
		}
		prev := i - 1
		for prev >= 0 && f.toks[prev].kind == token.Whitespace {
			prev--
		}
		next := i + 1
		for next < len(f.toks) && f.toks[next].kind == token.Whitespace {
			next++
		}
		if next >= len(f.toks) || !isString(prev) || !isString(next) {
			continue
		}
		p, n := f.toks[prev].content, f.toks[next].content
		if p[0] != n[0] || len(p) < 2 || len(n) < 2 {
			continue
		}
		if c := p[len(p)-2:len(p)-1] + n[1:2]; c == "?>" || c == "<?" {
			continue
		}
		if u.AllowMultiline && f.toks[prev].line != f.toks[next].line {
			continue
		}
		errs = append(errs, fixer.CodingStandardError{
			Line:    t.line,
			Message: "String concat is not required here; use a single string instead",
			Checker: u.Name() + ".Found",
		})
	}
	return errs
}
