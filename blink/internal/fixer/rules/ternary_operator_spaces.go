package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// TernaryOperatorSpaces ensures a single space around the ternary "?" and its
// matching ":" ("$a?$b:$c" -> "$a ? $b : $c"). The elvis "?:" keeps the "?" and
// ":" adjacent (one space before "?", one after ":"). Nullable types ("?int")
// and null-coalescing ("??", "?->") are left untouched.
type TernaryOperatorSpaces struct{}

func (TernaryOperatorSpaces) Name() string {
	return `PhpCsFixer\Fixer\Operator\TernaryOperatorSpacesFixer`
}

func (TernaryOperatorSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/TernaryOperatorSpacesFixer.php"
}

// sides records whether the before and after side of an operator token needs a
// single space.
type sides struct{ before, after bool }

func (TernaryOperatorSpaces) Fix(s *tokens.Stream) bool {
	mark := map[int]sides{}
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != "?" {
			continue
		}
		// a "?" in type position (return type ": ?int", param "(?int $x)",
		// union "int|?string") is a nullable marker, never a ternary
		if prev, ok := prevSignificant(s, i); ok {
			switch prev.Value {
			case ":", "(", ",", "|", "&":
				continue
			}
		}
		m := matchTernaryColon(s, i)
		if m < 0 {
			continue // nullable type or coalescing "?": not a ternary
		}
		if ternaryIsElvis(s, i, m) {
			mark[i] = sides{before: true}
			mark[m] = sides{after: true}
		} else {
			mark[i] = sides{before: true, after: true}
			mark[m] = sides{before: true, after: true}
		}
	}
	if len(mark) == 0 {
		return false
	}

	changed := false
	// Right-to-left so insertions at higher indices never shift a token that a
	// lower original index still refers to.
	for i := s.Len() - 1; i >= 0; i-- {
		sp, ok := mark[i]
		if !ok {
			continue
		}
		if sp.after && i+1 < s.Len() {
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
		if sp.before && i > 0 {
			prev := s.At(i - 1)
			if prev.Kind == token.Whitespace {
				if !hasNewline(prev.Value) && prev.Value != " " {
					s.SetValue(i-1, " ")
					changed = true
				}
			} else {
				s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
				changed = true
			}
		}
	}
	return changed
}

// matchTernaryColon returns the index of the ":" matching the ternary "?" at i,
// or -1 when there is none (making the "?" a nullable-type marker). Nested
// ternaries and bracketed groups are skipped; the search stops at the end of the
// statement.
func matchTernaryColon(s *tokens.Stream, i int) int {
	depth := 0   // nested ternary depth
	bracket := 0 // (), [], {} nesting
	for j := i + 1; j < s.Len(); j++ {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "{":
			if bracket == 0 {
				return -1 // a ternary cannot span a block; "?" was a type marker
			}
			bracket++
		case "(", "[":
			bracket++
		case ")", "]", "}":
			if bracket == 0 {
				return -1 // closes an enclosing group before any ternary ":"
			}
			bracket--
		case ";":
			if bracket == 0 {
				return -1 // statement ended, no ternary
			}
		case "?":
			if bracket == 0 {
				depth++
			}
		case ":":
			if bracket == 0 {
				if depth == 0 {
					return j
				}
				depth--
			}
		}
	}
	return -1
}

// ternaryIsElvis reports whether the "?" at i and its colon at m form an elvis
// operator, i.e. there is no operand between them.
func ternaryIsElvis(s *tokens.Stream, i, m int) bool {
	for k := i + 1; k < m; k++ {
		if s.At(k).Kind != token.Whitespace {
			return false
		}
	}
	return true
}
