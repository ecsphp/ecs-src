package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/NamingConventions/ValidFunctionNameSniff.php
//
// SquizValidFunctionName checks methods as PEAR's ValidFunctionName does, and
// wants plain functions in camel caps without a double underscore prefix.
type SquizValidFunctionName struct{}

func (SquizValidFunctionName) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\NamingConventions\ValidFunctionNameSniff`
}

func (SquizValidFunctionName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/NamingConventions/ValidFunctionNameSniff.php"
}

func (SquizValidFunctionName) Fix(*tokens.Stream) bool { return false }

func (v SquizValidFunctionName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, v.Name())
	a.run(abstractScopeOO, phpcsSet("function"), func(i, scope int) {
		pearValidFunctionNameMethod(a, i, scope)
	}, func(i int) {
		name := a.declarationName(i)
		if name == "" {
			return
		}
		test := name
		if abstractScopeMagicPrefixed(name) {
			a.addError(i, "DoubleUnderscore", `Function name "%s" is invalid; only PHP magic methods should be prefixed with a double underscore`, name)
			test = strings.TrimLeft(name, "_")
		}
		if !abstractScopeIsCamelCaps(test, false) {
			a.addError(i, "NotCamelCaps", `Function name "%s" is not in camel caps format`, name)
		}
	})
	return a.errors
}
