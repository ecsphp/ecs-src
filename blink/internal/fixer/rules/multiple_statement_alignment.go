package rules

import "github.com/ecsphp/ecs-src/blink/internal/tokens"

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/MultipleStatementAlignmentSniff.php
//
// MultipleStatementAlignment is a no-op: the sniff would align the "=" of
// consecutive assignments, but it reports through addFixableWarning and ECS
// drops warnings (outside its SniffFileProcessor::ESCALATE_WARNINGS_SNIFF
// list), so under ECS the sniff never changes code.
type MultipleStatementAlignment struct{}

func (MultipleStatementAlignment) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Formatting\MultipleStatementAlignmentSniff`
}

func (MultipleStatementAlignment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/MultipleStatementAlignmentSniff.php"
}

func (MultipleStatementAlignment) Fix(*tokens.Stream) bool {
	return false
}
