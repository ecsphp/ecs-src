package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsEolChar mirrors PHP_CodeSniffer's Common::detectLineEndings: the first
// line ending in the file ("\r\n", "\r" or "\n"), "\n" when there is none.
func phpcsEolChar(s *tokens.Stream) string {
	for i := range s.Len() {
		v := s.At(i).Value
		if j := strings.IndexAny(v, "\r\n"); j >= 0 {
			if v[j] == '\n' {
				return "\n"
			}
			if j+1 < len(v) && v[j+1] == '\n' {
				return "\r\n"
			}
			if j+1 == len(v) && i+1 < s.Len() && strings.HasPrefix(s.At(i+1).Value, "\n") {
				return "\r\n"
			}
			return "\r"
		}
	}
	return "\n"
}

// phpcsHasOpenTag reports whether the file has an open tag a sniff registered on
// it would fire for; withEcho also counts "<?=".
func phpcsHasOpenTag(s *tokens.Stream, withEcho bool) bool {
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.OpenTag && (withEcho || !strings.HasPrefix(t.Value, "<?=")) {
			return true
		}
	}
	return false
}

// phpcsLastToken is the index of the last non-empty token, -1 for an empty file.
func phpcsLastToken(s *tokens.Stream) int {
	for i := s.Len() - 1; i >= 0; i-- {
		if s.At(i).Value != "" {
			return i
		}
	}
	return -1
}
