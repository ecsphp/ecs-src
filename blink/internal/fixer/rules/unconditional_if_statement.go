package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/UnconditionalIfStatementSniff.php
//
// UnconditionalIfStatement flags an if whose condition is the literal true or false. The sniff only raises warnings,
// which ECS drops, so under ECS it reports nothing and fixes nothing.
type UnconditionalIfStatement struct{}

func (UnconditionalIfStatement) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\UnconditionalIfStatementSniff`
}

func (UnconditionalIfStatement) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/UnconditionalIfStatementSniff.php"
}

func (UnconditionalIfStatement) Fix(*tokens.Stream) bool { return false }

func (UnconditionalIfStatement) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
