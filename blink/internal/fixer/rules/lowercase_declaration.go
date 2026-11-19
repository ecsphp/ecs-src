package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// lowercaseDeclarationKeywords are the control structure keywords the sniff registers.
var lowercaseDeclarationKeywords = map[string]bool{
	"if": true, "else": true, "elseif": true, "foreach": true, "for": true, "do": true,
	"switch": true, "while": true, "try": true, "catch": true, "match": true,
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/LowercaseDeclarationSniff.php
//
// LowercaseDeclaration lowercases control structure keywords: "IF ($a)" -> "if ($a)".
type LowercaseDeclaration struct{}

func (LowercaseDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\ControlStructures\LowercaseDeclarationSniff`
}

func (LowercaseDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/LowercaseDeclarationSniff.php"
}

func (LowercaseDeclaration) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Keyword || !lowercaseDeclarationKeywords[strings.ToLower(t.Value)] {
			continue
		}
		if lower := lowerASCII(t.Value); lower != t.Value {
			s.SetValue(i, lower)
			changed = true
		}
	}
	return changed
}
