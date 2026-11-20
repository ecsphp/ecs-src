package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Classes/DuplicateClassNameSniff.php
//
// DuplicateClassName reports nothing under ECS: upstream only warns about a
// class name declared twice across the files of a run, and ECS drops the
// warnings of this sniff. It fixes nothing.
type DuplicateClassName struct{}

func (DuplicateClassName) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Classes\DuplicateClassNameSniff`
}

func (DuplicateClassName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Classes/DuplicateClassNameSniff.php"
}

func (DuplicateClassName) Fix(*tokens.Stream) bool { return false }

func (DuplicateClassName) Errors(*tokens.Stream) []fixer.CodingStandardError { return nil }
