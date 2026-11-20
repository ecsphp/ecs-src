package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DiscourageGotoSniff.php
//
// DiscourageGoto only warns about goto, and ECS drops the warnings of this
// sniff, so it reports nothing.
type DiscourageGoto struct{}

func (DiscourageGoto) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\DiscourageGotoSniff`
}

func (DiscourageGoto) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/PHP/DiscourageGotoSniff.php"
}

func (DiscourageGoto) Fix(*tokens.Stream) bool { return false }
