package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/FunctionDeclarationSniff.php
//
// SquizFunctionDeclaration reports a function declaration not laid out as
// "function abc(...)". It fixes nothing.
type SquizFunctionDeclaration struct{}

func (SquizFunctionDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Functions\FunctionDeclarationSniff`
}

func (SquizFunctionDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/FunctionDeclarationSniff.php"
}

func (SquizFunctionDeclaration) Fix(*tokens.Stream) bool { return false }

var squizFunctionDeclarationPatterns = []abstractPattern{
	abstractPatternParse("function abc(...);"),
	abstractPatternParse("function abc(...)"),
	abstractPatternParse("abstract function abc(...);"),
}

func (r SquizFunctionDeclaration) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return abstractPatternErrors(s, r.Name(), squizFunctionDeclarationPatterns, false, func(m *siModel, i int) bool {
		return m.code[i] == "function"
	})
}
