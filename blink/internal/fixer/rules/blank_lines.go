package rules

import (
	"regexp"

	"blink/internal/token"
	"blink/internal/tokens"
)

var threeOrMoreNewlines = regexp.MustCompile(`\n{3,}`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoExtraBlankLinesFixer.php
//
// NoExtraBlankLines collapses two or more consecutive blank lines into one. It
// runs after no_whitespace_in_blank_line, so blank lines are already bare.
type NoExtraBlankLines struct{}

func (NoExtraBlankLines) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\NoExtraBlankLinesFixer`
}

func (NoExtraBlankLines) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoExtraBlankLinesFixer.php"
}

func (NoExtraBlankLines) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Whitespace {
			continue
		}
		// merge any adjacent whitespace tokens (e.g. left by import removal) so
		// a run of blank lines lives in one token the regex can collapse
		for i+1 < s.Len() && s.At(i+1).Kind == token.Whitespace {
			s.SetValue(i, s.At(i).Value+s.At(i+1).Value)
			s.RemoveAt(i + 1)
		}
		if v := threeOrMoreNewlines.ReplaceAllString(s.At(i).Value, "\n\n"); v != s.At(i).Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}
