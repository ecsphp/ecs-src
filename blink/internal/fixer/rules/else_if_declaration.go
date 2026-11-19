package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/ElseIfDeclarationSniff.php
//
// SquizElseIfDeclaration splits "elseif" into "else if".
type SquizElseIfDeclaration struct{}

func (SquizElseIfDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\ControlStructures\ElseIfDeclarationSniff`
}

func (SquizElseIfDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/ElseIfDeclarationSniff.php"
}

func (SquizElseIfDeclaration) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.Keyword && strings.EqualFold(t.Value, "elseif") {
			s.SetValue(i, "else if")
			changed = true
		}
	}
	return changed
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/ControlStructures/ElseIfDeclarationSniff.php
//
// PSR2ElseIfDeclaration is a no-op: the sniff would join "else if" into
// "elseif", but it reports through addFixableWarning and ECS drops warnings
// (outside its SniffFileProcessor::ESCALATE_WARNINGS_SNIFF list), so under ECS
// the sniff never changes code.
type PSR2ElseIfDeclaration struct{}

func (PSR2ElseIfDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\ControlStructures\ElseIfDeclarationSniff`
}

func (PSR2ElseIfDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/ControlStructures/ElseIfDeclarationSniff.php"
}

func (PSR2ElseIfDeclaration) Fix(*tokens.Stream) bool {
	return false
}
