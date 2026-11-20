package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/NonExecutableCodeSniff.php
//
// NonExecutableCode reports code after return, exit and the like, but only
// as warnings, which ECS drops for this sniff, so blink reports nothing.
type NonExecutableCode struct{}

func (NonExecutableCode) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\NonExecutableCodeSniff`
}

func (NonExecutableCode) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/NonExecutableCodeSniff.php"
}

func (NonExecutableCode) Fix(*tokens.Stream) bool { return false }

func (NonExecutableCode) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
