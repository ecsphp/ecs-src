package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// removeWhitespaceSafely ports Tokens::removeWhitespaceSafely (dir -1 for
// removeLeadingWhitespace, 1 for removeTrailingWhitespace). After a "//" or "#"
// comment, the line break that ends the comment stays.
func removeWhitespaceSafely(s *tokens.Stream, index, dir int) {
	w := nonEmptySibling(s, index, dir)
	if w < 0 || s.At(w).Kind != token.Whitespace {
		return
	}
	newContent := ""
	if w-1 >= 0 && s.At(w-1).Kind == token.Comment && !strings.HasPrefix(s.At(w-1).Value, "/*") {
		v := s.At(w).Value
		nl := leadingLineBreak(v)
		if nl != "" && len(v) == len(nl) {
			return
		}
		newContent = nl
	}
	if newContent == "" {
		clearTokenAt(s, w)
	} else {
		s.SetValue(w, newContent)
	}
}

func leadingLineBreak(v string) string {
	switch {
	case strings.HasPrefix(v, "\r\n"):
		return "\r\n"
	case strings.HasPrefix(v, "\n"):
		return "\n"
	case strings.HasPrefix(v, "\r"):
		return "\r"
	}
	return ""
}
