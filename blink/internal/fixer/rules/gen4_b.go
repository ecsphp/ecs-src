package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// isLoneDollar reports whether the token is the bare "$" that introduces a
// variable-variable ("$$foo" lexes as a Variable "$" followed by a Variable
// "$foo"). It is not itself a real variable.
func isLoneDollar(t token.Token) bool {
	return t.Kind == token.Variable && t.Value == "$"
}

// isRealVariable reports whether the token is a concrete "$name" variable rather
// than the bare "$" of a variable-variable.
func isRealVariable(t token.Token) bool {
	return t.Kind == token.Variable && len(t.Value) > 1
}

// isDynamicVarPrefix reports whether the significant token at p makes the
// variable after it an indirect (dynamic) access that should be braced: the bare
// "$" of a variable-variable, or an object operator ("->" / "?->") introducing a
// dynamic property/method name.
func isDynamicVarPrefix(t token.Token) bool {
	if isLoneDollar(t) {
		return true
	}
	return t.Kind == token.Punct && (t.Value == "->" || t.Value == "?->")
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ExplicitIndirectVariableFixer.php
//
// ExplicitIndirectVariable braces indirect variables so their meaning is
// explicit: "$$foo" -> "${$foo}", "$$foo['bar']" -> "${$foo}['bar']",
// "$foo->$bar" -> "$foo->{$bar}". Only the "$name" directly after a bare "$" or
// an object operator is wrapped; any following "[...]" or "->x" stays outside the
// braces, so the rewrite is a pure clarification with no change in behaviour.
type ExplicitIndirectVariable struct{}

func (ExplicitIndirectVariable) Name() string {
	return `PhpCsFixer\Fixer\LanguageConstruct\ExplicitIndirectVariableFixer`
}

func (ExplicitIndirectVariable) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/LanguageConstruct/ExplicitIndirectVariableFixer.php"
}

func (ExplicitIndirectVariable) Fix(s *tokens.Stream) bool {
	changed := false
	// Right-to-left so the two inserted braces (always at/after the current
	// index) never disturb positions still to be scanned.
	for i := s.Len() - 1; i >= 0; i-- {
		if !isRealVariable(s.At(i)) {
			continue
		}
		p := prevSignificantIndex(s, i)
		if p < 0 || !isDynamicVarPrefix(s.At(p)) {
			continue
		}
		s.InsertAt(i+1, token.Token{Kind: token.Punct, Value: "}"})
		s.InsertAt(i, token.Token{Kind: token.Punct, Value: "{"})
		changed = true
	}
	return changed
}
