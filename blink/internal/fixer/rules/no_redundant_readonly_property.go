package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoRedundantReadonlyPropertyFixer.php
//
// NoRedundantReadonlyProperty removes the "readonly" modifier from a property
// (or a promoted constructor property) of a readonly class, where it is implied.
type NoRedundantReadonlyProperty struct{}

func (NoRedundantReadonlyProperty) Name() string {
	return `PhpCsFixer\Fixer\ClassNotation\NoRedundantReadonlyPropertyFixer`
}

func (NoRedundantReadonlyProperty) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ClassNotation/NoRedundantReadonlyPropertyFixer.php"
}

func (NoRedundantReadonlyProperty) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if !taIsClassKw(s, i) || !nrrpClassIsReadonly(s, i) {
			continue
		}
		open := classyBodyOpen(s, i)
		if open < 0 {
			continue
		}
		for _, e := range classBodyElements(s, open) {
			if e.kind != "property" && e.kind != "promoted_property" {
				continue
			}
			if nrrpRemoveReadonly(s, e.index) {
				changed = true
			}
		}
	}
	return changed
}

// nrrpClassIsReadonly mirrors TokensAnalyzer::getClassyModifiers()['readonly']:
// a "readonly" keyword sits in the modifier run before the class keyword.
func nrrpClassIsReadonly(s *tokens.Stream, classKw int) bool {
	for j := sigPrev(s, classKw); j >= 0; j = sigPrev(s, j) {
		switch {
		case kwIs(s.At(j), "readonly"):
			return true
		case kwIs(s.At(j), "final"), kwIs(s.At(j), "abstract"):
			continue
		}
		return false
	}
	return false
}

// nrrpRemoveReadonly walks the modifiers before a property's defining variable
// and clears the first "readonly" it finds, with the whitespace after it.
func nrrpRemoveReadonly(s *tokens.Stream, memberIndex int) bool {
	for j := prevMeaningfulIndex(s, memberIndex); j >= 0; j = prevMeaningfulIndex(s, j) {
		role := modifierRole(s, j, true)
		if role == "" {
			return false
		}
		if role == "readonly" {
			// clear the whitespace right after readonly, then readonly itself,
			// keeping token indices stable (cleared tokens render as nothing)
			if j+1 < s.Len() && s.At(j+1).Kind == token.Whitespace {
				s.SetValue(j+1, "")
			}
			s.Set(j, token.Token{Kind: token.Whitespace, Value: ""})
			return true
		}
	}
	return false
}
