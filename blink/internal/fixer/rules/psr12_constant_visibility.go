package rules

import "github.com/ecsphp/ecs-src/blink/internal/tokens"

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Properties/ConstantVisibilitySniff.php
//
// PSR12ConstantVisibility only warns about a class constant without visibility,
// and ECS drops PHP_CodeSniffer warnings, so under ECS it reports and fixes
// nothing.
type PSR12ConstantVisibility struct{}

func (PSR12ConstantVisibility) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Properties\ConstantVisibilitySniff`
}

func (PSR12ConstantVisibility) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Properties/ConstantVisibilitySniff.php"
}

func (PSR12ConstantVisibility) Fix(*tokens.Stream) bool { return false }
