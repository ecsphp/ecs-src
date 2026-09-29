package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/NoShortBoolCastFixer.php
//
// NoShortBoolCast rewrites the double-not short bool cast to an explicit cast:
// "!!$x" -> "(bool) $x". Two adjacent "!" Punct tokens (only whitespace between)
// are the short cast; a comment between them is left untouched.
type NoShortBoolCast struct{}

func (NoShortBoolCast) Name() string {
	return `PhpCsFixer\Fixer\CastNotation\NoShortBoolCastFixer`
}

func (NoShortBoolCast) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/NoShortBoolCastFixer.php"
}

func (NoShortBoolCast) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 1; i-- {
		if !isBang(s.At(i)) {
			continue
		}
		j := prevSignificantIndex(s, i)
		if j < 0 || !isBang(s.At(j)) {
			continue
		}

		s.ReplaceRange(j, i, []token.Token{
			{Kind: token.Punct, Value: "("},
			{Kind: token.Ident, Value: "bool"},
			{Kind: token.Punct, Value: ")"},
		})
		// ensure a single space after the new ")"
		after := j + 3
		if after < s.Len() && s.At(after).Kind != token.Whitespace {
			s.InsertAt(after, token.Token{Kind: token.Whitespace, Value: " "})
		}
		changed = true
		i = j
	}
	return changed
}

func isBang(t token.Token) bool {
	return t.Kind == token.Punct && t.Value == "!"
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/NoUnsetCastFixer.php
//
// NoUnsetCast replaces an "(unset)" cast assignment with null: "$a = (unset) $b;"
// -> "$a = null;". It applies only to the exact shape "= (unset) $var ;" (or a
// closing tag in place of ";"), since "(unset) $x" always evaluates to null.
type NoUnsetCast struct{}

func (NoUnsetCast) Name() string {
	return `PhpCsFixer\Fixer\CastNotation\NoUnsetCastFixer`
}

func (NoUnsetCast) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/NoUnsetCastFixer.php"
}

func (NoUnsetCast) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		typeIdx, closeIdx, ok := castAt(s, i)
		if !ok || strings.ToLower(s.At(typeIdx).Value) != "unset" {
			continue
		}

		assignIdx := prevSignificantIndex(s, i)
		if assignIdx < 0 || s.At(assignIdx).Kind != token.Punct || s.At(assignIdx).Value != "=" {
			continue
		}
		varIdx := nextSignificantIndex(s, closeIdx)
		if varIdx < 0 || s.At(varIdx).Kind != token.Variable {
			continue
		}
		afterVar := nextSignificantIndex(s, varIdx)
		if afterVar < 0 {
			continue
		}
		if s.At(afterVar).Kind != token.CloseTag &&
			(s.At(afterVar).Kind != token.Punct || s.At(afterVar).Value != ";") {
			continue
		}

		repl := []token.Token{{Kind: token.Ident, Value: "null"}}
		if s.At(assignIdx+1).Kind != token.Whitespace {
			repl = append([]token.Token{{Kind: token.Whitespace, Value: " "}}, repl...)
		}
		s.ReplaceRange(i, varIdx, repl)
		changed = true
	}
	return changed
}
