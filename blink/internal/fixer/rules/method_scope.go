package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Scope/MethodScopeSniff.php
//
// MethodScope reports a method declared without a visibility.
type MethodScope struct{}

func (MethodScope) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Scope\MethodScopeSniff`
}

func (MethodScope) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Scope/MethodScopeSniff.php"
}

func (MethodScope) Fix(*tokens.Stream) bool { return false }

func (m MethodScope) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, m.Name())
	a.run(abstractScopeOO, phpcsSet("function"), func(i, scope int) {
		if !a.deepest(i, scope) {
			return
		}
		name := a.declarationName(i)
		if name != "" && !a.methodProps(i).scopeSpecified {
			a.addError(i, "Missing", `Visibility must be declared on method "%s"`, name)
		}
	}, nil)
	return a.errors
}
