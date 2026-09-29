package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/ElseifFixer.php
//
// Elseif merges "else if" into "elseif".
type Elseif struct{}

func (Elseif) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\ElseifFixer`
}

func (Elseif) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/ElseifFixer.php"
}

func (Elseif) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "else" {
			continue
		}
		if i+2 < s.Len() &&
			s.At(i+1).Kind == token.Whitespace && !hasNewline(s.At(i+1).Value) &&
			s.At(i+2).Kind == token.Keyword && strings.ToLower(s.At(i+2).Value) == "if" {
			s.SetValue(i, "elseif")
			s.RemoveAt(i + 2) // remove "if" first (higher index)
			s.RemoveAt(i + 1) // then the whitespace
			changed = true
		}
	}
	return changed
}
