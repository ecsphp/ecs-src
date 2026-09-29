package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ReturnNotation/SimplifiedNullReturnFixer.php
//
// SimplifiedNullReturn turns "return null;" into "return;", except when the
// enclosing function declares a nullable return type (?T, a union containing
// null, or mixed) - there the explicit null is kept.
type SimplifiedNullReturn struct{}

func (SimplifiedNullReturn) Name() string {
	return `PhpCsFixer\Fixer\ReturnNotation\SimplifiedNullReturnFixer`
}

func (SimplifiedNullReturn) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ReturnNotation/SimplifiedNullReturnFixer.php"
}

func (SimplifiedNullReturn) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "return" {
			continue
		}
		if memberPrev(s, i) {
			continue // ->return / ::return member access
		}
		// expect exactly "return null ;"
		j := nextSignificantIndex(s, i)
		if j < 0 || !isNullLiteral(s.At(j)) {
			continue
		}
		k := nextSignificantIndex(s, j)
		if k < 0 || s.At(k).Kind != token.Punct || s.At(k).Value != ";" {
			continue
		}
		if enclosingReturnTypeNullable(s, i) {
			continue
		}
		// drop everything between "return" and ";" -> "return;"
		for x := k - 1; x > i; x-- {
			s.RemoveAt(x)
		}
		changed = true
	}
	return changed
}

// enclosingReturnTypeNullable reports whether the innermost function enclosing
// position `at` declares a nullable return type.
func enclosingReturnTypeNullable(s *tokens.Stream, at int) bool {
	depth := 0
	for j := at - 1; j >= 0; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case "}":
			depth++
		case "{":
			if depth > 0 {
				depth--
				continue
			}
			kind, kw := classifyBrace(s, j)
			switch kind {
			case braceFunctionDecl, braceClosure:
				return funcReturnTypeNullable(s, kw, j)
			case braceClassLike:
				return false // a return cannot sit directly in a class body
			default:
				// inner control/free block; the function is further out
			}
		}
	}
	return false
}

// funcReturnTypeNullable inspects the signature of the function whose keyword is
// at kw and whose body opens at brace, reporting a nullable return type.
func funcReturnTypeNullable(s *tokens.Stream, kw, brace int) bool {
	paramsOpen := -1
	for x := kw + 1; x < brace; x++ {
		if s.At(x).Kind == token.Punct && s.At(x).Value == "(" {
			paramsOpen = x
			break
		}
	}
	if paramsOpen < 0 {
		return false
	}
	paramsClose := s.MatchForward(paramsOpen)
	if paramsClose < 0 || paramsClose >= brace {
		return false
	}
	c := nextSignificantIndex(s, paramsClose)
	// a closure may carry a "use (...)" clause between the params and the return type
	if c >= 0 && s.At(c).Kind == token.Keyword && strings.EqualFold(s.At(c).Value, "use") {
		uo := nextSignificantIndex(s, c)
		if uo < 0 || s.At(uo).Kind != token.Punct || s.At(uo).Value != "(" {
			return false
		}
		uc := s.MatchForward(uo)
		if uc < 0 || uc >= brace {
			return false
		}
		c = nextSignificantIndex(s, uc)
	}
	if c < 0 || s.At(c).Kind != token.Punct || s.At(c).Value != ":" {
		return false // no return type declared
	}
	var b strings.Builder
	for x := c + 1; x < brace; x++ {
		tk := s.At(x)
		switch tk.Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		b.WriteString(tk.Value)
	}
	typ := strings.ToLower(b.String())
	if typ == "" {
		return false
	}
	if strings.HasPrefix(typ, "?") {
		return true
	}
	for _, part := range strings.FieldsFunc(typ, func(r rune) bool { return r == '|' || r == '&' }) {
		part = strings.Trim(part, "()")
		if part == "null" || part == "mixed" {
			return true
		}
	}
	return false
}
