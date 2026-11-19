package rules

import "github.com/ecsphp/ecs-src/blink/internal/tokens"

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/RequireStrictTypesSniff.php
//
// RequireStrictTypes is a no-op: the sniff's only fix ("strict_types=0" -> "1")
// is an addFixableWarning, which ECS drops, so under ECS it never changes code.
type RequireStrictTypes struct{}

func (RequireStrictTypes) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\RequireStrictTypesSniff`
}

func (RequireStrictTypes) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/RequireStrictTypesSniff.php"
}

func (RequireStrictTypes) Fix(*tokens.Stream) bool {
	return false
}
