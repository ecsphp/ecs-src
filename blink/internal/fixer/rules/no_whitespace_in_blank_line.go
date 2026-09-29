package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoWhitespaceInBlankLineFixer.php
//
// NoWhitespaceInBlankLine removes spaces and tabs from otherwise-empty lines.
// It only clears interior blank lines inside a whitespace run; the last segment
// (indentation of the next code line) and the first segment (trailing
// whitespace of the previous line) are left to their own fixers.
type NoWhitespaceInBlankLine struct{}

func (NoWhitespaceInBlankLine) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\NoWhitespaceInBlankLineFixer`
}

func (NoWhitespaceInBlankLine) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoWhitespaceInBlankLineFixer.php"
}

func (NoWhitespaceInBlankLine) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Whitespace {
			continue
		}
		segments := strings.Split(t.Value, "\n")
		if len(segments) <= 2 {
			continue // no interior blank line
		}
		touched := false
		for j := 1; j < len(segments)-1; j++ {
			if segments[j] != "" {
				segments[j] = ""
				touched = true
			}
		}
		if touched {
			s.SetValue(i, strings.Join(segments, "\n"))
			changed = true
		}
	}
	return changed
}
