package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/UselessOverridingMethodSniff.php
//
// UselessOverridingMethod flags a method that only calls its parent. The sniff only raises warnings,
// which ECS drops, so under ECS it reports nothing and fixes nothing.
type UselessOverridingMethod struct{}

func (UselessOverridingMethod) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\UselessOverridingMethodSniff`
}

func (UselessOverridingMethod) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/UselessOverridingMethodSniff.php"
}

func (UselessOverridingMethod) Fix(*tokens.Stream) bool { return false }

func (UselessOverridingMethod) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
