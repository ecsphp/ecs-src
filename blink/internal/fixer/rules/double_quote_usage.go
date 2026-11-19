package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// doubleQuoteEscapes keep a double-quoted string as is: escape sequences single
// quotes cannot express, and a literal "'".
var doubleQuoteEscapes = []string{
	`\0`, `\1`, `\2`, `\3`, `\4`, `\5`, `\6`, `\7`,
	`\n`, `\r`, `\f`, `\t`, `\v`, `\x`, `\b`, `\e`, `\u`, `'`,
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Strings/DoubleQuoteUsageSniff.php
//
// DoubleQuoteUsage turns a double-quoted string without interpolation or
// special escapes into a single-quoted one: "abc" -> 'abc', "\"x\$" -> '"x$'.
type DoubleQuoteUsage struct{}

func (DoubleQuoteUsage) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Strings\DoubleQuoteUsageSniff`
}

func (DoubleQuoteUsage) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Strings/DoubleQuoteUsageSniff.php"
}

func (DoubleQuoteUsage) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.String || t.Part != token.Whole {
			continue
		}
		// PHP_CodeSniffer splits a lowercase "b" prefix off as a binary cast;
		// with "B" the string keeps its prefix and is not double-quoted to it
		prefix, v := token.SplitBinaryPrefix(t.Value)
		if prefix == "B" || len(v) < 2 || v[0] != '"' || v[len(v)-1] != '"' {
			continue
		}
		if slicesContainsSub(v, doubleQuoteEscapes) {
			continue
		}
		inner := strings.ReplaceAll(v[1:len(v)-1], `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\$`, `$`)
		s.SetValue(i, prefix+"'"+inner+"'")
		changed = true
	}
	return changed
}

func slicesContainsSub(v string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(v, sub) {
			return true
		}
	}
	return false
}
