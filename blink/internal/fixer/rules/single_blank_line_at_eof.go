package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/SingleBlankLineAtEofFixer.php
//
// SingleBlankLineAtEndOfFile ensures the file ends with exactly one newline.
type SingleBlankLineAtEndOfFile struct{}

func (SingleBlankLineAtEndOfFile) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\SingleBlankLineAtEofFixer`
}

func (SingleBlankLineAtEndOfFile) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/SingleBlankLineAtEofFixer.php"
}

func (SingleBlankLineAtEndOfFile) Fix(s *tokens.Stream) bool {
	if s.Len() == 0 {
		return false
	}
	last := s.Len() - 1
	t := s.At(last)
	// a file ending in inline HTML or a tag is left alone (php-cs-fixer skips
	// T_INLINE_HTML / T_CLOSE_TAG / T_OPEN_TAG), so a template keeps its ending
	if t.Kind == token.InlineHTML || t.Kind == token.CloseTag || t.Kind == token.OpenTag {
		return false
	}
	if t.Kind == token.Whitespace {
		// the final whitespace token is entirely trailing; collapse to one \n
		if t.Value != "\n" {
			s.SetValue(last, "\n")
			return true
		}
		return false
	}
	// file does not end in whitespace: append a single newline, unless the last
	// token already carries a trailing newline (e.g. a comment)
	if strings.HasSuffix(t.Value, "\n") {
		return false
	}
	s.InsertAt(s.Len(), token.Token{Kind: token.Whitespace, Value: "\n"})
	return true
}
