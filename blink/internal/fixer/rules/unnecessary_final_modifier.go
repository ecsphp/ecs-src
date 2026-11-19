package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/UnnecessaryFinalModifierSniff.php
//
// UnnecessaryFinalModifier flags final methods in a final class. The sniff only raises warnings,
// which ECS drops, so under ECS it reports nothing and fixes nothing.
type UnnecessaryFinalModifier struct{}

func (UnnecessaryFinalModifier) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\UnnecessaryFinalModifierSniff`
}

func (UnnecessaryFinalModifier) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/UnnecessaryFinalModifierSniff.php"
}

func (UnnecessaryFinalModifier) Fix(*tokens.Stream) bool { return false }

func (UnnecessaryFinalModifier) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
