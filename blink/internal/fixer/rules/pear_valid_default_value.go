package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Functions/ValidDefaultValueSniff.php
//
// PEARValidDefaultValue reports a parameter without a default that follows one
// with a default. It fixes nothing.
type PEARValidDefaultValue struct{}

func (PEARValidDefaultValue) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Functions\ValidDefaultValueSniff`
}

func (PEARValidDefaultValue) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Functions/ValidDefaultValueSniff.php"
}

func (PEARValidDefaultValue) Fix(*tokens.Stream) bool { return false }

func (r PEARValidDefaultValue) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	m := newSiModel(newSlvFile(lexer.Lex(s.Render())))
	f := m.f
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if c := m.code[i]; c != "function" && c != "closure" && c != "fn" || siIsMember(f, i) {
			continue
		}
		open := f.slvmParenOpener(i)
		if open < 0 || f.toks[open].closer < 0 {
			continue
		}
		defaultFound := false
		for _, p := range phpcsMethodParameters(f, open) {
			if p.variadicToken >= 0 {
				continue
			}
			if p.defaultToken >= 0 {
				end := f.toks[open].closer
				if p.commaToken >= 0 {
					end = p.commaToken
				}
				v := strings.ToLower(strings.TrimSpace(f.content(p.defaultToken, end-1)))
				defaultFound = p.typeHintToken < 0 || v != "null" && v != `\null`
				continue
			}
			if defaultFound {
				errs = append(errs, fixer.CodingStandardError{Line: f.toks[p.token].line,
					Message: "Arguments with default values must be at the end of the argument list", Checker: r.Name() + ".NotAtEnd"})
				break
			}
		}
	}
	return errs
}
