package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/BlankLineAfterOpeningTagFixer.php
//
// BlankLineAfterOpeningTag ensures one blank line after the opening "<?php" tag
// when code starts on a following line: "<?php\n$x" becomes "<?php\n\n$x". Code
// on the same line as the tag is left untouched.
type BlankLineAfterOpeningTag struct{}

func (BlankLineAfterOpeningTag) Name() string {
	return `PhpCsFixer\Fixer\PhpTag\BlankLineAfterOpeningTagFixer`
}

func (BlankLineAfterOpeningTag) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpTag/BlankLineAfterOpeningTagFixer.php"
}

func (BlankLineAfterOpeningTag) Fix(s *tokens.Stream) bool {
	if s.Len() < 3 {
		return false
	}
	if s.At(0).Kind != token.OpenTag || s.At(0).Value != "<?php" {
		return false
	}
	ws := s.At(1)
	if ws.Kind != token.Whitespace {
		return false
	}
	if !strings.HasPrefix(ws.Value, "\n") || strings.HasPrefix(ws.Value, "\n\n") {
		return false
	}
	s.SetValue(1, "\n"+ws.Value)
	return true
}
