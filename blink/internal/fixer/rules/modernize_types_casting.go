package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/ModernizeTypesCastingFixer.php
//
// ModernizeTypesCasting replaces global intval, floatval, doubleval, strval and
// boolval calls that have a single argument with the matching cast:
// "intval($x)" becomes "(int) $x". An argument of more than one token keeps
// its parentheses, and the cast is wrapped in parentheses when an array access
// or "**" follows. Risky: the functions may be overridden.
type ModernizeTypesCasting struct{}

func (ModernizeTypesCasting) Name() string {
	return `PhpCsFixer\Fixer\CastNotation\ModernizeTypesCastingFixer`
}

func (ModernizeTypesCasting) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/ModernizeTypesCastingFixer.php"
}

// modernizeCasts lists the functions in upstream's processing order with the
// type of the cast replacing each.
var modernizeCasts = []struct{ function, cast string }{
	{"intval", "int"},
	{"floatval", "float"},
	{"doubleval", "float"},
	{"strval", "string"},
	{"boolval", "bool"},
}

func (ModernizeTypesCasting) Fix(s *tokens.Stream) bool {
	changed := false
	for _, mc := range modernizeCasts {
		curr := 0
		for {
			name, open, closeIdx, ok := aliasFind(s, mc.function, curr, s.Len()-1)
			if !ok || closeIdx < 0 {
				break
			}
			curr = open
			// another argument count means a different (overriding) function
			if len(aliasArguments(s, open, closeIdx)) != 1 {
				continue
			}

			paramEnd := closeIdx
			if comma := prevMeaningfulIndex(s, closeIdx); isPunctVal(s, comma, ",") {
				removeWhitespaceSafely(s, comma, 1)
				clearTokenAt(s, comma)
				paramEnd = comma
			}
			// something more complex than a single token keeps its parentheses
			preserveParentheses := phpTokenCount(s, open+1, paramEnd-1) > 1

			after := nextMeaningfulIndex(s, closeIdx)
			wrapInParentheses := isPunctVal(s, after, "[") || isPunctVal(s, after, "{") || isPunctVal(s, after, "**")

			if prev := prevMeaningfulIndex(s, name); isPunctVal(s, prev, `\`) {
				removeWhitespaceSafely(s, prev, 1)
				clearTokenAt(s, prev)
			}

			// blink lexes a cast as "(" type ")"
			replacement := []token.Token{
				{Kind: token.Punct, Value: "("},
				{Kind: token.Ident, Value: mc.cast},
				{Kind: token.Punct, Value: ")"},
				{Kind: token.Whitespace, Value: " "},
			}
			if wrapInParentheses {
				replacement = append([]token.Token{{Kind: token.Punct, Value: "("}}, replacement...)
			}

			if !preserveParentheses {
				removeWhitespaceSafely(s, closeIdx, -1)
				clearTokenAt(s, closeIdx)
				removeWhitespaceSafely(s, open, -1)
				removeWhitespaceSafely(s, open, 1)
				clearTokenAt(s, open)
			} else {
				// the cast brings its own trailing space
				removeWhitespaceSafely(s, name, 1)
			}
			if wrapInParentheses {
				s.InsertAt(closeIdx, token.Token{Kind: token.Punct, Value: ")"})
			}
			s.ReplaceRange(name, name, replacement)
			changed = true
			// nested transformations: resume right at the cast
			curr = name
		}
	}
	return changed
}

// phpTokenCount returns how many tokens PHP's tokenizer would produce for the
// non-whitespace tokens in [from, to], as far as telling 0, 1 and "more"
// apart: a binary-prefixed string is one token, an interpolated string, a
// heredoc and a backtick string are several.
func phpTokenCount(s *tokens.Stream, from, to int) int {
	n := 0
	for i := from; i <= to; i++ {
		t := s.At(i)
		switch t.Kind {
		case token.Whitespace:
			continue
		case token.Ident:
			if end, ok := constantStringAt(s, i); ok && end == i+1 {
				continue // the "b" prefix; the string counts below
			}
		case token.String:
			if !aliasIsConstantString(t) {
				n += 2
				continue
			}
		}
		n++
	}
	return n
}
