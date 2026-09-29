package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ReturnNotation/NoUselessReturnFixer.php
//
// NoUselessReturn removes a bare "return;" that is the last statement of a
// function, method or closure body.
type NoUselessReturn struct{}

func (NoUselessReturn) Name() string {
	return `PhpCsFixer\Fixer\ReturnNotation\NoUselessReturnFixer`
}

func (NoUselessReturn) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ReturnNotation/NoUselessReturnFixer.php"
}

func (NoUselessReturn) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Keyword || !strings.EqualFold(s.At(i).Value, "return") {
			continue
		}
		if prev, ok := prevSignificant(s, i); ok && prev.Kind == token.Punct {
			switch prev.Value {
			case "->", "?->", "::":
				continue
			}
		}
		semi := nextSignificantIndex(s, i)
		if semi < 0 || s.At(semi).Kind != token.Punct || s.At(semi).Value != ";" {
			continue
		}
		after := nextSignificantIndex(s, semi)
		if after < 0 || s.At(after).Kind != token.Punct || s.At(after).Value != "}" {
			continue
		}
		open := s.MatchBackward(after)
		if open < 0 {
			continue
		}
		if kind, _ := classifyBrace(s, open); kind != braceFunctionDecl && kind != braceClosure {
			continue
		}
		for k := semi; k >= i; k-- {
			s.RemoveAt(k)
		}
		i--
		changed = true
	}
	return changed
}
