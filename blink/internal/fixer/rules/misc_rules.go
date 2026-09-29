package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/StandardizeNotEqualsFixer.php
//
// StandardizeNotEquals rewrites the "<>" operator to "!=".
type StandardizeNotEquals struct{}

func (StandardizeNotEquals) Name() string {
	return `PhpCsFixer\Fixer\Operator\StandardizeNotEqualsFixer`
}

func (StandardizeNotEquals) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/StandardizeNotEqualsFixer.php"
}

func (StandardizeNotEquals) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		if s.At(i).Kind == token.Punct && s.At(i).Value == "<>" {
			s.SetValue(i, "!=")
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/NoEmptyStatementFixer.php
//
// NoEmptyStatement removes redundant semicolons ("$a = 1;;" -> "$a = 1;"). The
// empty statements of a for-header (inside "(...)") are left alone.
type NoEmptyStatement struct{}

func (NoEmptyStatement) Name() string {
	return `PhpCsFixer\Fixer\Semicolon\NoEmptyStatementFixer`
}

func (NoEmptyStatement) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Semicolon/NoEmptyStatementFixer.php"
}

func (NoEmptyStatement) Fix(s *tokens.Stream) bool {
	changed := false
	depth := 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "(", "[":
			depth++
		case ")", "]":
			depth--
		case ";":
			if depth != 0 {
				continue
			}
			if prev, ok := prevSignificant(s, i); ok && prev.Value == ";" {
				s.RemoveAt(i)
				i--
				changed = true
			}
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/LineEndingFixer.php
//
// LineEnding normalizes CRLF line endings to LF in whitespace and comments.
// String contents (including heredoc) are left untouched.
type LineEnding struct{}

func (LineEnding) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\LineEndingFixer`
}

func (LineEnding) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/LineEndingFixer.php"
}

func (LineEnding) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Whitespace && t.Kind != token.Comment && t.Kind != token.DocComment {
			continue
		}
		if v := strings.ReplaceAll(t.Value, "\r\n", "\n"); v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}
