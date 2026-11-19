package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsOperators are PHP_CodeSniffer's Tokens::OPERATORS.
var phpcsOperators = map[string]bool{
	"+": true, "-": true, "*": true, "/": true, "%": true, "**": true,
	"<=>": true, "??": true, "&": true, "|": true, "^": true, "<<": true, ">>": true,
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Strings/EchoedStringsSniff.php
//
// EchoedStrings drops the parentheses around an echoed value: echo("x"); -> echo "x";.
// Values holding an operator keep them.
type EchoedStrings struct{}

func (EchoedStrings) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Strings\EchoedStringsSniff`
}

func (EchoedStrings) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Strings/EchoedStringsSniff.php"
}

func (EchoedStrings) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Keyword || !strings.EqualFold(t.Value, "echo") {
			continue
		}
		open := i + 1
		for open < s.Len() && s.At(open).Kind == token.Whitespace {
			open++
		}
		if open >= s.Len() || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		end := i
		for end < s.Len() && s.At(end).Kind != token.CloseTag && (s.At(end).Kind != token.Punct || s.At(end).Value != ";") {
			end++
		}
		if end >= s.Len() {
			continue
		}
		closeIdx := end - 1
		for closeIdx > open && s.At(closeIdx).Kind == token.Whitespace {
			closeIdx--
		}
		if s.At(closeIdx).Value != ")" || s.MatchForward(open) != closeIdx {
			continue
		}
		hasOperator := false
		for j := i; j < end; j++ {
			if s.At(j).Kind == token.Punct && phpcsOperators[s.At(j).Value] {
				hasOperator = true
				break
			}
		}
		if hasOperator {
			continue
		}
		s.RemoveAt(closeIdx)
		s.RemoveAt(open)
		if s.At(open-1).Kind != token.Whitespace {
			s.InsertAt(open, token.Token{Kind: token.Whitespace, Value: " "})
		}
		changed = true
	}
	return changed
}
