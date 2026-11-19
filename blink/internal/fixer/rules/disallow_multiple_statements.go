package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/DisallowMultipleStatementsSniff.php
//
// DisallowMultipleStatements breaks a statement that shares its line with the
// previous one onto its own line: "$a = 1; $b = 2;" -> "$a = 1;\n$b = 2;".
type DisallowMultipleStatements struct{}

func (DisallowMultipleStatements) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Formatting\DisallowMultipleStatementsSniff`
}

func (DisallowMultipleStatements) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Formatting/DisallowMultipleStatementsSniff.php"
}

func (DisallowMultipleStatements) Fix(s *tokens.Stream) bool {
	// prev is the last ";" seen since the last open tag (-1 for none).
	prev, prevLine, prevInFor, ignored := -1, 0, false, false
	line := 1
	var parens []bool // each open "(": whether it is a for() condition
	var breaks []int  // semicolons to put a newline after
	inFor := func() bool {
		for _, f := range parens {
			if f {
				return true
			}
		}
		return false
	}
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		switch {
		case t.Kind == token.OpenTag:
			prev, ignored = -1, false
		case t.Kind == token.Comment && isPhpcsIgnoreComment(t.Value):
			ignored = true
		case t.Kind == token.Punct && t.Value == "(":
			parens = append(parens, isForParen(s, i))
		case t.Kind == token.Punct && t.Value == ")":
			if len(parens) > 0 {
				parens = parens[:len(parens)-1]
			}
		case t.Kind == token.Punct && t.Value == ";":
			inForNow := inFor()
			if prev >= 0 && !ignored && !inForNow && !prevInFor && prevLine == line {
				breaks = append(breaks, prev)
			}
			prev, prevLine, prevInFor, ignored = i, line, inForNow, false
		}
		line += strings.Count(t.Value, "\n")
	}
	if len(breaks) == 0 {
		return false
	}
	eol := phpcsEolChar(s)
	for _, p := range slices.Backward(breaks) {
		if p+1 < s.Len() && s.At(p+1).Kind == token.Whitespace {
			s.RemoveAt(p + 1)
		}
		s.InsertAt(p+1, token.Token{Kind: token.Whitespace, Value: eol})
	}
	return true
}

// isForParen reports whether the "(" at i opens a for() condition.
func isForParen(s *tokens.Stream, i int) bool {
	j := i - 1
	for j >= 0 && isPhpcsEmpty(s.At(j)) {
		j--
	}
	return j >= 0 && s.At(j).Kind == token.Keyword && strings.EqualFold(s.At(j).Value, "for")
}

// isPhpcsIgnoreComment reports whether PHP_CodeSniffer's tokenizer turns a line
// of the comment into T_PHPCS_IGNORE, which the sniff reports but never fixes.
func isPhpcsIgnoreComment(v string) bool {
	for l := range strings.Lines(v) {
		text := strings.ToLower(strings.TrimLeft(l, " \t/*#"))
		text = strings.TrimPrefix(text, "@")
		if strings.HasPrefix(text, "phpcs:ignore") && !strings.HasPrefix(text, "phpcs:ignorefile") {
			return true
		}
	}
	return false
}
