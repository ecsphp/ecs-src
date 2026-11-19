package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Formatting/OperatorBracketSniff.php
//
// OperatorBracket wraps arithmetic and bitwise operations in parentheses:
// "$a = $b + 1;" -> "$a = ($b + 1);".
type OperatorBracket struct{}

func (OperatorBracket) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Formatting\OperatorBracketSniff`
}

func (OperatorBracket) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Formatting/OperatorBracketSniff.php"
}

// Tokens::OPERATORS.
var obOperators = phpcsSet("+", "-", "*", "/", "%", "**", "<=>", "??", "&", "|", "^", "<<", ">>")

var obAssignment = phpcsSet("=", "&=", "|=", ".=", "/=", "-=", "**=", "%=", "*=", "+=", "^=", "double_arrow",
	"<<=", ">>=", "??=")

var obComparison = phpcsSet("==", "!=", "<>", "===", "!==", "<=", ">=", "<", ">", "<=>", "??")

var obInvalidBefore = phpcsSet(",", "colon", "(", "[", "{", "open_short_array", "case", "exit", "match_arrow")

// allowed inside a bracketed operation of a function call, besides operators and names
var obAllowedInside = phpcsSet("variable", "lnumber", "whitespace", "self", "static", "parent", "->", "?->", "::",
	"[", "]", "~")

// addMissingBracketsError's allowed tokens, besides empty, operator, cast and name tokens
var obAllowedExpr = phpcsSet("variable", "lnumber", "constant_encapsed_string", "double_quoted_string", "whitespace",
	"self", "static", "->", "?->", "::", "isset", "array", "~")

type obFile struct {
	f        *slvFile
	sc       *phpcsScope
	code     []string
	notOpRef map[int]bool // "&" references and "|"/"&" type operators
}

func (OperatorBracket) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		o := newObFile(f)
		for i := range f.toks {
			if o.isOperator(i) {
				o.process(i)
			}
		}
	})
}

func newObFile(f *slvFile) *obFile {
	o := &obFile{f: f, sc: newPhpcsScope(f), code: make([]string, len(f.toks)), notOpRef: map[int]bool{}}
	for i := range f.toks {
		o.code[i] = o.classify(i)
	}
	o.markReferences()
	return o
}

// markReferences finds the "&" File::isReference accepts and the "|"/"&" the
// tokenizer turns into type operators, on a stream of the same source.
func (o *obFile) markReferences() {
	f := o.f
	var b strings.Builder
	at := map[int]int{}
	for i, t := range f.toks {
		if t.kind == token.Punct && (t.orig == "&" || t.orig == "|") {
			at[b.Len()] = i
		}
		b.WriteString(t.orig)
	}
	if len(at) == 0 {
		return
	}
	s := tokens.New(lexer.Lex(b.String()))
	ctx := newPhpcsOperatorContext(s)
	off := 0
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if j, ok := at[off]; ok && t.Kind == token.Punct && (t.Value == "&" || t.Value == "|") {
			if phpcsTypeOperator(s, i) || t.Value == "&" && ctx.isReference(s, i) {
				o.notOpRef[j] = true
			}
			if ctx.ternaryRefAmp[i] {
				o.markTernaryFn(j)
			}
		}
		off += len(t.Value)
	}
}

// markTernaryFn retokenizes the "fn&(...):" PHPCS misreads inside a ternary,
// with "&" at amp: "fn" is a plain name and the ":" an inline else.
func (o *obFile) markTernaryFn(amp int) {
	f := o.f
	if fn := f.prevEffective(amp - 1); fn >= 0 {
		o.code[fn] = "string"
	}
	open := f.nextNonWhitespace(amp+1, len(f.toks))
	if open < 0 || f.toks[open].closer < 0 {
		return
	}
	if colon := f.nextNonWhitespace(f.toks[open].closer+1, len(f.toks)); colon >= 0 && f.toks[colon].orig == ":" {
		o.code[colon] = "inline_else"
	}
}

func (o *obFile) classify(i int) string {
	f := o.f
	t := f.toks[i]
	if f.isParamName(i) {
		return "param_name"
	}
	switch t.kind {
	case token.Variable:
		if t.orig == "$" {
			return "dollar_open_curly_braces"
		}
	case token.Number:
		return "lnumber"
	case token.String:
		switch {
		case t.heredoc:
			return "heredoc"
		case t.interp:
			return "double_quoted_string"
		}
		return "constant_encapsed_string"
	case token.Ident:
		switch strings.ToLower(t.orig) {
		case "true", "false", "null", "self", "parent":
			if p := f.prevEffective(i - 1); !f.is(p, "->") && !f.is(p, "?->") && !f.is(p, "::") {
				return strings.ToLower(t.orig)
			}
		}
	case token.Keyword:
		if p := f.prevEffective(i - 1); f.is(p, "->") || f.is(p, "?->") || f.is(p, "::") {
			return "string"
		}
	case token.Punct:
		switch {
		case len(t.orig) > 2 && t.orig[0] == '(':
			if strings.EqualFold(strings.Trim(t.orig, "( \t)"), "void") {
				return "void_cast"
			}
			return "cast"
		case t.orig == "[" && f.isShortArrayOpen(i):
			return "open_short_array"
		case t.orig == "]" && t.closer >= 0 && f.isShortArrayOpen(t.closer):
			return "close_short_array"
		case t.orig == "=>":
			if c := o.sc.code[i]; c == "match_arrow" || c == "fn_arrow" {
				return c
			}
			return "double_arrow"
		case t.orig == ":":
			if o.sc.code[i] == "inline_else" {
				return "inline_else"
			}
			return "colon"
		}
	}
	return o.sc.code[i]
}

func (o *obFile) isEmpty(i int) bool { return o.f.isEmpty(i) }

func (o *obFile) isName(i int) bool { return o.code[i] == "string" }

func (o *obFile) isOperator(i int) bool {
	return o.f.toks[i].kind == token.Punct && obOperators[o.code[i]] && !o.notOpRef[i]
}

func (o *obFile) isEquality(i int) bool {
	switch o.code[i] {
	case "==", "!=", "<>", "===", "!==", "<=", ">=":
		return true
	}
	return false
}

func (o *obFile) process(ptr int) {
	f, code := o.f, o.code
	if code[ptr] == "-" {
		prev := f.prevEffective(ptr - 1)
		if prev >= 0 && code[prev] == "return" {
			return
		}
		number := f.nextNonWhitespace(ptr+1, len(f.toks))
		if number >= 0 && code[number] == "lnumber" {
			if previous := f.prevNonWhitespace(ptr - 1); previous >= 0 {
				if obAssignment[code[previous]] || o.isEquality(previous) || obComparison[code[previous]] || o.isOperator(previous) {
					return
				}
			}
		}
	}

	if prevTok := f.prevNonWhitespace(ptr - 1); prevTok >= 0 && obInvalidBefore[code[prevTok]] {
		return
	}

	if code[ptr] == "|" {
		if p := o.sc.nested[ptr]; p >= 0 && o.sc.pOwner[p] >= 0 && code[o.sc.pOwner[p]] == "catch" {
			return
		}
	}

	allowed := func(i int) bool {
		c := code[i]
		return o.isName(i) || o.isOperator(i) || obAllowedInside[c]
	}

	lastBracket := -1
	for bracket := o.sc.nested[ptr]; bracket >= 0; bracket = o.sc.nested[bracket] {
		endBracket := f.toks[bracket].closer
		prevToken := f.prevNonWhitespace(bracket - 1)
		prevCode := ""
		if prevToken >= 0 {
			prevCode = code[prevToken]
		}
		if prevCode == "isset" {
			break
		}
		if prevToken >= 0 && o.isName(prevToken) || prevCode == "switch" || prevCode == "match" {
			prev := ptr - 1
			for ; prev > bracket; prev-- {
				if allowed(prev) {
					continue
				}
				if code[prev] == ")" && f.toks[prev].closer >= 0 {
					prev = f.toks[prev].closer
				} else {
					break
				}
			}
			if prev != bracket {
				break
			}
			next := ptr + 1
			for ; next < endBracket; next++ {
				if allowed(next) {
					continue
				}
				if code[next] == "(" && f.toks[next].closer >= 0 {
					next = f.toks[next].closer
				} else {
					break
				}
			}
			if next != endBracket {
				break
			}
		}
		if phpcsScopeOpenerCodes[prevCode] && prevCode != "switch" && prevCode != "match" {
			break
		}
		if prevCode == "(" && endBracket < ptr {
			continue
		}
		lastBracket = bracket
		break
	}

	if lastBracket < 0 || f.toks[lastBracket].closer < ptr {
		o.addMissingBrackets(ptr)
		return
	}
	square := -1
	for j := ptr - 1; j >= lastBracket; j-- {
		if code[j] == "[" || code[j] == "]" {
			square = j
			break
		}
	}
	if square >= 0 && code[square] == "[" {
		for j := ptr + 1; j < len(f.toks); j++ {
			if code[j] == "[" || code[j] == "]" {
				if code[j] == "]" {
					o.addMissingBrackets(ptr)
				}
				break
			}
		}
	}
}

func (o *obFile) exprToken(i int, after bool) bool {
	c := o.code[i]
	return o.isEmpty(i) || o.isOperator(i) || c == "cast" || c == "void_cast" || o.isName(i) || obAllowedExpr[c] ||
		after && (c == "=" || c == "new")
}

func (o *obFile) addMissingBrackets(ptr int) {
	f, code := o.f, o.code
	before := ptr - 1
	for ; before > 0; before-- {
		if o.exprToken(before, false) {
			continue
		}
		c := code[before]
		if (c == ")" || c == "]" || c == "close_short_array") && f.toks[before].closer >= 0 {
			before = f.toks[before].closer
			continue
		}
		break
	}
	before = f.nextEffective(before+1, len(f.toks))
	if before < 0 || code[before] == "void_cast" || o.isVoidCastParen(before) {
		return
	}

	after := ptr + 1
	for ; after < len(f.toks); after++ {
		if o.exprToken(after, true) {
			continue
		}
		c := code[after]
		if c == "(" || c == "[" || c == "open_short_array" {
			if f.toks[after].closer < 0 {
				return
			}
			after = f.toks[after].closer
			continue
		}
		break
	}
	after = f.prevEffective(after - 1)
	if before == after || before == ptr || after == ptr {
		return
	}
	f.beginChangeset()
	f.replaceToken(before, "("+f.toks[before].content)
	f.replaceToken(after, f.toks[after].content+")")
	f.endChangeset()
}

// isVoidCastParen reports whether the "(" at i opens "(void)", a cast PHP
// 8.5 lexes as T_VOID_CAST.
func (o *obFile) isVoidCastParen(i int) bool {
	f := o.f
	if !f.is(i, "(") || f.toks[i].closer < 0 {
		return false
	}
	return strings.EqualFold(strings.Trim(f.content(i+1, f.toks[i].closer-1), " \t"), "void")
}
