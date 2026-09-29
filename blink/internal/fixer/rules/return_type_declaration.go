package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// ReturnTypeDeclaration normalizes the return-type colon: no space before, one
// space after ("function f() : int" and "function f():int" -> "function f(): int").
// The colon is only a return type when it follows the ")" of a function parameter
// list, so ternary, case and alternative-syntax colons are left untouched.
type ReturnTypeDeclaration struct{}

func (ReturnTypeDeclaration) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\ReturnTypeDeclarationFixer`
}

func (ReturnTypeDeclaration) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/ReturnTypeDeclarationFixer.php"
}

func (ReturnTypeDeclaration) Fix(s *tokens.Stream) bool {
	changed := false
	i := 0
	for i < s.Len() {
		if s.At(i).Kind != token.Punct || s.At(i).Value != ":" {
			i++
			continue
		}
		if !isReturnTypeColon(s, i) {
			i++
			continue
		}

		// after side: exactly one space, unless a newline follows (leave multiline)
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

		// before side: no space, but keep newline-spanning whitespace intact
		if i > 0 && s.At(i-1).Kind == token.Whitespace && !hasNewline(s.At(i-1).Value) {
			s.RemoveAt(i - 1)
			changed = true
			i-- // colon shifted left by the removed whitespace
		}
		i++
	}
	return changed
}

// isReturnTypeColon reports whether the ":" at index i closes a function
// signature: its previous significant token is a ")" that matches a "(" opened
// by a function/fn declaration.
func isReturnTypeColon(s *tokens.Stream, i int) bool {
	c := prevSignificantIndex(s, i)
	if c < 0 || s.At(c).Kind != token.Punct || s.At(c).Value != ")" {
		return false
	}
	open := s.MatchBackward(c)
	if open < 0 {
		return false
	}
	p := prevSignificantIndex(s, open)
	if p < 0 {
		return false
	}
	pt := s.At(p)
	if pt.Kind == token.Keyword {
		v := strings.ToLower(pt.Value)
		return v == "function" || v == "fn"
	}
	// named function: an identifier name preceded (optionally via "&") by "function"
	if pt.Kind == token.Ident {
		q := prevSignificantIndex(s, p)
		if q >= 0 && s.At(q).Kind == token.Punct && s.At(q).Value == "&" {
			q = prevSignificantIndex(s, q)
		}
		return q >= 0 && s.At(q).Kind == token.Keyword && strings.ToLower(s.At(q).Value) == "function"
	}
	return false
}

// prevSignificantIndex returns the index of the first non-whitespace token before
// i, or -1 if there is none.
func prevSignificantIndex(s *tokens.Stream, i int) int {
	for j := i - 1; j >= 0; j-- {
		if s.Kind(j) != token.Whitespace {
			return j
		}
	}
	return -1
}
