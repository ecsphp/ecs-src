package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

var castTypes = map[string]bool{
	"int": true, "integer": true, "bool": true, "boolean": true,
	"float": true, "double": true, "real": true, "string": true,
	"array": true, "object": true, "unset": true, "binary": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/CastSpacesFixer.php
//
// CastSpaces normalizes a type cast to no inner spaces and a single space after:
// "(int)$x" and "( int )$x" both become "(int) $x". A flat token stream cannot
// see PHP cast tokens, so casts are detected heuristically as "(" type ")" not
// preceded by a call, index or value.
type CastSpaces struct{}

func (CastSpaces) Name() string {
	return `PhpCsFixer\Fixer\CastNotation\CastSpacesFixer`
}

func (CastSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/CastNotation/CastSpacesFixer.php"
}

func (CastSpaces) Fix(s *tokens.Stream) bool {
	changed := false
	i := 0
	for i < s.Len() {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "(" {
			i++
			continue
		}

		open := i
		j := open + 1
		openWS := -1
		if j < s.Len() && s.At(j).Kind == token.Whitespace {
			openWS = j
			j++
		}
		if j >= s.Len() || !isCastType(s.At(j)) {
			i++
			continue
		}
		j++
		closeWS := -1
		if j < s.Len() && s.At(j).Kind == token.Whitespace {
			closeWS = j
			j++
		}
		if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != ")" {
			i++
			continue
		}
		closeIdx := j

		// skip when "(" follows a call, index or value: then it is grouping, not a cast
		if prev, ok := prevSignificant(s, open); ok {
			if prev.Kind == token.Ident || prev.Kind == token.Variable ||
				prev.Value == ")" || prev.Value == "]" {
				i++
				continue
			}
		}

		// mutate high index to low so earlier indices stay valid
		if closeIdx+1 < s.Len() {
			next := s.At(closeIdx + 1)
			if next.Kind == token.Whitespace {
				if !hasNewline(next.Value) && next.Value != " " {
					s.SetValue(closeIdx+1, " ")
					changed = true
				}
			} else {
				s.InsertAt(closeIdx+1, token.Token{Kind: token.Whitespace, Value: " "})
				changed = true
			}
		}
		if closeWS >= 0 {
			s.RemoveAt(closeWS)
			changed = true
		}
		if openWS >= 0 {
			s.RemoveAt(openWS)
			changed = true
		}
		i++
	}
	return changed
}

func isCastType(t token.Token) bool {
	if t.Kind != token.Ident && t.Kind != token.Keyword {
		return false
	}
	return castTypes[strings.ToLower(t.Value)]
}
