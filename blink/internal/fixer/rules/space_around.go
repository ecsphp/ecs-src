package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

func hasNewline(s string) bool { return strings.ContainsAny(s, "\n\r") }

// nextSignificantValue returns the value of the first non-whitespace token after
// index i, or "" if none.
func nextSignificantValue(s *tokens.Stream, i int) string {
	for j := i + 1; j < s.Len(); j++ {
		if s.At(j).Kind != token.Whitespace {
			return s.At(j).Value
		}
	}
	return ""
}

// prevSignificant returns the first non-whitespace token before index i.
func prevSignificant(s *tokens.Stream, i int) (token.Token, bool) {
	for j := i - 1; j >= 0; j-- {
		if s.At(j).Kind != token.Whitespace {
			return s.At(j), true
		}
	}
	return token.Token{}, false
}

// memberPrev reports whether the token before index i is an object/static
// access operator, meaning the token at i is a property or method name.
func memberPrev(s *tokens.Stream, i int) bool {
	if prev, ok := prevSignificant(s, i); ok {
		return prev.Value == "->" || prev.Value == "?->" || prev.Value == "::"
	}
	return false
}

// isOperand reports whether a token can be the operand of ++/-- (a variable,
// name, or a closing ) or ]).
func isOperand(t token.Token) bool {
	if t.Kind == token.Variable || t.Kind == token.Ident {
		return true
	}
	return t.Kind == token.Punct && (t.Value == ")" || t.Value == "]")
}

// castAt detects a cast at an opening "(" and returns the type token index and
// the closing ")" index. It reuses CastSpaces' guard so grouping, calls and
// index accesses are not mistaken for casts.
func castAt(s *tokens.Stream, open int) (typeIdx, closeIdx int, ok bool) {
	if s.At(open).Kind != token.Punct || s.At(open).Value != "(" {
		return 0, 0, false
	}
	j := open + 1
	if j < s.Len() && s.At(j).Kind == token.Whitespace {
		j++
	}
	if j >= s.Len() || !isCastType(s.At(j)) {
		return 0, 0, false
	}
	typeIdx = j
	j++
	if j < s.Len() && s.At(j).Kind == token.Whitespace {
		j++
	}
	if j >= s.Len() || s.At(j).Kind != token.Punct || s.At(j).Value != ")" {
		return 0, 0, false
	}
	if prev, ok := prevSignificant(s, open); ok {
		if prev.Kind == token.Ident || prev.Kind == token.Variable ||
			prev.Value == ")" || prev.Value == "]" {
			return 0, 0, false
		}
	}
	return typeIdx, j, true
}

// insideDeclareArgs reports whether index i sits inside a declare( ... ) header,
// so binary-operator spacing leaves declare_equal_normalize to handle its "=".
func insideDeclareArgs(s *tokens.Stream, i int) bool {
	depth := 0
	for j := i - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")":
			depth++
		case "(":
			if depth == 0 {
				if prev, ok := prevSignificant(s, j); ok {
					return prev.Kind == token.Keyword && strings.EqualFold(prev.Value, "declare")
				}
				return false
			}
			depth--
		case ";":
			if depth == 0 {
				return false
			}
		}
	}
	return false
}

// normalizeSpaceAround forces exactly one space on both sides of every token the
// target predicate selects. Whitespace spanning a newline is left alone so
// intentional alignment survives.
func normalizeSpaceAround(s *tokens.Stream, target func(*tokens.Stream, int) bool) bool {
	changed := false
	i := 0
	for i < s.Len() {
		if !target(s, i) {
			i++
			continue
		}

		if i > 0 {
			prev := s.At(i - 1)
			if prev.Kind == token.Whitespace {
				if !hasNewline(prev.Value) && prev.Value != " " {
					s.SetValue(i-1, " ")
					changed = true
				}
			} else {
				s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
				i++ // operator shifted right by the inserted space
				changed = true
			}
		}

		if i+1 < s.Len() {
			next := s.At(i + 1)
			if next.Kind == token.Whitespace {
				if !hasNewline(next.Value) && next.Value != " " {
					s.SetValue(i+1, " ")
					changed = true
				}
			} else {
				s.InsertAt(i+1, token.Token{Kind: token.Whitespace, Value: " "})
				changed = true
			}
		}
		i++
	}
	return changed
}
