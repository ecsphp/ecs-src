package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/FullOpeningTagFixer.php
//
// FullOpeningTag replaces the short open tag "<?" with "<?php". "<?=" is left
// alone.
type FullOpeningTag struct{}

func (FullOpeningTag) Name() string {
	return `PhpCsFixer\Fixer\PhpTag\FullOpeningTagFixer`
}

func (FullOpeningTag) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/FullOpeningTagFixer.php"
}

func (FullOpeningTag) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.OpenTag || t.Value != "<?" {
			continue
		}
		s.SetValue(i, "<?php")
		// keep a separator so "<?$x" does not become "<?php$x"
		if i+1 < s.Len() && s.At(i+1).Kind != token.Whitespace {
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
		}
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/NoClosingTagFixer.php
//
// NoClosingTag removes the trailing "?>" from a file that is only PHP (no inline
// HTML), leaving a single trailing newline.
type NoClosingTag struct{}

func (NoClosingTag) Name() string {
	return `PhpCsFixer\Fixer\PhpTag\NoClosingTagFixer`
}

func (NoClosingTag) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/NoClosingTagFixer.php"
}

func (NoClosingTag) Fix(s *tokens.Stream) bool {
	last := -1
	for i := range s.Len() {
		t := s.At(i)
		// real inline HTML means a templating file; whitespace-only inline HTML
		// (e.g. the newline the lexer emits after "?>") does not count
		if t.Kind == token.InlineHTML && strings.TrimSpace(t.Value) != "" {
			return false
		}
		if t.Kind == token.CloseTag {
			last = i
		}
	}
	if last < 0 {
		return false
	}
	// only whitespace (or whitespace-only inline HTML) may follow the final "?>"
	for j := last + 1; j < s.Len(); j++ {
		k := s.At(j).Kind
		if k == token.Whitespace {
			continue
		}
		if k == token.InlineHTML && strings.TrimSpace(s.At(j).Value) == "" {
			continue
		}
		return false
	}
	for k := s.Len() - 1; k >= last; k-- {
		s.RemoveAt(k)
	}
	// normalize EOF to exactly one trailing newline (idempotent with eof rule)
	if s.Len() > 0 {
		li := s.Len() - 1
		if s.At(li).Kind == token.Whitespace {
			s.SetValue(li, "\n")
		} else {
			s.InsertAt(s.Len(), token.Token{Kind: token.Whitespace, Value: "\n"})
		}
	}
	return true
}
