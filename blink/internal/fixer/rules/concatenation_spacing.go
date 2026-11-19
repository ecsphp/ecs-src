package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Strings/ConcatenationSpacingSniff.php
//
// ConcatenationSpacing puts exactly Spacing spaces (default 0) on both sides of
// the concat operator: "'a' . $b" -> "'a'.$b". With Spacing 0 a number next to
// the removed space is wrapped, "1 . 'x'" -> "(1).'x'", so it cannot read as a
// float. The space after a heredoc closer is kept; with IgnoreNewlines a line
// break around the operator is too.
type ConcatenationSpacing struct {
	Spacing        int
	IgnoreNewlines bool
}

func (ConcatenationSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Strings\ConcatenationSpacingSniff`
}

func (ConcatenationSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Strings/ConcatenationSpacingSniff.php"
}

func (f ConcatenationSpacing) WithConfig(config map[string]any) fixer.Fixer {
	f.Spacing = configInt(config, "spacing", f.Spacing)
	if v, ok := config["ignoreNewlines"].(bool); ok {
		f.IgnoreNewlines = v
	}
	return f
}

// concatNewline is the spacing measured on a side of a concat that crosses a line.
const concatNewline = -1

func (f ConcatenationSpacing) Fix(s *tokens.Stream) bool {
	return phpcsRunUntilStable(s, f.pass)
}

func (f ConcatenationSpacing) pass(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if t := s.At(i); t.Kind != token.Punct || t.Value != "." {
			continue
		}
		// live coding: PHP_CodeSniffer needs two tokens after the operator
		if i+2 >= s.Len() {
			continue
		}
		prev := phpcsPrevNonEmpty(s, i-1)
		ignoreBefore := prev >= 0 && isHeredocEnd(s, prev)

		before := f.concatSideSpace(s, i-1)
		after := f.concatSideSpace(s, i+1)
		if (ignoreBefore || f.concatSpaceOK(before)) && f.concatSpaceOK(after) {
			continue
		}
		padding := strings.Repeat(" ", f.Spacing)
		// after first, so i stays valid
		if after != concatNewline || !f.IgnoreNewlines {
			if f.setConcatSpace(s, i+1, i+2, padding) {
				changed = true
			}
		}
		if !ignoreBefore && (before != concatNewline || !f.IgnoreNewlines) {
			if removed, did := f.setConcatSpaceBefore(s, i, padding); did {
				changed = true
				i -= removed
			}
		}
	}
	return changed
}

func (f ConcatenationSpacing) concatSpaceOK(n int) bool {
	return n == f.Spacing || (n == concatNewline && f.IgnoreNewlines)
}

// concatSideSpace measures the whitespace at w the way PHP_CodeSniffer does:
// 0 when there is none, concatNewline when it crosses a line, else its width.
func (ConcatenationSpacing) concatSideSpace(s *tokens.Stream, w int) int {
	if !phpcsWhitespaceAt(s, w) {
		return 0
	}
	if strings.ContainsAny(s.At(w).Value, "\r\n") {
		return concatNewline
	}
	return phpcsTabWidth(s, w)
}

// setConcatSpace makes the whitespace at w (after the operator) padding, and
// wraps a number right after it when the space goes.
func (f ConcatenationSpacing) setConcatSpace(s *tokens.Stream, w, operand int, padding string) bool {
	if !phpcsWhitespaceAt(s, w) {
		if f.Spacing > 0 {
			s.InsertAt(w, token.Token{Kind: token.Whitespace, Value: padding})
			return true
		}
		return false
	}
	if padding != "" {
		s.SetValue(w, padding)
	} else {
		s.RemoveAt(w)
		operand--
	}
	if f.Spacing == 0 && operand < s.Len() {
		wrapConcatNumber(s, operand)
	}
	return true
}

// setConcatSpaceBefore makes the whitespace before the operator at op padding,
// keeping a line comment's newline. It returns how many tokens were removed.
func (f ConcatenationSpacing) setConcatSpaceBefore(s *tokens.Stream, op int, padding string) (int, bool) {
	w := op - 1
	if !phpcsWhitespaceAt(s, w) {
		if f.Spacing > 0 {
			s.InsertAt(op, token.Token{Kind: token.Whitespace, Value: padding})
			return -1, true
		}
		return 0, false
	}
	removed := 0
	switch {
	case padding != "":
		keep := phpcsCommentNewline(s, w)
		s.SetValue(w, s.At(w).Value[:keep]+padding)
	case dropPhpcsWhitespace(s, w):
		removed = 1
	}
	if f.Spacing == 0 && removed == 1 && w-1 >= 0 {
		wrapConcatNumber(s, w-1)
	}
	return removed, true
}

// wrapConcatNumber parenthesizes the number at i once; a number between two
// concats is wrapped by the first, like PHP_CodeSniffer's conflicting changesets.
func wrapConcatNumber(s *tokens.Stream, i int) {
	if t := s.At(i); t.Kind == token.Number && !strings.HasPrefix(t.Value, "(") {
		s.SetValue(i, "("+t.Value+")")
	}
}
