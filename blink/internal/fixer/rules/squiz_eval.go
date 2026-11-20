package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/EvalSniff.php
//
// SquizEval only warns about eval(), and ECS drops the warnings of this
// sniff, so it reports nothing.
type SquizEval struct{}

func (SquizEval) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\EvalSniff`
}

func (SquizEval) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/EvalSniff.php"
}

func (SquizEval) Fix(*tokens.Stream) bool { return false }
