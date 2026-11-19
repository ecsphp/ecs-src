package rules

import "github.com/ecsphp/ecs-src/blink/internal/tokens"

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/EmptyPHPStatementSniff.php
//
// EmptyPHPStatement is a no-op: the sniff would drop superfluous semicolons
// and empty "<?php ?>" pairs, but it reports through addFixableWarning and ECS
// drops warnings, so under ECS the sniff never changes code.
type EmptyPHPStatement struct{}

func (EmptyPHPStatement) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\EmptyPHPStatementSniff`
}

func (EmptyPHPStatement) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/EmptyPHPStatementSniff.php"
}

func (EmptyPHPStatement) Fix(*tokens.Stream) bool {
	return false
}
