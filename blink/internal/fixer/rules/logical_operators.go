package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/LogicalOperatorsFixer.php
//
// LogicalOperators replaces the "and" and "or" operators (T_LOGICAL_AND,
// T_LOGICAL_OR) with "&&" and "||". Risky: they have a lower precedence.
type LogicalOperators struct{}

func (LogicalOperators) Name() string {
	return `PhpCsFixer\Fixer\Operator\LogicalOperatorsFixer`
}

func (LogicalOperators) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/LogicalOperatorsFixer.php"
}

func (LogicalOperators) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword {
			continue
		}
		var op string
		switch strings.ToLower(t.Value) {
		case "and":
			op = "&&"
		case "or":
			op = "||"
		default:
			continue
		}
		if logicalOperatorIsName(s, i) {
			continue
		}
		s.Set(i, token.Token{Kind: token.Punct, Value: op, Pos: t.Pos})
		changed = true
	}
	return changed
}

// logicalOperatorIsName reports an "and"/"or" that PHP reads as a name, not an
// operator: a named argument, a constant, an enum case, a trait alias or a
// by-reference method name. After "->", "::" and "function" blink already lexes
// it as an identifier.
func logicalOperatorIsName(s *tokens.Stream, i int) bool {
	next := sigNext(s, i)
	if isPunctVal(s, next, ":") || isPunctVal(s, next, "=") || (next >= 0 && kwIs(s.At(next), "as")) {
		return true
	}
	prev := sigPrev(s, i)
	if prev < 0 {
		return false
	}
	if kwIs(s.At(prev), "case") || kwIs(s.At(prev), "as") {
		return true
	}
	prevPrev := sigPrev(s, prev)
	if prevPrev < 0 {
		return false
	}
	if isPunctVal(s, prev, "&") {
		return kwIs(s.At(prevPrev), "function")
	}
	return s.At(prev).Kind == token.Keyword && isModifierKeyword(s.At(prev).Value) && kwIs(s.At(prevPrev), "as")
}
