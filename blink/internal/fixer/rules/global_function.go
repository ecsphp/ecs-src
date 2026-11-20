package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/GlobalFunctionSniff.php
//
// GlobalFunction only warns about a function declared outside any scope, and
// ECS drops that warning, so it reports nothing.
type GlobalFunction struct{}

func (GlobalFunction) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Functions\GlobalFunctionSniff`
}

func (GlobalFunction) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/GlobalFunctionSniff.php"
}

func (GlobalFunction) Fix(*tokens.Stream) bool { return false }

func (GlobalFunction) Errors(*tokens.Stream) []fixer.CodingStandardError {
	return nil
}
