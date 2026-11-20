package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// slevomat's YodaHelper, shared by DisallowYodaComparison and RequireYodaComparison.

const (
	slvYodaVariable     = 999
	slvYodaConstant     = 1
	slvYodaFunctionCall = 998
)

// slvYodaCasts maps a cast to whether DynamismForTokens skips it (all but
// binary and void casts).
var slvYodaCasts = map[string]bool{
	"int": true, "integer": true, "double": true, "float": true, "real": true, "string": true,
	"array": true, "object": true, "bool": true, "boolean": true, "unset": true,
	"binary": false, "void": false,
}

var slvYodaAssignments = phpcsSet("=", "&=", "|=", ".=", "/=", "-=", "**=", "%=", "*=", "+=", "^=", "=>",
	"<<=", ">>=", "??=")

// slvYodaCast returns the cast name of the token at i ("(int)" -> "int").
func (f *slvFile) slvYodaCast(i int) (string, bool) {
	t := f.toks[i]
	if t.kind != token.Punct || len(t.orig) < 3 || t.orig[0] != '(' {
		return "", false
	}
	name := strings.ToLower(strings.Trim(t.orig, "( \t)"))
	_, ok := slvYodaCasts[name]
	return name, ok
}

// slvYodaCode is the PHP_CodeSniffer token code YodaHelper looks at.
func (f *slvFile) slvYodaCode(sc *phpcsScope, i int) string {
	t := f.toks[i]
	switch t.kind {
	case token.Number:
		return "number"
	case token.String:
		if t.interp || t.heredoc {
			return "other"
		}
		return "constant_encapsed_string"
	case token.Ident:
		// PHP_CodeSniffer tokenizes a fully qualified "\true" as T_TRUE
		switch v := strings.ToLower(strings.TrimPrefix(t.orig, `\`)); v {
		case "true", "false", "null":
			if p := f.prevEffective(i - 1); !f.isMemberOperator(p) && !f.isKeyword(p, "function") && !f.isKeyword(p, "const") {
				return v
			}
		}
		if f.lineCallIsName(i) {
			if f.lineCallIsPlainString(i) {
				return "string"
			}
			return "name"
		}
		return strings.ToLower(strings.TrimPrefix(t.orig, `\`))
	case token.Keyword:
		if p := f.prevEffective(i - 1); f.isMemberOperator(p) {
			return "string"
		}
		return strings.ToLower(t.orig)
	case token.OpenTag:
		if strings.HasPrefix(t.orig, "<?=") {
			return "open_tag_with_echo"
		}
		return "open_tag"
	case token.Comment, token.DocComment:
		return "comment"
	case token.Whitespace:
		return "whitespace"
	case token.Variable:
		if t.orig == "$" {
			return "dollar"
		}
		return "variable"
	case token.Punct:
		if _, ok := f.slvYodaCast(i); ok {
			return "cast"
		}
		switch {
		case f.isShortArrayOpen(i):
			return "open_short_array"
		case f.is(i, "]") && t.closer >= 0 && f.isShortArrayOpen(t.closer):
			return "close_short_array"
		}
		return sc.code[i]
	}
	return "other"
}

func (f *slvFile) slvYodaStop(sc *phpcsScope, i int) bool {
	switch c := f.slvYodaCode(sc, i); c {
	case "&&", "||", ";", "open_tag", "?", "inline_else", "and", "or", "xor", "??", "case", ":", "return", ",",
		"match_arrow", "fn_arrow", "comment":
		return true
	default:
		return slvYodaAssignments[c]
	}
}

// slvYodaLeft is YodaHelper::getLeftSideTokens, trimmed of whitespace.
func (f *slvFile) slvYodaLeft(sc *phpcsScope, cmp int) []int {
	parens, arrays := 0, 0
	var side []int
	for i := cmp - 1; i >= 0; i-- {
		if parens == 0 && arrays == 0 && f.slvYodaStop(sc, i) {
			break
		}
		switch f.slvYodaCode(sc, i) {
		case "close_short_array":
			arrays++
		case "open_short_array":
			if arrays == 0 {
				return f.slvYodaTrim(side)
			}
			arrays--
		}
		if f.is(i, ")") {
			parens++
		} else if f.is(i, "(") {
			if parens == 0 {
				break
			}
			parens--
		}
		side = append([]int{i}, side...)
	}
	return f.slvYodaTrim(side)
}

// slvYodaRight is YodaHelper::getRightSideTokens, trimmed of whitespace.
func (f *slvFile) slvYodaRight(sc *phpcsScope, cmp int) []int {
	parens, arrays := 0, 0
	var side []int
	for i := cmp + 1; i < len(f.toks); i++ {
		if parens == 0 && arrays == 0 && f.slvYodaStop(sc, i) {
			break
		}
		switch f.slvYodaCode(sc, i) {
		case "open_short_array":
			arrays++
		case "close_short_array":
			if arrays == 0 {
				return f.slvYodaTrim(side)
			}
			arrays--
		}
		if f.is(i, "(") {
			parens++
		} else if f.is(i, ")") {
			if parens == 0 {
				break
			}
			parens--
		}
		side = append(side, i)
	}
	return f.slvYodaTrim(side)
}

func (f *slvFile) slvYodaTrim(side []int) []int {
	for len(side) > 0 && f.toks[side[0]].kind == token.Whitespace {
		side = side[1:]
	}
	for len(side) > 0 && f.toks[side[len(side)-1]].kind == token.Whitespace {
		side = side[:len(side)-1]
	}
	return side
}

// slvYodaDynamism is YodaHelper::getDynamismForTokens; ok is false for null.
func (f *slvFile) slvYodaDynamism(sc *phpcsScope, side []int) (int, bool) {
	var toks []int
	for _, i := range side {
		switch f.slvYodaCode(sc, i) {
		case "whitespace", "comment", "+", "-":
			continue
		case "cast":
			if name, _ := f.slvYodaCast(i); slvYodaCasts[name] {
				continue
			}
		}
		toks = append(toks, i)
	}
	n := len(toks)
	code := func(k int) string { return f.slvYodaCode(sc, toks[k]) }
	if n > 0 {
		if code(0) == "variable" {
			return slvYodaVariable, true
		}
		if last := toks[n-1]; f.is(last, ")") {
			if open := f.toks[last].closer; open >= 0 && sc.pOwner[open] >= 0 && sc.code[sc.pOwner[open]] == "array" {
				return 0, true
			}
			return slvYodaFunctionCall, true
		}
		if n == 1 && code(0) == "string" {
			return slvYodaConstant, true
		}
	}
	if n > 2 && code(n-2) == "::" {
		switch code(n - 1) {
		case "variable":
			return slvYodaVariable, true
		case "string":
			return slvYodaConstant, true
		}
	}
	if n > 0 {
		switch code(0) {
		case "true", "false", "null", "number", "open_short_array", "array", "constant_encapsed_string":
			return 0, true
		case "variable":
			return slvYodaVariable, true
		case "string":
			return slvYodaFunctionCall, true
		case "cast":
			return 3, true
		}
	}
	return 0, false
}

// slvYodaFix is YodaHelper::fix: it swaps the two sides.
func (f *slvFile) slvYodaFix(left, right []int) {
	sideContent := func(side []int) string {
		var b strings.Builder
		for _, i := range side {
			b.WriteString(f.toks[i].content)
		}
		return b.String()
	}
	f.beginChangeset()
	f.change(left[0], left[len(left)-1], sideContent(right))
	f.change(right[0], right[len(right)-1], sideContent(left))
	f.endChangeset()
}

func (f *slvFile) slvYodaComparison(i int) bool {
	return f.is(i, "===") || f.is(i, "!==") || f.is(i, "==") || f.is(i, "!=") || f.is(i, "<>")
}
