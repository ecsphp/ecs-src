package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Classes/OpeningBraceSpaceSniff.php
//
// OpeningBraceSpace removes the blank lines after the opening brace of a class,
// interface, trait or enum body.
type OpeningBraceSpace struct{}

func (OpeningBraceSpace) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Classes\OpeningBraceSpaceSniff`
}

func (OpeningBraceSpace) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Classes/OpeningBraceSpaceSniff.php"
}

func (OpeningBraceSpace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		open := phpcsOOScopeOpener(s, i)
		if open < 0 || open+1 >= s.Len() || s.At(open+1).Kind != token.Whitespace {
			continue
		}
		ws := s.At(open + 1).Value
		first := strings.Index(ws, "\n")
		last := strings.LastIndex(ws, "\n")
		if first < 0 || first == last || open+2 >= s.Len() {
			continue
		}
		s.SetValue(open+1, ws[:first+1]+ws[last+1:])
		changed = true
	}
	return changed
}
