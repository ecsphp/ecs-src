package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Arrays/DisallowLongArraySyntaxSniff.php
//
// DisallowLongArraySyntax rewrites "array(...)" to "[...]". Only the keyword and
// the parentheses change: "array /* c */ (1)" -> " /* c */ [1]". An array
// without a matching ")" is left alone.
type DisallowLongArraySyntax struct{}

func (DisallowLongArraySyntax) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Arrays\DisallowLongArraySyntaxSniff`
}

func (DisallowLongArraySyntax) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Arrays/DisallowLongArraySyntaxSniff.php"
}

func (DisallowLongArraySyntax) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Keyword || !strings.EqualFold(t.Value, "array") {
			continue
		}
		open := i + 1
		for open < s.Len() && isPhpcsEmpty(s.At(open)) {
			open++
		}
		if open >= s.Len() || s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
			continue
		}
		closeIdx := s.MatchForward(open)
		if closeIdx < 0 {
			continue
		}
		s.SetValue(open, "[")
		s.SetValue(closeIdx, "]")
		s.RemoveAt(i)
		changed = true
	}
	return changed
}
