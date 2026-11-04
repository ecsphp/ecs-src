package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NotOperatorWithSpaceFixer.php
//
// NotOperatorWithSpace ensures the logical NOT operator "!" has both a leading
// and a trailing whitespace ("!$bar" -> "! $bar", "(!true)" -> "( ! true)"). A
// space is only inserted where one is missing; existing whitespace is kept as is.
type NotOperatorWithSpace struct{}

func (NotOperatorWithSpace) Name() string {
	return `PhpCsFixer\Fixer\Operator\NotOperatorWithSpaceFixer`
}

func (NotOperatorWithSpace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/NotOperatorWithSpaceFixer.php"
}

func (NotOperatorWithSpace) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "!" {
			continue
		}
		if i+1 >= s.Len() || s.At(i+1).Kind != token.Whitespace {
			s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
		}
		if i-1 >= 0 && s.At(i-1).Kind != token.Whitespace {
			s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
			changed = true
		}
	}
	return changed
}
