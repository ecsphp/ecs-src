package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR1/Sniffs/Files/SideEffectsSniff.php
//
// SideEffects only warns about a file that both declares symbols and has side
// effects. ECS drops the warnings of every sniff it does not escalate, so under
// ECS the sniff reports nothing. It fixes nothing.
type SideEffects struct{}

func (SideEffects) Name() string {
	return `PHP_CodeSniffer\Standards\PSR1\Sniffs\Files\SideEffectsSniff`
}

func (SideEffects) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR1/Sniffs/Files/SideEffectsSniff.php"
}

func (SideEffects) Fix(*tokens.Stream) bool { return false }

func (SideEffects) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
