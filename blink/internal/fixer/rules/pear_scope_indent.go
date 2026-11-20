package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/WhiteSpace/ScopeIndentSniff.php
//
// PEARScopeIndent is Generic ScopeIndent that does not indent switch bodies.
type PEARScopeIndent struct {
	ScopeIndent
}

func (PEARScopeIndent) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\WhiteSpace\ScopeIndentSniff`
}

func (PEARScopeIndent) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/WhiteSpace/ScopeIndentSniff.php"
}

func (r PEARScopeIndent) WithConfig(config map[string]any) fixer.Fixer {
	if c, ok := r.ScopeIndent.WithConfig(config).(ScopeIndent); ok {
		r.ScopeIndent = c
	}
	return r
}

func (r PEARScopeIndent) Fix(s *tokens.Stream) bool { return r.ScopeIndent.Fix(s) }
