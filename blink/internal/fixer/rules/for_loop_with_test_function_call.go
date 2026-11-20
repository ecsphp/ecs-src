package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/ForLoopWithTestFunctionCallSniff.php
//
// ForLoopWithTestFunctionCall flags a function call in a for loop condition. The sniff only raises warnings,
// which ECS drops, so under ECS it reports nothing and fixes nothing.
type ForLoopWithTestFunctionCall struct{}

func (ForLoopWithTestFunctionCall) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\ForLoopWithTestFunctionCallSniff`
}

func (ForLoopWithTestFunctionCall) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/ForLoopWithTestFunctionCallSniff.php"
}

func (ForLoopWithTestFunctionCall) Fix(*tokens.Stream) bool { return false }

func (ForLoopWithTestFunctionCall) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
