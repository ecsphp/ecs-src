package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsOperatorPuncts are the operator tokens Squiz OperatorSpacing registers:
// PHP_CodeSniffer's COMPARISON_TOKENS, OPERATORS and ASSIGNMENT_TOKENS.
var phpcsOperatorPuncts = map[string]bool{
	"==": true, "!=": true, "<>": true, "===": true, "!==": true,
	"<": true, ">": true, "<=": true, ">=": true, "<=>": true, "??": true,
	"+": true, "-": true, "*": true, "/": true, "%": true, "**": true,
	"&": true, "|": true, "^": true, "<<": true, ">>": true,
	"=": true, "+=": true, "-=": true, "*=": true, "/=": true, ".=": true,
	"%=": true, "**=": true, "&=": true, "|=": true, "^=": true,
	"<<=": true, ">>=": true, "??=": true, "=>": true,
}

// phpcsOperatorContext is the per-file token classification PHP_CodeSniffer's
// tokenizer would produce for OperatorSpacing.
type phpcsOperatorContext struct {
	inlineElse  map[int]bool // ":" of a ternary (T_INLINE_ELSE)
	notArrow    map[int]bool // "=>" of a match arm or arrow function
	paramParens map[int]bool // "(" opening function/closure/fn parameters
	useParens   map[int]bool // "(" opening a closure use list
	declare     map[int]int  // declare keyword -> its ")" (or -1 when unclosed)
}

func newPhpcsOperatorContext(s *tokens.Stream) phpcsOperatorContext {
	o := phpcsOperatorContext{
		inlineElse:  map[int]bool{},
		notArrow:    map[int]bool{},
		paramParens: map[int]bool{},
		useParens:   map[int]bool{},
		declare:     map[int]int{},
	}
	type pending struct{ depth int }
	var ternaries []pending
	depth := 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		switch t.Kind {
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "declare":
				if open := phpcsNextNonEmpty(s, i+1); open < s.Len() && s.At(open).Value == "(" {
					o.declare[i] = s.MatchForward(open)
				} else {
					o.declare[i] = -1
				}
			case "function", "fn":
				o.markFunction(s, i)
			case "match":
				o.markMatch(s, i)
			}
		case token.Punct:
			switch t.Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
				for len(ternaries) > 0 && ternaries[len(ternaries)-1].depth > depth {
					ternaries = ternaries[:len(ternaries)-1]
				}
			case "?":
				if !phpcsIsNullable(s, i) {
					ternaries = append(ternaries, pending{depth})
				}
			case ":":
				if n := len(ternaries); n > 0 && ternaries[n-1].depth == depth {
					o.inlineElse[i] = true
					ternaries = ternaries[:n-1]
				}
			case ";":
				for len(ternaries) > 0 && ternaries[len(ternaries)-1].depth >= depth {
					ternaries = ternaries[:len(ternaries)-1]
				}
			}
		}
	}
	return o
}

// markFunction records the parameter (and closure use) parentheses of the
// function, closure or arrow function at i, and the "=>" of an arrow function.
func (o phpcsOperatorContext) markFunction(s *tokens.Stream, i int) {
	isFn := strings.EqualFold(s.At(i).Value, "fn")
	j := phpcsNextNonEmpty(s, i+1)
	if j < s.Len() && s.At(j).Value == "&" {
		j = phpcsNextNonEmpty(s, j+1)
	}
	if j < s.Len() && s.At(j).Kind != token.Punct {
		j = phpcsNextNonEmpty(s, j+1)
	}
	if j >= s.Len() || s.At(j).Value != "(" {
		return
	}
	closeIdx := s.MatchForward(j)
	if closeIdx < 0 {
		return
	}
	o.paramParens[j] = true
	next := phpcsNextNonEmpty(s, closeIdx+1)
	if next < s.Len() && s.At(next).Kind == token.Keyword && strings.EqualFold(s.At(next).Value, "use") {
		if open := phpcsNextNonEmpty(s, next+1); open < s.Len() && s.At(open).Value == "(" {
			o.useParens[open] = true
		}
	}
	if !isFn {
		return
	}
	depth := 0
	for k := closeIdx + 1; k < s.Len(); k++ {
		switch v := s.At(k).Value; {
		case s.At(k).Kind != token.Punct:
		case v == "(" || v == "[" || v == "{":
			depth++
		case v == ")" || v == "]" || v == "}":
			depth--
			if depth < 0 {
				return
			}
		case v == "=>" && depth == 0:
			o.notArrow[k] = true
			return
		case v == ";" && depth == 0:
			return
		}
	}
}

// markMatch records the arm arrows of the match expression at i.
func (o phpcsOperatorContext) markMatch(s *tokens.Stream, i int) {
	open := phpcsNextNonEmpty(s, i+1)
	if open >= s.Len() || s.At(open).Value != "(" {
		return
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return
	}
	body := phpcsNextNonEmpty(s, closeIdx+1)
	if body >= s.Len() || s.At(body).Value != "{" {
		return
	}
	end := s.MatchForward(body)
	if end < 0 {
		return
	}
	depth := 0
	for k := body + 1; k < end; k++ {
		if s.At(k).Kind != token.Punct {
			continue
		}
		switch v := s.At(k).Value; v {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		case "=>":
			if depth == 0 {
				o.notArrow[k] = true
			}
		}
	}
}

// isOperatorToken reports whether the token at i is one Squiz OperatorSpacing
// registers (PSR12 also adds boolean operators and "." via withPSR12).
func (o phpcsOperatorContext) isOperatorToken(s *tokens.Stream, i int, withPSR12 bool) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "instanceof":
			return true
		case "and", "or", "xor":
			return withPSR12
		}
	case token.Punct:
		switch t.Value {
		case "?":
			return !phpcsIsNullable(s, i)
		case ":":
			return o.inlineElse[i]
		case "=>":
			return !o.notArrow[i]
		case "&&", "||", ".":
			return withPSR12
		}
		return phpcsOperatorPuncts[t.Value]
	}
	return false
}

// isOperator ports Squiz OperatorSpacingSniff::isOperator: parameter defaults,
// "=&", references, type operators and unary "+"/"-" are not operators.
func (o phpcsOperatorContext) isOperator(s *tokens.Stream, i int) bool {
	switch s.At(i).Value {
	case "=":
		if o.paramParens[phpcsInnermostParen(s, i)] {
			return false
		}
		if i+1 < s.Len() && s.At(i+1).Value == "&" {
			return false
		}
	case "&":
		if o.isReference(s, i) || phpcsTypeOperator(s, i) {
			return false
		}
	case "|":
		if phpcsTypeOperator(s, i) {
			return false
		}
	case "-", "+":
		if p := phpcsPrevNonEmpty(s, i-1); p >= 0 && phpcsNonOperand(s, p, o) {
			return false
		}
	}
	return true
}

// isReference ports File::isReference for the "&" at i.
func (o phpcsOperatorContext) isReference(s *tokens.Stream, i int) bool {
	before := phpcsPrevNonEmpty(s, i-1)
	if before < 0 {
		return false
	}
	b := s.At(before)
	lower := strings.ToLower(b.Value)
	if b.Kind == token.Keyword && (lower == "function" || lower == "fn" || lower == "as") {
		return true
	}
	if b.Kind == token.Punct && (phpcsAssignment[b.Value] || b.Value == "=>") {
		return true
	}
	after := phpcsNextNonEmpty(s, i+1)
	if after < s.Len() && s.At(after).Kind == token.Keyword && strings.EqualFold(s.At(after).Value, "new") {
		return true
	}
	// a parameter passed by reference: "&" before the variable, ahead of any default
	if open := phpcsInnermostParen(s, i); open >= 0 && (o.paramParens[open] || o.useParens[open]) && !phpcsInParamDefault(s, open, i) {
		return true
	}
	if b.Kind == token.Punct && (b.Value == "(" || b.Value == "," || (b.Value == "[" && phpcsShortArrayOpen(s, before))) {
		k := after
		for k < s.Len() {
			t := s.At(k)
			if isPhpcsEmpty(t) || t.Kind == token.Ident || t.Value == `\` || t.Value == "::" ||
				(t.Kind == token.Keyword && (strings.EqualFold(t.Value, "static") || strings.EqualFold(t.Value, "namespace"))) {
				k++
				continue
			}
			break
		}
		return k < s.Len() && s.At(k).Kind == token.Variable
	}
	return false
}

var phpcsAssignment = map[string]bool{
	"=": true, "+=": true, "-=": true, "*=": true, "/=": true, ".=": true,
	"%=": true, "**=": true, "&=": true, "|=": true, "^=": true,
	"<<=": true, ">>=": true, "??=": true,
}

// phpcsNonOperand reports whether the token at p makes a following "+"/"-" a
// sign rather than an operator (Squiz OperatorSpacing's nonOperandTokens).
func phpcsNonOperand(s *tokens.Stream, p int, o phpcsOperatorContext) bool {
	t := s.At(p)
	switch t.Kind {
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "and", "or", "xor", "return", "echo", "print", "yield", "case":
			return true
		}
	case token.Ident:
		lower := strings.ToLower(t.Value)
		return lower == "exit" || lower == "die"
	case token.Punct:
		switch t.Value {
		case "&&", "||", ":", ",", "?", "{", "(", "[", ".", "=>":
			return true
		case ")":
			if open := s.MatchBackward(p); open >= 0 {
				if _, _, ok := phpCastAt(s, open); ok {
					return true
				}
			}
			return false
		}
		return phpcsOperatorPuncts[t.Value]
	}
	return false
}

// phpcsInnermostParen is the "(" of the innermost parentheses around i, or -1.
func phpcsInnermostParen(s *tokens.Stream, i int) int {
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
				return j
			}
			depth--
		}
	}
	return -1
}

// phpcsInParamDefault reports whether i sits after the "=" of its parameter
// within the parameter list opened at open.
func phpcsInParamDefault(s *tokens.Stream, open, i int) bool {
	depth := 0
	for j := i - 1; j > open; j-- {
		t := s.At(j)
		if t.Kind != token.Punct {
			continue
		}
		switch t.Value {
		case ")", "]", "}":
			depth++
		case "(", "[", "{":
			depth--
		case ",":
			if depth == 0 {
				return false
			}
		case "=":
			if depth == 0 {
				return true
			}
		}
	}
	return false
}

// phpcsTypeName reports whether t can be part of a type name for the
// tokenizer's union/intersection detection.
func phpcsTypeName(s *tokens.Stream, i int, allowStatic bool) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Ident:
		return true
	case token.Punct:
		return t.Value == `\`
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "callable", "namespace":
			return true
		case "static":
			return allowStatic
		case "array":
			n := phpcsNextNonEmpty(s, i+1)
			return n >= s.Len() || s.At(n).Value != "("
		}
	}
	return false
}

// phpcsTypeOperator ports the tokenizer's retokenizing of "|" and "&" into
// T_TYPE_UNION / T_TYPE_INTERSECTION in parameter, property, constant and
// return types.
func phpcsTypeOperator(s *tokens.Stream, i int) bool {
	suspected := ""
	after := 0
	// parentheses of a DNF group ahead count towards the pairing check: PHPCS
	// converts the whole type once it reaches the operator after the group
	parens := 0
	for x := i + 1; x < s.Len() && suspected == ""; x++ {
		t := s.At(x)
		if isPhpcsEmpty(t) {
			continue
		}
		if phpcsTypeName(s, x, true) {
			after++
			continue
		}
		if after > 0 && (t.Value == "&" || t.Value == "..." || t.Value == "|") {
			continue
		}
		if t.Kind == token.Punct && (t.Value == "(" || t.Value == ")") {
			parens++
			continue
		}
		switch {
		case t.Kind == token.Variable:
			suspected = "param"
		case t.Value == "=>" || t.Value == ";" || t.Value == "{":
			suspected = "return"
		case t.Value == "=":
			suspected = "constant"
		default:
			return false
		}
	}
	if after == 0 || suspected == "" {
		return false
	}
	confirmed := false
loop:
	for x := i - 1; x >= 0; x-- {
		t := s.At(x)
		lower := strings.ToLower(t.Value)
		if isPhpcsEmpty(t) {
			continue
		}
		if suspected == "param" && t.Value == "(" {
			if p := phpcsPrevNonEmpty(s, x-1); p >= 0 && phpcsOwnsParams(s, p) {
				confirmed = true
				break
			}
		}
		if phpcsTypeName(s, x, suspected != "param") {
			continue
		}
		switch {
		case t.Value == "?" || t.Value == "|" || t.Value == "&":
			continue
		case t.Value == "(" || t.Value == ")":
			parens++
			continue
		case suspected == "return" && t.Value == ":":
			if p := phpcsPrevNonEmpty(s, x-1); p >= 0 && s.At(p).Value == ")" {
				if open := s.MatchBackward(p); open >= 0 {
					if q := phpcsPrevNonEmpty(s, open-1); q >= 0 && (phpcsOwnsParams(s, q) || (s.At(q).Kind == token.Keyword && strings.EqualFold(s.At(q).Value, "use"))) {
						confirmed = true
					}
				}
			}
			break loop
		case suspected == "constant" && t.Kind == token.Keyword && lower == "const":
			confirmed = true
		case suspected == "param" && t.Kind == token.Keyword && phpcsNullableModifier(lower) && lower != "function" && lower != "fn":
			confirmed = true
		}
		break
	}
	if !confirmed && suspected == "param" {
		if open := phpcsInnermostParen(s, i); open >= 0 {
			if p := phpcsPrevNonEmpty(s, open-1); p >= 0 && phpcsOwnsParams(s, p) {
				confirmed = true
			}
		}
	}
	return confirmed && parens%2 == 0
}

// phpcsOwnsParams reports whether the token at p (just before a "(") makes
// that parenthesis a parameter list: a function name, "function", "fn" or "fn&".
func phpcsOwnsParams(s *tokens.Stream, p int) bool {
	t := s.At(p)
	if t.Value == "&" {
		if p = phpcsPrevNonEmpty(s, p-1); p < 0 {
			return false
		}
		t = s.At(p)
	}
	if t.Kind == token.Keyword {
		lower := strings.ToLower(t.Value)
		return lower == "function" || lower == "fn"
	}
	if t.Kind == token.Ident {
		q := phpcsPrevNonEmpty(s, p-1)
		if q >= 0 && s.At(q).Value == "&" {
			q = phpcsPrevNonEmpty(s, q-1)
		}
		return q >= 0 && s.At(q).Kind == token.Keyword && strings.EqualFold(s.At(q).Value, "function")
	}
	return false
}
