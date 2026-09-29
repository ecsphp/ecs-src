package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/LinebreakAfterOpeningTagFixer.php
//
// LinebreakAfterOpeningTag moves code off the opening tag line: "<?php $x" becomes
// "<?php\n$x". It fires only for the full "<?php" tag at the start of the file with
// real code on the same line; a tag already followed by a newline, a short echo
// tag, or a lone "<?php ?>" is left untouched.
type LinebreakAfterOpeningTag struct{}

func (LinebreakAfterOpeningTag) Name() string {
	return `PhpCsFixer\Fixer\PhpTag\LinebreakAfterOpeningTagFixer`
}

func (LinebreakAfterOpeningTag) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/LinebreakAfterOpeningTagFixer.php"
}

func (LinebreakAfterOpeningTag) Fix(s *tokens.Stream) bool {
	if s.Len() < 3 {
		return false
	}
	if s.At(0).Kind != token.OpenTag || s.At(0).Value != "<?php" {
		return false
	}
	next := s.At(1)
	if next.Kind != token.Whitespace {
		return false // "<?php" not followed by a whitespace separator
	}
	if strings.Contains(next.Value, "\n") {
		return false // code already starts on a following line
	}
	// only reflow when real code (not a closing tag) follows on the same line
	if s.At(2).Kind == token.CloseTag {
		return false
	}
	s.SetValue(1, "\n")
	return true
}
