package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/ForLoopShouldBeWhileLoopSniff.php
//
// ForLoopShouldBeWhileLoop suggests a while loop for a for loop with only a condition. The sniff only raises warnings,
// which ECS drops, so under ECS it reports nothing and fixes nothing.
type ForLoopShouldBeWhileLoop struct{}

func (ForLoopShouldBeWhileLoop) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\ForLoopShouldBeWhileLoopSniff`
}

func (ForLoopShouldBeWhileLoop) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/ForLoopShouldBeWhileLoopSniff.php"
}

func (ForLoopShouldBeWhileLoop) Fix(*tokens.Stream) bool { return false }

func (ForLoopShouldBeWhileLoop) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
