package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/SpaceBeforeCastSniff.php
//
// SpaceBeforeCast puts a single space before a cast: "$a =(int) $b" -> "$a = (int) $b".
// A b"..." binary string counts as a cast; one at the start of a line, or
// indented, is left alone.
type SpaceBeforeCast struct{}

func (SpaceBeforeCast) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Formatting\SpaceBeforeCastSniff`
}

func (SpaceBeforeCast) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/SpaceBeforeCastSniff.php"
}

func (SpaceBeforeCast) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		closeIdx, _, ok := phpCastAt(s, i)
		if !ok && isBinaryStringCast(s.At(i)) {
			closeIdx, ok = i, true
		}
		if !ok {
			continue
		}
		switch {
		case startsLine(s, i):
		case s.At(i-1).Kind != token.Whitespace:
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
			closeIdx++
			changed = true
		case !strings.Contains(s.At(i-1).Value, "\n") && !startsLine(s, i-1) &&
			phpcsTabWidth(s, i-1) != 1:
			s.SetValue(i-1, " ")
			changed = true
		}
		i = closeIdx
	}
	return changed
}

// isBinaryStringCast reports a b"..." / b'...' literal: PHP_CodeSniffer splits
// its "b" off as a T_BINARY_CAST token.
func isBinaryStringCast(t token.Token) bool {
	return t.Kind == token.String && (t.Part == token.Whole || t.Part == token.Head) &&
		(strings.HasPrefix(t.Value, `b"`) || strings.HasPrefix(t.Value, "b'"))
}

// startsLine reports whether the token at i begins a line (PHP_CodeSniffer column 1).
func startsLine(s *tokens.Stream, i int) bool {
	return i == 0 || strings.HasSuffix(s.At(i-1).Value, "\n")
}

// phpcsTabWidth is the length PHP_CodeSniffer gives the single-line whitespace
// token at i: tabs expand to the next multiple of 4 (ECS's tabWidth) from the
// token's column.
func phpcsTabWidth(s *tokens.Stream, i int) int {
	var parts []string
	for j := i - 1; j >= 0; j-- {
		v := s.At(j).Value
		if nl := strings.LastIndexByte(v, '\n'); nl >= 0 {
			parts = append(parts, v[nl+1:])
			break
		}
		parts = append(parts, v)
	}
	slices.Reverse(parts)
	return expandedWidth(s.At(i).Value, expandedWidth(strings.Join(parts, ""), 0))
}

func expandedWidth(v string, col int) int {
	start := col
	for _, r := range v {
		if r == '\t' {
			col += 4 - col%4
		} else {
			col++
		}
	}
	return col - start
}
