package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NotOperatorWithSuccessorSpaceFixer.php
//
// NotOperatorWithSuccessorSpace ensures exactly one space after the "!" operator
// ("!$a" -> "! $a").
type NotOperatorWithSuccessorSpace struct{}

func (NotOperatorWithSuccessorSpace) Name() string {
	return `PhpCsFixer\Fixer\Operator\NotOperatorWithSuccessorSpaceFixer`
}

func (NotOperatorWithSuccessorSpace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NotOperatorWithSuccessorSpaceFixer.php"
}

func (NotOperatorWithSuccessorSpace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "!" {
			continue
		}
		if i+1 >= s.Len() {
			continue
		}
		if s.At(i+1).Kind == token.Whitespace {
			if s.At(i+1).Value != " " {
				s.SetValue(i+1, " ")
				changed = true
			}
			continue
		}
		s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
		changed = true
	}
	return changed
}
