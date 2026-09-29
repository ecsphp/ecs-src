package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/YodaStyleFixer.php
//
// YodaStyle here follows ECS's common-set configuration (equal=false,
// identical=false, less_and_greater=false): non-yoda. When a constant is on the
// left of an == / === / != / !== comparison and a variable expression on the
// right, they are swapped so the variable comes first ("null === $x" -> "$x ===
// null"). "<"/">" comparisons are left untouched.
type YodaStyle struct{}

func (YodaStyle) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\YodaStyleFixer`
}

func (YodaStyle) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/YodaStyleFixer.php"
}

func yodaOp(v string) bool {
	switch v {
	case "==", "===", "!=", "!==", "<", ">", "<=", ">=":
		return true
	}
	return false
}

// yodaMirror flips a relational operator when its operands are swapped, so
// "5 > $z" becomes "$z < 5". Equality operators are symmetric and unchanged.
func yodaMirror(v string) string {
	switch v {
	case "<":
		return ">"
	case ">":
		return "<"
	case "<=":
		return ">="
	case ">=":
		return "<="
	}
	return v
}

func isYodaLiteral(t token.Token) bool {
	switch t.Kind {
	case token.Number, token.String:
		return true
	case token.Ident:
		return true
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "true", "false", "null":
			return true
		}
	}
	return false
}

func (YodaStyle) Fix(s *tokens.Stream) bool {
	type swap struct{ ls, le, rs, re int }
	var swaps []swap
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || !yodaOp(t.Value) {
			continue
		}
		// left must be a bounded constant
		ls, le, ok := leftLiteralOperand(s, i)
		if !ok || !isLeftBoundary(s, prevSignificantIndex(s, ls)) {
			continue
		}
		// right must be a variable expression (not itself a constant), bounded
		rs := nextSignificantIndex(s, i)
		if rs < 0 {
			continue
		}
		re := rightPrimaryEnd(s, rs)
		if re < 0 {
			continue
		}
		if rs == re && isYodaLiteral(s.At(rs)) {
			continue // both sides constant
		}
		if !isRightBoundary(s, nextSignificantIndex(s, re)) {
			continue
		}
		swaps = append(swaps, swap{ls, le, rs, re})
	}
	if len(swaps) == 0 {
		return false
	}
	for _, sw := range slices.Backward(swaps) {
		left := append([]token.Token(nil), s.Tokens()[sw.ls:sw.le+1]...)
		mid := append([]token.Token(nil), s.Tokens()[sw.le+1:sw.rs]...)
		right := append([]token.Token(nil), s.Tokens()[sw.rs:sw.re+1]...)
		for k := range mid {
			if mid[k].Kind == token.Punct {
				mid[k].Value = yodaMirror(mid[k].Value)
			}
		}
		repl := append(append(append([]token.Token(nil), right...), mid...), left...)
		s.ReplaceRange(sw.ls, sw.re, repl)
	}
	return true
}

// leftLiteralOperand returns the span of a constant left operand ending just
// before the comparison at op: a plain literal, signed number, empty array,
// bare constant, or "Name::class".
func leftLiteralOperand(s *tokens.Stream, op int) (int, int, bool) {
	le := prevSignificantIndex(s, op)
	if le < 0 {
		return 0, 0, false
	}
	t := s.At(le)
	// "]" of an empty array "[]"
	if t.Kind == token.Punct && t.Value == "]" {
		o := s.MatchBackward(le)
		if o >= 0 && nextSignificantIndex(s, o) == le {
			return o, le, true
		}
		return 0, 0, false
	}
	// "class" of "Name::class"
	if t.Kind == token.Ident && strings.EqualFold(t.Value, "class") {
		p := prevSignificantIndex(s, le)
		if p >= 0 && s.At(p).Value == "::" {
			n := prevSignificantIndex(s, p)
			if n >= 0 && s.At(n).Kind == token.Ident {
				return n, le, true
			}
		}
		return 0, 0, false
	}
	if isYodaLiteral(t) {
		// signed number "-1"
		if t.Kind == token.Number {
			p := prevSignificantIndex(s, le)
			if p >= 0 && s.At(p).Kind == token.Punct && (s.At(p).Value == "-" || s.At(p).Value == "+") {
				// only treat as sign when what precedes the sign is a boundary
				if isLeftBoundary(s, prevSignificantIndex(s, p)) {
					return p, le, true
				}
			}
		}
		return le, le, true
	}
	return 0, 0, false
}

// rightPrimaryEnd walks right from rs over a primary expression (name/variable,
// member and static access, matched call/index groups) and returns its last
// token index, or -1.
func rightPrimaryEnd(s *tokens.Stream, rs int) int {
	if !isPrimaryStart(s.At(rs)) {
		return -1
	}
	end := rs
	for {
		n := nextSignificantIndex(s, end)
		if n < 0 {
			return end
		}
		nt := s.At(n)
		if nt.Kind == token.Punct {
			switch nt.Value {
			case "->", "?->", "::", `\`:
				m := nextSignificantIndex(s, n)
				if m < 0 {
					return end
				}
				end = m
				continue
			case "(", "[":
				c := s.MatchForward(n)
				if c < 0 {
					return end
				}
				end = c
				continue
			}
		}
		return end
	}
}

// isPrimaryStart reports whether t can begin a primary (variable) expression.
func isPrimaryStart(t token.Token) bool {
	switch t.Kind {
	case token.Variable, token.Ident:
		return true
	case token.Keyword:
		return isPrimaryKeyword(t.Value)
	case token.Punct:
		return t.Value == `\`
	}
	return false
}

func isPrimaryKeyword(v string) bool {
	switch strings.ToLower(v) {
	case "static", "self", "parent":
		return true
	}
	return false
}

func isLeftBoundary(s *tokens.Stream, p int) bool {
	if p < 0 {
		return true
	}
	t := s.At(p)
	if t.Kind == token.Punct {
		switch t.Value {
		case "(", "[", "{", ",", ";", "&&", "||", "?", "??", ":", "=", "!", "=>", ".":
			return true
		}
		return false
	}
	if t.Kind == token.Keyword {
		switch strings.ToLower(t.Value) {
		case "return", "and", "or", "xor", "echo", "print", "case", "if", "elseif", "while":
			return true
		}
	}
	return false
}

func isRightBoundary(s *tokens.Stream, j int) bool {
	if j < 0 {
		return true
	}
	t := s.At(j)
	if t.Kind == token.Punct {
		switch t.Value {
		case ")", "]", "}", ";", ",", ":", "&&", "||", "?", "??", ".", "=>":
			return true
		}
		return false
	}
	if t.Kind == token.Keyword {
		switch strings.ToLower(t.Value) {
		case "and", "or", "xor":
			return true
		}
	}
	return false
}
