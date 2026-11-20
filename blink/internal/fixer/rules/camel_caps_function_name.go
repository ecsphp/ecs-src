package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/CamelCapsFunctionNameSniff.php
//
// CamelCapsFunctionName reports function and method names not in strict camel
// caps, and names with a double underscore prefix that are no magic method.
type CamelCapsFunctionName struct{}

func (CamelCapsFunctionName) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\NamingConventions\CamelCapsFunctionNameSniff`
}

func (CamelCapsFunctionName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/NamingConventions/CamelCapsFunctionNameSniff.php"
}

func (CamelCapsFunctionName) Fix(*tokens.Stream) bool { return false }

func (c CamelCapsFunctionName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, c.Name())
	a.run(abstractScopeOO, phpcsSet("function"), func(i, scope int) {
		if !a.deepest(i, scope) {
			return
		}
		methodName := a.declarationName(i)
		if methodName == "" {
			return
		}
		className := a.className(scope)
		errorData := className + "::" + methodName
		methodLc, classLc := strings.ToLower(methodName), strings.ToLower(className)
		if abstractScopeMagicPrefixed(methodName) {
			if magic := methodLc[2:]; abstractScopeMagicMethods[magic] || abstractScopeDoubleUnderscoreMethods[magic] {
				return
			}
			a.addError(i, "MethodDoubleUnderscore", `Method name "%s" is invalid; only PHP magic methods should be prefixed with a double underscore`, errorData)
		}
		if methodLc == classLc || methodLc == "_"+classLc {
			return
		}
		if abstractScopeIsCamelCaps(strings.TrimLeft(methodName, "_"), true) {
			return
		}
		if props := a.methodProps(i); props.scopeSpecified {
			a.addError(i, "ScopeNotCamelCaps", `%s method name "%s" is not in camel caps format`, abstractScopeUcfirst(props.scope), errorData)
		} else {
			a.addError(i, "NotCamelCaps", `Method name "%s" is not in camel caps format`, errorData)
		}
	}, func(i int) {
		name := a.declarationName(i)
		if name == "" {
			return
		}
		if abstractScopeMagicPrefixed(name) {
			if strings.ToLower(name[2:]) == "autoload" {
				return
			}
			a.addError(i, "FunctionDoubleUnderscore", `Function name "%s" is invalid; only PHP magic methods should be prefixed with a double underscore`, name)
		}
		if !abstractScopeIsCamelCaps(strings.TrimLeft(name, "_"), true) {
			a.addError(i, "NotCamelCaps", `Function name "%s" is not in camel caps format`, name)
		}
	})
	return a.errors
}
