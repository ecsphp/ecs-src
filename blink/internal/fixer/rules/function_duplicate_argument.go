package rules

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/FunctionDuplicateArgumentSniff.php
//
// FunctionDuplicateArgument reports a parameter name a function declares twice.
type FunctionDuplicateArgument struct{}

func (FunctionDuplicateArgument) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Functions\FunctionDuplicateArgumentSniff`
}

func (FunctionDuplicateArgument) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/FunctionDuplicateArgumentSniff.php"
}

func (FunctionDuplicateArgument) Fix(*tokens.Stream) bool { return false }

func (r FunctionDuplicateArgument) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(s.Tokens())
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if !f.isNamedFunction(i) || sc.pCloser[i] < 0 {
			continue
		}
		closer := sc.pCloser[i]
		var seen []string
		for j := f.toks[closer].closer + 1; j < closer; j++ {
			if f.toks[j].kind != token.Variable {
				continue
			}
			v := f.toks[j].orig
			if slices.Contains(seen, v) {
				errs = append(errs, squizReportError(r.Name(), "Found", f.toks[j].line,
					`Variable "`+v+`" appears more than once in function declaration`))
			} else {
				seen = append(seen, v)
			}
		}
	}
	return errs
}
