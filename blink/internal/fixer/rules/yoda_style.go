package rules

import (
	"slices"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// yodaMode is how a comparison group is normalized. The zero value (yodaNonYoda)
// reproduces ECS's common-set configuration (equal/identical/less_and_greater
// all false), which is blink's default.
type yodaMode int

const (
	yodaNonYoda yodaMode = iota // false: move the variable to the left
	yodaLeave                   // null: leave the comparison as it is
	yodaYoda                    // true: move the constant to the left
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/YodaStyleFixer.php
//
// YodaStyle rewrites comparisons to the configured style per operator group.
// The zero value follows ECS's common set (equal=false, identical=false,
// less_and_greater=false): a constant on the left of a comparison and a variable
// on the right are swapped so the variable comes first ("null === $x" -> "$x ===
// null"). false is non-yoda, null leaves the group alone, true is yoda.
type YodaStyle struct {
	equal       yodaMode
	identical   yodaMode
	lessGreater yodaMode
}

func (YodaStyle) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\YodaStyleFixer`
}

func (YodaStyle) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/YodaStyleFixer.php"
}

func (f YodaStyle) WithConfig(config map[string]any) fixer.Fixer {
	f.equal = yodaModeFromConfig(config, "equal", f.equal)
	f.identical = yodaModeFromConfig(config, "identical", f.identical)
	f.lessGreater = yodaModeFromConfig(config, "less_and_greater", f.lessGreater)
	// always_move_variable is accepted but not honoured: blink's variable
	// detection does not model it.
	return f
}

// yodaModeFromConfig maps a bool|null option to a yodaMode; a missing key keeps
// the current value and an explicit null leaves that group untouched.
func yodaModeFromConfig(config map[string]any, key string, def yodaMode) yodaMode {
	v, ok := config[key]
	if !ok {
		return def
	}
	if v == nil {
		return yodaLeave
	}
	if b, ok := v.(bool); ok {
		if b {
			return yodaYoda
		}
		return yodaNonYoda
	}
	return def
}

// modeFor returns the configured mode for a comparison operator.
func (f YodaStyle) modeFor(op string) yodaMode {
	switch op {
	case "===", "!==":
		return f.identical
	case "<", ">", "<=", ">=":
		return f.lessGreater
	default: // "==", "!="
		return f.equal
	}
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

func (f YodaStyle) Fix(s *tokens.Stream) bool {
	type swap struct{ ls, le, rs, re int }
	var swaps []swap
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Punct || !yodaOp(t.Value) {
			continue
		}
		switch f.modeFor(t.Value) {
		case yodaLeave:
			continue
		case yodaNonYoda:
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
		case yodaYoda:
			// right must be a bounded constant
			rs, re, ok := yodaRightLiteralOperand(s, i)
			if !ok || !isRightBoundary(s, nextSignificantIndex(s, re)) {
				continue
			}
			// left must be a variable expression (not itself a constant), bounded
			le := prevSignificantIndex(s, i)
			if le < 0 {
				continue
			}
			ls := yodaLeftPrimaryStart(s, le)
			if ls < 0 {
				continue
			}
			if ls == le && isYodaLiteral(s.At(ls)) {
				continue // both sides constant
			}
			if !isLeftBoundary(s, prevSignificantIndex(s, ls)) {
				continue
			}
			swaps = append(swaps, swap{ls, le, rs, re})
		}
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
	// class constant "Name::CONST" or "Name::class", possibly namespaced
	if t.Kind == token.Ident {
		if p := prevSignificantIndex(s, le); p >= 0 && s.At(p).Kind == token.Punct && s.At(p).Value == "::" {
			n := prevSignificantIndex(s, p)
			if n >= 0 && s.At(n).Kind == token.Ident {
				return qualifiedNameStart(s, n), le, true
			}
			return 0, 0, false
		}
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

// qualifiedNameStart walks left from the last identifier of a qualified name over
// "\Ident" namespace segments and an optional leading "\", returning the name's
// first token index.
func qualifiedNameStart(s *tokens.Stream, end int) int {
	start := end
	for {
		p := prevSignificantIndex(s, start)
		if p < 0 || s.At(p).Kind != token.Punct || s.At(p).Value != `\` {
			return start
		}
		pp := prevSignificantIndex(s, p)
		if pp >= 0 && s.At(pp).Kind == token.Ident {
			start = pp
			continue
		}
		return p // leading "\Foo"
	}
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

// yodaRightLiteralOperand returns the span of a constant right operand starting
// just after the comparison at op: a plain literal, signed number, empty array,
// bare constant, or "Name::class". It mirrors leftLiteralOperand.
func yodaRightLiteralOperand(s *tokens.Stream, op int) (int, int, bool) {
	rs := nextSignificantIndex(s, op)
	if rs < 0 {
		return 0, 0, false
	}
	t := s.At(rs)
	// signed number "-1"
	if t.Kind == token.Punct && (t.Value == "-" || t.Value == "+") {
		n := nextSignificantIndex(s, rs)
		if n >= 0 && s.At(n).Kind == token.Number {
			return rs, n, true
		}
		return 0, 0, false
	}
	// "[" of an empty array "[]"
	if t.Kind == token.Punct && t.Value == "[" {
		c := s.MatchForward(rs)
		if c >= 0 && nextSignificantIndex(s, rs) == c {
			return rs, c, true
		}
		return 0, 0, false
	}
	if isYodaLiteral(t) {
		// "Name::class"
		if t.Kind == token.Ident {
			n := nextSignificantIndex(s, rs)
			if n >= 0 && s.At(n).Value == "::" {
				c := nextSignificantIndex(s, n)
				if c >= 0 && s.At(c).Kind == token.Ident && strings.EqualFold(s.At(c).Value, "class") {
					return rs, c, true
				}
			}
		}
		return rs, rs, true
	}
	return 0, 0, false
}

// yodaLeftPrimaryStart walks left from a primary's last token (end) over member,
// static and namespace access and matched call/index groups, returning the first
// token index of that primary, or -1 when end does not end a primary.
func yodaLeftPrimaryStart(s *tokens.Stream, end int) int {
	if !yodaIsPrimaryEndToken(s.At(end)) {
		return -1
	}
	cur := end
	for {
		t := s.At(cur)
		if t.Kind == token.Punct && (t.Value == ")" || t.Value == "]") {
			o := s.MatchBackward(cur)
			if o < 0 {
				return -1
			}
			p := prevSignificantIndex(s, o)
			if p < 0 {
				return -1 // a bare "(...)" group, not a variable primary
			}
			cur = p
			continue
		}
		p := prevSignificantIndex(s, cur)
		if p >= 0 && s.At(p).Kind == token.Punct {
			switch s.At(p).Value {
			case "->", "?->", "::", `\`:
				pp := prevSignificantIndex(s, p)
				if pp < 0 {
					return cur
				}
				cur = pp
				continue
			}
		}
		return cur
	}
}

// yodaIsPrimaryEndToken reports whether t can end a primary (variable) expression.
func yodaIsPrimaryEndToken(t token.Token) bool {
	switch t.Kind {
	case token.Variable, token.Ident:
		return true
	case token.Keyword:
		return isPrimaryKeyword(t.Value)
	case token.Punct:
		return t.Value == ")" || t.Value == "]"
	}
	return false
}
