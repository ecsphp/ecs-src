package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/NoSilencedErrorsSniff.php
//
// NoSilencedErrors warns about the @ operator unless its error property is
// set. ECS drops the warnings of this sniff, so with the defaults it reports
// nothing; blink does not port the error property.
type NoSilencedErrors struct{}

func (NoSilencedErrors) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\NoSilencedErrorsSniff`
}

func (NoSilencedErrors) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/NoSilencedErrorsSniff.php"
}

func (NoSilencedErrors) Fix(*tokens.Stream) bool { return false }
