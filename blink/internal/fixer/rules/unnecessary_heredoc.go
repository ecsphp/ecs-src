package rules

import "github.com/ecsphp/ecs-src/blink/internal/tokens"

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Strings/UnnecessaryHeredocSniff.php
//
// UnnecessaryHeredoc is a no-op: the sniff turns a heredoc without
// interpolation into a nowdoc through addFixableWarning, which ECS drops.
type UnnecessaryHeredoc struct{}

func (UnnecessaryHeredoc) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Strings\UnnecessaryHeredocSniff`
}

func (UnnecessaryHeredoc) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Strings/UnnecessaryHeredocSniff.php"
}

func (UnnecessaryHeredoc) Fix(*tokens.Stream) bool {
	return false
}
