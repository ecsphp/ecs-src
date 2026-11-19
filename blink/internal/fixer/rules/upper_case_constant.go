package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/UpperCaseConstantSniff.php
//
// UpperCaseConstant uppercases true, false and null ("true" -> "TRUE"), with
// LowerCaseConstant's skipping of type declarations.
type UpperCaseConstant struct{}

func (UpperCaseConstant) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\UpperCaseConstantSniff`
}

func (UpperCaseConstant) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/UpperCaseConstantSniff.php"
}

func (UpperCaseConstant) Fix(s *tokens.Stream) bool {
	return fixConstantCase(s, strings.ToUpper)
}
