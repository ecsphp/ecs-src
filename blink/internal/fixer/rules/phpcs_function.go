package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsFunctionBody returns the braces of the body of the function or closure
// whose "function" keyword is at i; ok is false for a body-less (abstract or
// interface) method and for "use function" imports.
func phpcsFunctionBody(s *tokens.Stream, i int) (open, closeIdx int, ok bool) {
	if t := s.At(i); t.Kind != token.Keyword || !strings.EqualFold(t.Value, "function") {
		return 0, 0, false
	}
	j := phpcsNextNonEmpty(s, i+1)
	if j < s.Len() && s.At(j).Value == "&" {
		j = phpcsNextNonEmpty(s, j+1)
	}
	if j < s.Len() && s.At(j).Kind == token.Ident {
		j = phpcsNextNonEmpty(s, j+1)
	}
	if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != "(" {
		return 0, 0, false
	}
	for j = s.MatchForward(j); j >= 0 && j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(":
			j = s.MatchForward(j)
			if j < 0 {
				return 0, 0, false
			}
		case "{":
			if c := s.MatchForward(j); c >= 0 {
				return j, c, true
			}
			return 0, 0, false
		case ";", "}":
			return 0, 0, false
		}
	}
	return 0, 0, false
}

// phpcsNextNonEmpty is the index of the first token at or after i that is not
// whitespace or a comment, s.Len() when there is none.
func phpcsNextNonEmpty(s *tokens.Stream, i int) int {
	for i < s.Len() && isPhpcsEmpty(s.At(i)) {
		i++
	}
	return i
}

// phpcsLineBreaks counts the line breaks in v ("\r\n", "\n" or a lone "\r").
func phpcsLineBreaks(v string) int {
	return strings.Count(v, "\n") + strings.Count(v, "\r") - strings.Count(v, "\r\n")
}

// phpcsAfterLastBreak is the part of v after its last line break.
func phpcsAfterLastBreak(v string) string {
	return v[strings.LastIndexAny(v, "\r\n")+1:]
}

// phpcsThroughFirstBreak is v up to and including its first line break.
func phpcsThroughFirstBreak(v string) string {
	i := strings.IndexAny(v, "\r\n")
	if i < 0 {
		return v
	}
	if v[i] == '\r' && i+1 < len(v) && v[i+1] == '\n' {
		i++
	}
	return v[:i+1]
}
