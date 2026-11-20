package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/JumbledIncrementerSniff.php
//
// JumbledIncrementer flags an inner for loop incrementing the outer loop variable. The sniff only raises warnings,
// which ECS drops, so under ECS it reports nothing and fixes nothing.
type JumbledIncrementer struct{}

func (JumbledIncrementer) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\JumbledIncrementerSniff`
}

func (JumbledIncrementer) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/JumbledIncrementerSniff.php"
}

func (JumbledIncrementer) Fix(*tokens.Stream) bool { return false }

func (JumbledIncrementer) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
