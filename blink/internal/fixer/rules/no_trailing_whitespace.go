package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoTrailingWhitespaceFixer.php
//
// NoTrailingWhitespace removes spaces and tabs at the end of a line. Like
// PHP-CS-Fixer, it trims a whitespace token only up to its first line break,
// so a line holding nothing but whitespace is left alone, and the same goes for
// a single-line token unless it is the file's last.
type NoTrailingWhitespace struct{}

func (NoTrailingWhitespace) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\NoTrailingWhitespaceFixer`
}

func (NoTrailingWhitespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoTrailingWhitespaceFixer.php"
}

func (NoTrailingWhitespace) Fix(s *tokens.Stream) bool {
	changed := false
	last := s.Len() - 1
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Whitespace {
			continue
		}
		head, rest := t.Value, ""
		if nl := strings.IndexAny(t.Value, "\r\n"); nl >= 0 {
			head, rest = t.Value[:nl], t.Value[nl:]
		} else if i != last {
			continue
		}
		// an open tag that already ends the line leaves this token's head as
		// the next line's indentation
		if i > 0 && s.At(i-1).Kind == token.OpenTag && strings.HasSuffix(s.At(i-1).Value, "\n") {
			continue
		}
		if trimmed := strings.TrimRight(head, " \t"); trimmed != head {
			s.SetValue(i, trimmed+rest)
			changed = true
		}
	}
	return changed
}
