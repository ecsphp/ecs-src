package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsCommentNewline is how many leading bytes of the whitespace token at w
// belong to the "//" or "#" comment before it: PHP_CodeSniffer's comment token
// ends with its newline, blink's does not.
func phpcsCommentNewline(s *tokens.Stream, w int) int {
	if w == 0 || !isLineComment(s.At(w-1)) {
		return 0
	}
	v := s.At(w).Value
	switch {
	case strings.HasPrefix(v, "\r\n"):
		return 2
	case strings.HasPrefix(v, "\n"), strings.HasPrefix(v, "\r"):
		return 1
	}
	return 0
}

// phpcsWhitespaceAt reports whether PHP_CodeSniffer sees a whitespace token at
// i, i.e. blink's whitespace there is more than a line comment's newline.
func phpcsWhitespaceAt(s *tokens.Stream, i int) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Whitespace &&
		phpcsCommentNewline(s, i) < len(s.At(i).Value)
}

// dropPhpcsWhitespace removes the whitespace at w, keeping a line comment's
// newline. It reports whether the token itself was removed.
func dropPhpcsWhitespace(s *tokens.Stream, w int) bool {
	keep := phpcsCommentNewline(s, w)
	if keep == 0 {
		s.RemoveAt(w)
		return true
	}
	s.SetValue(w, s.At(w).Value[:keep])
	return false
}

// phpcsSpaceAfter ports the shared fix of SpaceAfterCast and SpaceAfterNot:
// exactly spacing spaces between the token at op and its operand. A comment in
// between, or a line break with ignoreNewlines, is left alone.
func phpcsSpaceAfter(s *tokens.Stream, op, spacing int, ignoreNewlines bool) bool {
	next := op + 1
	for next < s.Len() && isPhpcsEmpty(s.At(next)) {
		next++
	}
	if next >= s.Len() {
		return false
	}
	ws := op + 1
	hasWS := s.At(ws).Kind == token.Whitespace
	newline := hasWS && strings.Contains(s.At(ws).Value, "\n")
	if ignoreNewlines && newline {
		return false
	}
	if next != ws && (!hasWS || next != ws+1) {
		return false
	}
	padding := strings.Repeat(" ", spacing)
	switch {
	case !hasWS:
		if spacing == 0 {
			return false
		}
		s.InsertAt(ws, token.Token{Kind: token.Whitespace, Value: padding})
	case newline || phpcsTabWidth(s, ws) != spacing:
		if spacing > 0 {
			s.SetValue(ws, padding)
		} else {
			s.RemoveAt(ws)
		}
	default:
		return false
	}
	return true
}
