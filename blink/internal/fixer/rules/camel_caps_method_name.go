package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR1/Sniffs/Methods/CamelCapsMethodNameSniff.php
//
// CamelCapsMethodName reports method names not in camel caps; acronyms may
// open the name.
type CamelCapsMethodName struct{}

func (CamelCapsMethodName) Name() string {
	return `PHP_CodeSniffer\Standards\PSR1\Sniffs\Methods\CamelCapsMethodNameSniff`
}

func (CamelCapsMethodName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR1/Sniffs/Methods/CamelCapsMethodNameSniff.php"
}

func (CamelCapsMethodName) Fix(*tokens.Stream) bool { return false }

func (c CamelCapsMethodName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, c.Name())
	a.run(abstractScopeOO, phpcsSet("function"), func(i, scope int) {
		if !a.deepest(i, scope) {
			return
		}
		methodName := a.declarationName(i)
		if methodName == "" {
			return
		}
		if abstractScopeMagicPrefixed(methodName) {
			if magic := strings.ToLower(methodName[2:]); abstractScopeMagicMethods[magic] || abstractScopeDoubleUnderscoreMethods[magic] {
				return
			}
		}
		if test := strings.TrimLeft(methodName, "_"); test != "" && !abstractScopeIsCamelCaps(test, false) {
			a.addError(i, "NotCamelCaps", `Method name "%s" is not in camel caps format`, a.className(scope)+"::"+methodName)
		}
	}, nil)
	return a.errors
}
