package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Helpers/ReferencedNameHelper.php

type slvRefType uint8

const (
	slvRefClass slvRefType = iota
	slvRefFunction
	slvRefConstant
)

// slvReferencedName is slevomat's ReferencedName.
type slvReferencedName struct {
	name       string
	start, end int
	typ        slvRefType
}

// slvNames carries the PHP_CodeSniffer token types ReferencedNameHelper reads
// that blink's tokens do not tell apart: type operators, nullable "?" and the
// ternary ":".
type slvNames struct {
	f          *slvFile
	typeOps    map[int]bool // T_TYPE_UNION, T_TYPE_INTERSECTION and type parentheses
	nullable   map[int]bool // T_NULLABLE
	inlineElse map[int]bool // T_INLINE_ELSE
}

func newSlvNames(f *slvFile) *slvNames {
	n := &slvNames{f: f, typeOps: map[int]bool{}, nullable: map[int]bool{}, inlineElse: map[int]bool{}}
	for i := range f.toks {
		if f.is(i, "?") && n.isNullable(i) {
			n.nullable[i] = true
		}
	}
	n.markInlineElse()
	n.markTypeOperators()
	return n
}

// referencedNames is ReferencedNameHelper::getAllReferencedNames.
func (n *slvNames) referencedNames(openTag int) []slvReferencedName {
	f := n.f
	var out []slvReferencedName
	begin := openTag + 1
	for {
		p := -1
		for j := max(begin, 0); j < len(f.toks); j++ {
			if n.isNameToken(j) || n.isParsedString(j) {
				p = j
				break
			}
		}
		if p < 0 {
			return out
		}
		if n.isParsedString(p) {
			var b strings.Builder
			cur := p
			for cur < len(f.toks) && n.isParsedString(cur) {
				b.WriteString(f.toks[cur].content)
				cur++
			}
			for _, name := range slvNamesFromString(b.String()) {
				out = append(out, slvReferencedName{name, p, p, slvRefClass})
			}
			begin = cur
			continue
		}
		if a := n.attributeBefore(p, begin); a >= 0 && f.toks[a].closer > p {
			begin = f.toks[a].closer + 1
			continue
		}
		if !n.isReferencedName(p) {
			j := p + 1
			for j < len(f.toks) && (f.isEmpty(j) || n.isNameToken(j) || n.isParsedString(j)) {
				j++
			}
			begin = j
			continue
		}
		out = append(out, slvReferencedName{f.toks[p].content, p, p, n.referenceType(p, p)})
		begin = p + 1
	}
}

func (n *slvNames) attributeBefore(i, stop int) int {
	for j := i - 1; j >= stop && j >= 0; j-- {
		if n.f.toks[j].kind == token.AttributeStart {
			return j
		}
	}
	return -1
}

// isNameToken tells a TokenHelper::NAME_TOKEN_CODES token: PHP_CodeSniffer
// makes an "array" not opening an array and the "function" or "const" of an
// import a T_STRING, and a named argument or a goto label no name.
func (n *slvNames) isNameToken(i int) bool {
	f := n.f
	if i < 0 || i >= len(f.toks) {
		return false
	}
	switch f.toks[i].kind {
	case token.Keyword:
		switch strings.ToLower(f.toks[i].orig) {
		case "array":
			next := f.nextEffective(i+1, len(f.toks))
			return next >= 0 && !f.is(next, "(")
		case "function", "const":
			return f.isKeyword(f.prevEffective(i-1), "use")
		}
	case token.Ident:
		return f.isName(i) && !n.isParamName(i) && !n.isGotoLabel(i)
	}
	return false
}

func (n *slvNames) isPlainString(i int) bool {
	return i >= 0 && i < len(n.f.toks) && n.f.toks[i].kind == token.Ident && !strings.Contains(n.f.toks[i].orig, `\`)
}

// isParamName tells T_PARAM_NAME: a named argument's label.
func (n *slvNames) isParamName(i int) bool {
	f := n.f
	if !n.isPlainString(i) {
		return false
	}
	next := f.nextEffective(i+1, len(f.toks))
	if !f.is(next, ":") {
		return false
	}
	prev := f.prevEffective(i - 1)
	return f.is(prev, "(") || f.is(prev, ",")
}

// isGotoLabel tells T_GOTO_LABEL.
func (n *slvNames) isGotoLabel(i int) bool {
	f := n.f
	if !n.isPlainString(i) || n.isParamName(i) || f.is(i-1, "::") {
		return false
	}
	if !f.is(f.nextEffective(i+1, len(f.toks)), ":") {
		return false
	}
	for x := i - 1; x > 0; x-- {
		t := f.toks[x]
		switch {
		case f.is(x, "::"), f.isKeyword(x, "case"), f.is(x, "?") && !n.nullable[x], f.isKeyword(x, "enum"):
			return false
		case f.is(x, ";"), f.is(x, "{"), t.kind == token.OpenTag:
			return true
		}
	}
	return true
}

func (n *slvNames) isGotoColon(i int) bool {
	return n.isGotoLabel(n.f.prevEffective(i - 1))
}

// isColon tells T_COLON from T_INLINE_ELSE and T_GOTO_COLON.
func (n *slvNames) isColon(i int) bool {
	return n.f.is(i, ":") && !n.inlineElse[i] && !n.isGotoColon(i)
}

var slvNowdocStart = regexp.MustCompile(`^[bB]?<<<[ \t]*'`)

// isParsedString tells a T_DOUBLE_QUOTED_STRING or T_HEREDOC token.
func (n *slvNames) isParsedString(i int) bool {
	f := n.f
	t := f.toks[i]
	if t.kind != token.String {
		return false
	}
	if t.heredoc {
		if slvIsHeredoc(t.orig) || !strings.HasSuffix(t.orig, "\n") {
			return false
		}
		s := i
		for s > 0 && !slvIsHeredoc(f.toks[s].orig) {
			s--
		}
		return !slvNowdocStart.MatchString(f.toks[s].orig)
	}
	s := i
	for s > 0 && f.toks[s-1].kind == token.String && !f.toks[s-1].heredoc && strings.HasSuffix(f.toks[s-1].orig, "\n") {
		s--
	}
	v := strings.TrimLeft(f.toks[s].orig, "bB")
	return strings.HasPrefix(v, `"`)
}

// slvNamesFromString is ReferencedNameHelper::getReferencedNamesFromString.
func slvNamesFromString(content string) []string {
	toks := lexer.Lex("<?php " + content)
	isPart := func(j int) bool {
		return j >= 0 && j < len(toks) && (toks[j].Kind == token.Ident || toks[j].Kind == token.Punct && toks[j].Value == `\`)
	}
	var names []string
	for p, t := range toks {
		switch {
		case t.Kind == token.Punct && t.Value == "::":
			name := ""
			for j := p - 1; isPart(j); j-- {
				name = toks[j].Value + name
			}
			if name != "" {
				names = append(names, name)
			}
		case t.Kind == token.Keyword && strings.EqualFold(t.Value, "new"):
			j := p + 1
			for j < len(toks) && toks[j].Kind == token.Whitespace {
				j++
			}
			var name strings.Builder
			for ; isPart(j); j++ {
				name.WriteString(toks[j].Value)
			}
			if name.Len() > 0 {
				names = append(names, name.String())
			}
		}
	}
	return names
}

// isConstKeyword tells T_CONST from the T_STRING "const" of an import.
func (n *slvNames) isConstKeyword(i int) bool {
	return n.f.isKeyword(i, "const") && !n.f.isKeyword(n.f.prevEffective(i-1), "use")
}

// isFunctionKeyword tells T_FUNCTION: no closure, no "use function".
func (n *slvNames) isFunctionKeyword(i int) bool {
	f := n.f
	return f.isKeyword(i, "function") && !f.isKeyword(f.prevEffective(i-1), "use") && !f.isClosure(i)
}

func (n *slvNames) isEnumCase(i int) bool {
	f := n.f
	if !f.isKeyword(i, "case") {
		return false
	}
	o := f.enclosingScope(i)
	return o >= 0 && f.isKeyword(f.braceOwner(o), "enum")
}

// referenceType is ReferencedNameHelper::getReferenceType.
func (n *slvNames) referenceType(start, end int) slvRefType {
	f := n.f
	next := f.nextEffective(end+1, len(f.toks))
	prev := f.prevEffective(start - 1)
	isUnion := func(i int) bool { return f.is(i, "|") && n.typeOps[i] }
	isIntersection := func(i int) bool { return f.is(i, "&") && n.typeOps[i] }
	if f.is(next, "(") && !n.typeOps[next] {
		if f.isKeyword(prev, "new") {
			return slvRefClass
		}
		return slvRefFunction
	}
	if n.isConstKeyword(prev) {
		return slvRefClass
	}
	if isUnion(prev) || isUnion(next) || isIntersection(prev) || isIntersection(next) {
		return slvRefClass
	}
	if f.is(next, "&") {
		after := f.nextEffective(next+1, len(f.toks))
		if f.isVariable(after) || f.is(after, "...") {
			return slvRefClass
		}
		return slvRefConstant
	}
	if f.isVariable(next) || f.is(next, "...") {
		return slvRefClass
	}
	if n.isColon(prev) {
		if n.isParamName(f.prevEffective(prev-1)) && !f.is(next, "::") {
			return slvRefConstant
		}
		return slvRefClass
	}
	for _, k := range []string{"extends", "implements", "instanceof", "use", "new"} {
		if f.isKeyword(prev, k) {
			return slvRefClass
		}
	}
	if f.is(prev, "?") && n.nullable[prev] || f.is(next, "::") {
		return slvRefClass
	}
	if f.is(prev, ",") {
		q := prev - 1
		for q >= 0 && (f.is(q, ",") || n.isNameToken(q) || f.isEmpty(q)) {
			q--
		}
		if f.isKeyword(q, "implements") || f.isKeyword(q, "extends") || f.isKeyword(q, "use") {
			return slvRefClass
		}
		return slvRefConstant
	}
	bitwiseOrOpen := func(i int) bool { return (f.is(i, "|") || f.is(i, "(")) && !n.typeOps[i] }
	if bitwiseOrOpen(prev) {
		q := prev - 1
		for q >= 0 && (bitwiseOrOpen(q) || n.isNameToken(q) || f.isEmpty(q)) {
			q--
		}
		if f.isKeyword(q, "catch") {
			return slvRefClass
		}
	}
	return slvRefConstant
}

var slvrnSimpleTypeHints = map[string]bool{
	"int": true, "integer": true, "false": true, "float": true, "string": true, "bool": true,
	"boolean": true, "callable": true, "self": true, "array": true, "iterable": true, "void": true,
	"never": true, "object": true,
}

// isReferencedName is ReferencedNameHelper::isReferencedName.
func (n *slvNames) isReferencedName(start int) bool {
	f := n.f
	if n.isPropertyHookName(start) {
		return false
	}
	next := f.nextEffective(start+1, len(f.toks))
	prev := f.prevEffective(start - 1)
	if f.is(next, "::") {
		return !f.is(prev, "->") && !f.is(prev, "?->")
	}
	if n.innermostConditionIsUse(start) {
		return false
	}
	if f.is(next, "=") {
		q := start - 1
		for q >= 0 && (f.isEmpty(q) || n.isTypeHintToken(q) || f.is(q, "?") && n.nullable[q]) {
			q--
		}
		if n.isConstKeyword(q) {
			return false
		}
	}
	if f.isKeyword(prev, "use") {
		for k := start - 1; k >= 0; k-- {
			if f.isKeyword(k, "class") || f.isKeyword(k, "trait") || f.isKeyword(k, "enum") {
				closer := f.scopeCloser(k)
				return closer >= 0 && start > f.toks[closer].closer && start < closer
			}
		}
		return false
	}
	if f.is(prev, "(") && f.isKeyword(f.prevEffective(prev-1), "declare") {
		return false
	}
	if f.is(prev, ",") && f.refFindPreviousLocal(prev-1, func(i int) bool { return f.isKeyword(i, "declare") }) >= 0 {
		return false
	}
	switch {
	case f.is(prev, ","):
		if c := f.refFindPreviousLocal(prev-1, n.isConstKeyword); c >= 0 {
			array := false
			for j := c + 1; j < start; j++ {
				if f.isShortArrayOpen(j) || f.isKeyword(j, "array") {
					array = true
					break
				}
			}
			if !array {
				return false
			}
		}
	case f.is(prev, "&") && !n.typeOps[prev]:
		before := f.prevEffective(prev - 1)
		if !f.isVariable(before) && f.refFindPreviousLocal(prev-1, n.isFunctionKeyword) >= 0 {
			return false
		}
	case f.isKeyword(prev, "goto"):
		return false
	}
	if n.isFunctionKeyword(prev) || f.is(prev, "::") || f.is(prev, "->") || f.is(prev, "?->") ||
		f.isKeyword(prev, "namespace") || n.isEnumCase(prev) || f.isClassTypeKeyword(prev) {
		return false
	}
	if f.isKeyword(prev, "as") && !f.inParentheses(prev) {
		return false
	}
	name := f.toks[start].content
	if slvrnSimpleTypeHints[name] {
		return f.is(next, "(")
	}
	return true
}

// isTypeHintToken tells a TokenHelper::TYPE_HINT_TOKEN_CODES token.
func (n *slvNames) isTypeHintToken(i int) bool {
	f := n.f
	return f.toks[i].kind == token.Ident || f.isKeyword(i, "callable") || n.isNameToken(i) || n.typeOps[i]
}

func (n *slvNames) innermostConditionIsUse(i int) bool {
	f := n.f
	for o := f.enclosing(i); o >= 0; o = f.enclosing(o) {
		if !f.is(o, "{") {
			continue
		}
		if c := f.scopeCondition(o); c >= 0 {
			return f.isKeyword(c, "use")
		}
	}
	return false
}

func (f *slvFile) inParentheses(i int) bool {
	for o := f.enclosing(i); o >= 0; o = f.enclosing(o) {
		if f.is(o, "(") {
			return true
		}
	}
	return false
}

// refFindPreviousLocal is File::findPrevious with $local: it searches back from
// i, jumping over closed brackets and stopping at a ";".
func (f *slvFile) refFindPreviousLocal(i int, match func(int) bool) int {
	for ; i >= 0; i-- {
		if match(i) {
			return i
		}
		t := f.toks[i]
		if t.kind == token.Punct && (t.orig == ")" || t.orig == "]" || t.orig == "}") && t.closer >= 0 && t.closer < i {
			i = t.closer
			continue
		}
		if f.is(i, ";") {
			break
		}
	}
	return -1
}

// isPropertyHookName is ReferencedNameHelper::isPropertyHookName.
func (n *slvNames) isPropertyHookName(p int) bool {
	f := n.f
	lw := strings.ToLower(f.toks[p].content)
	if lw != "get" && lw != "set" {
		return false
	}
	next := f.nextEffective(p+1, len(f.toks))
	if !f.is(next, "=>") && !f.is(next, "{") && !f.is(next, "(") && !f.is(next, ";") {
		return false
	}
	enc := p
	for {
		enc--
		for enc >= 0 && !f.is(enc, "{") {
			enc--
		}
		if enc < 0 {
			return false
		}
		if c := f.toks[enc].closer; c >= 0 && c >= p {
			break
		}
	}
	if f.scopeCondition(enc) >= 0 {
		return false
	}
	v := enc - 1
	for v >= 0 && !f.isVariable(v) {
		v--
	}
	after := -1
	if v >= 0 {
		after = f.nextEffective(v+1, len(f.toks))
	}
	if after != enc {
		if after < 0 || !f.is(after, "=") || !f.isProperty(v) && !n.isPromotedProperty(v) {
			return false
		}
		stop := -1
		for j := after + 1; j < len(f.toks); j++ {
			if f.is(j, ";") || f.is(j, "{") || f.is(j, "}") {
				stop = j
				break
			}
		}
		if stop != enc {
			return false
		}
	}
	pp := f.prevEffective(p - 1)
	for pp >= 0 {
		if f.toks[pp].kind == token.AttributeEnd && f.toks[pp].closer >= 0 {
			pp = f.prevEffective(f.toks[pp].closer - 1)
		} else if f.isKeyword(pp, "final") || f.toks[pp].content == "&" {
			pp = f.prevEffective(pp - 1)
		} else {
			break
		}
	}
	return pp == enc || f.is(pp, ";") || f.is(pp, "}")
}

// isPromotedProperty tells a constructor parameter with a visibility or readonly
// modifier, which PropertyHelper::isProperty($promoted = true) accepts.
func (n *slvNames) isPromotedProperty(v int) bool {
	f := n.f
	p := v - 1
	for p >= 0 && (f.isEmpty(p) || f.isTypeHintToken(p) || f.is(p, "&") || f.is(p, "...")) {
		p--
	}
	if p < 0 || f.toks[p].kind != token.Keyword {
		return false
	}
	lw := strings.ToLower(f.toks[p].orig)
	if lw != "readonly" && lw != "final" && !strings.HasPrefix(lw, "public") && !strings.HasPrefix(lw, "protected") && !strings.HasPrefix(lw, "private") {
		return false
	}
	o := n.innermostParen(v)
	return o >= 0 && f.isKeyword(n.parenOwner(o), "function")
}

// isNullable is PHP_CodeSniffer's choice between T_NULLABLE and T_INLINE_THEN.
func (n *slvNames) isNullable(i int) bool {
	f := n.f
	if n.isConstKeyword(f.prevEffective(i - 1)) {
		return true
	}
	last := 0 // 1 a name, 2 "array"
	j := i + 1
	for ; j < len(f.toks); j++ {
		if f.isEmpty(j) {
			continue
		}
		if f.toks[j].kind == token.Ident || f.isKeyword(j, "namespace") || f.is(j, `\`) {
			last = 1
			continue
		}
		if f.isKeyword(j, "array") {
			last = 2
			continue
		}
		if last == 0 && !f.isKeyword(j, "callable") ||
			last == 2 && f.is(j, "(") ||
			last == 1 && (f.is(j, "::") || f.is(j, "(") || f.is(j, ":")) {
			return false
		}
		break
	}
	if j == len(f.toks) {
		return false
	}
	found := false
	lastSeen := -1
	for k := i - 1; k >= 0; k-- {
		if f.isKeyword(k, "static") && (f.is(lastSeen, "::") || f.is(lastSeen, "(")) {
			lastSeen = k
			continue
		}
		if !found && !f.isEmpty(k) {
			if f.is(k, ":") || f.is(k, ",") || f.toks[k].kind == token.AttributeEnd {
				return true
			}
			found = true
		}
		if f.toks[k].kind == token.Keyword {
			switch lw := strings.ToLower(f.toks[k].orig); {
			case lw == "function", lw == "fn", lw == "abstract", lw == "final", lw == "static", lw == "var", lw == "readonly",
				strings.HasPrefix(lw, "public"), strings.HasPrefix(lw, "protected"), strings.HasPrefix(lw, "private"):
				return true
			}
		}
		if f.is(k, "=>") || f.toks[k].kind == token.OpenTag || f.is(k, "=") || f.is(k, "{") || f.is(k, ";") {
			return false
		}
		if !f.isEmpty(k) {
			lastSeen = k
		}
	}
	return false
}

// markInlineElse finds the ":" PHP_CodeSniffer makes T_INLINE_ELSE.
func (n *slvNames) markInlineElse() {
	f := n.f
	var stack []int
	for i := range f.toks {
		if f.is(i, "?") {
			if !n.nullable[i] {
				stack = append(stack, i)
			}
			continue
		}
		if !f.is(i, ":") || len(stack) == 0 || n.isGotoColon(i) {
			continue
		}
		p := f.prevEffective(i - 1)
		if n.isParamName(p) {
			continue
		}
		if f.is(p, ")") && f.toks[p].closer >= 0 {
			q := f.toks[p].closer - 1
			for q > 0 && (f.isEmpty(q) || n.isPlainString(q)) {
				q--
			}
			if f.isKeyword(q, "function") || f.isKeyword(q, "fn") || f.isKeyword(q, "use") {
				continue
			}
		}
		top := stack[len(stack)-1]
		isCase := false
		for k := i; k > top; k-- {
			if f.isKeyword(k, "case") || f.isKeyword(k, "default") {
				isCase = true
				break
			}
			if f.is(k, ";") || f.is(k, "{") || f.is(k, "}") {
				break
			}
		}
		if isCase {
			continue
		}
		stack = stack[:len(stack)-1]
		n.inlineElse[i] = true
	}
}

// parenOwner is the parenthesis_owner PHP_CodeSniffer gives an "(", or -1.
func (n *slvNames) parenOwner(open int) int {
	f := n.f
	p := f.prevEffective(open - 1)
	if p < 0 {
		return -1
	}
	if f.toks[p].kind == token.Keyword {
		switch strings.ToLower(f.toks[p].orig) {
		case "array", "list", "function", "use", "class", "while", "for", "foreach", "switch", "if",
			"elseif", "catch", "declare", "match", "isset", "empty", "unset":
			return p
		}
		return -1
	}
	if f.toks[p].kind == token.Ident {
		q := f.prevEffective(p - 1)
		if f.is(q, "&") {
			q = f.prevEffective(q - 1)
		}
		if f.isKeyword(q, "function") {
			return q
		}
		switch strings.ToLower(f.toks[p].orig) {
		case "exit", "die", "eval":
			return p
		}
	}
	return -1
}

// markTypeOperators converts "|", "&" and parentheses of type declarations
// like PHP_CodeSniffer's PHP tokenizer does.
func (n *slvNames) markTypeOperators() {
	f := n.f
	lastSeen := len(f.toks)
	for i := range slices.Backward(f.toks) {
		if !f.is(i, "|") && !f.is(i, "&") && !f.is(i, ")") || n.typeOps[i] || lastSeen < i {
			continue
		}
		closeWithoutOwner := f.is(i, ")") && (f.toks[i].closer < 0 || n.parenOwner(f.toks[i].closer) < 0)
		suspected := ""
		after := 0
		for x := i + 1; x < len(f.toks); x++ {
			if f.isEmpty(x) {
				continue
			}
			if n.typeAllowed(x, true) {
				after++
				continue
			}
			if (after > 0 || closeWithoutOwner) && (f.is(x, "&") && !n.typeOps[x] || f.is(x, "...")) {
				continue
			}
			switch {
			case f.isVariable(x):
				suspected = "param"
			case f.is(x, "=>"), f.is(x, ";"):
				suspected = "return"
			case f.is(x, "{") && f.isKeyword(f.braceOwner(x), "function"):
				suspected = "return"
			case f.is(x, "="):
				suspected = "constant"
			}
			break
		}
		if after == 0 && !closeWithoutOwner || suspected == "" {
			continue
		}
		param := suspected == "param"
		ops := []int{i}
		parens := 0
		if f.is(i, ")") {
			parens++
		}
		before := 0
		confirmed := false
		skip := false
		x := i - 1
	scan:
		for ; x >= 0; x-- {
			if f.isEmpty(x) {
				continue
			}
			if param && f.is(x, "(") {
				if o := n.parenOwner(x); o >= 0 && f.isKeyword(o, "function") {
					confirmed = true
					break
				}
				y := x - 1
				for y > 0 && (f.isEmpty(y) || f.is(y, "&")) {
					y--
				}
				if f.isKeyword(y, "fn") {
					confirmed = true
					break
				}
			}
			if n.typeAllowed(x, !param) {
				before++
				continue
			}
			if before > 0 && f.is(x, "?") {
				continue
			}
			if f.is(x, "|") || f.is(x, "&") {
				ops = append(ops, x)
				continue
			}
			if f.is(x, "(") || f.is(x, ")") {
				parens++
				ops = append(ops, x)
				continue
			}
			if suspected == "return" && f.is(x, ":") {
				y := f.prevEffective(x - 1)
				if !f.is(y, ")") {
					skip = true
					break scan
				}
				open := f.toks[y].closer
				if o := n.parenOwner(open); open >= 0 && o >= 0 {
					confirmed = f.isKeyword(o, "function") || f.isKeyword(o, "use")
					break
				}
				if open >= 0 && f.isKeyword(f.prevEffective(open-1), "fn") {
					confirmed = true
				}
				break
			}
			if suspected == "constant" && n.isConstKeyword(x) {
				confirmed = true
				break
			}
			if param && f.toks[x].kind == token.Keyword {
				switch lw := strings.ToLower(f.toks[x].orig); {
				case lw == "var", lw == "static", lw == "readonly", lw == "final", lw == "abstract",
					strings.HasPrefix(lw, "public"), strings.HasPrefix(lw, "protected"), strings.HasPrefix(lw, "private"):
					confirmed = true
				}
			}
			break
		}
		if skip {
			continue
		}
		lastSeen = x
		if !confirmed && param {
			if last := n.innermostParen(i); last >= 0 {
				if o := n.parenOwner(last); o >= 0 && f.isKeyword(o, "function") {
					confirmed = true
				} else if fn := f.prevEffective(last - 1); f.isKeyword(fn, "fn") {
					y := fn - 1
					for y >= 0 && (f.isEmpty(y) || f.is(y, "&")) {
						y--
					}
					confirmed = !f.isKeyword(y, "function")
				}
			}
		}
		if !confirmed || parens%2 != 0 {
			continue
		}
		for _, o := range ops {
			n.typeOps[o] = true
		}
	}
}

func (n *slvNames) innermostParen(i int) int {
	for o := n.f.enclosing(i); o >= 0; o = n.f.enclosing(o) {
		if n.f.is(o, "(") {
			return o
		}
	}
	return -1
}

// typeAllowed tells a token PHP_CodeSniffer accepts in a type declaration.
func (n *slvNames) typeAllowed(i int, static bool) bool {
	f := n.f
	switch f.toks[i].kind {
	case token.Ident:
		return true
	case token.Keyword:
		if f.isKeyword(i, "static") {
			return static
		}
		return f.isKeyword(i, "callable") || f.isKeyword(i, "array") && n.isNameToken(i)
	}
	return false
}

// refNamespacePointers is NamespaceHelper::getAllNamespacesPointers.
func (f *slvFile) refNamespacePointers() []int {
	var out []int
	for i := range f.toks {
		if f.isKeyword(i, "namespace") && !f.is(f.nextEffective(i+1, len(f.toks)), `\`) {
			out = append(out, i)
		}
	}
	return out
}

// refNamespaceName is NamespaceHelper::getName.
func (f *slvFile) refNamespaceName(ns int) string {
	start := f.nextEffective(ns+1, len(f.toks))
	if start < 0 {
		return ""
	}
	end := start + 1
	for end < len(f.toks) && f.isName(end) {
		end++
	}
	return f.content(start, end-1)
}

// refCurrentNamespace is NamespaceHelper::findCurrentNamespaceName.
func (f *slvFile) refCurrentNamespace(i int) (string, bool) {
	pointers := f.refNamespacePointers()
	for _, p := range slices.Backward(pointers) {
		if p < i {
			return f.refNamespaceName(p), true
		}
	}
	return "", false
}

// slvUseID is UseStatement::getUniqueId in slvUseStatements' key format.
func slvUseID(typ slvRefType, name string) string {
	switch typ {
	case slvRefFunction:
		return "function " + strings.ToLower(name)
	case slvRefConstant:
		return "const " + name
	}
	return strings.ToLower(name)
}

func slvHasNamespace(name string) bool {
	return strings.Contains(strings.TrimLeft(name, `\`), `\`)
}

// resolveName is NamespaceHelper::resolveName, except that a constant PHP
// defines (defined()) still gets the current namespace.
func (u *slvUseStatements) resolveName(f *slvFile, name string, typ slvRefType, pointer int) string {
	if strings.HasPrefix(name, `\`) {
		return name
	}
	uses := u.forPointer(pointer)
	if fqn, ok := uses[slvUseID(typ, strings.TrimLeft(name, `\`))]; ok {
		return `\` + fqn
	}
	parts := strings.Split(strings.TrimLeft(name, `\`), `\`)
	if fqn, ok := uses[slvUseID(typ, parts[0])]; ok && len(parts) > 1 {
		return `\` + fqn + `\` + strings.Join(parts[1:], `\`)
	}
	name = `\` + name
	if ns, ok := f.refCurrentNamespace(pointer); ok {
		name = `\` + ns + name
	}
	return name
}

// referencedNamesInAttributes is ReferencedNameHelper::getAllReferencedNamesInAttributes.
func (n *slvNames) referencedNamesInAttributes(openTag int) []slvReferencedName {
	f := n.f
	var out []slvReferencedName
	for a := openTag + 1; a < len(f.toks); a++ {
		if f.toks[a].kind != token.AttributeStart || f.toks[a].closer < 0 {
			continue
		}
		closer := f.toks[a].closer
		level := 0
		for p := a + 1; p < closer; p++ {
			switch {
			case f.is(p, "("):
				level++
				continue
			case f.is(p, ")"):
				level--
				continue
			case !n.isNameToken(p):
				continue
			}
			typ := slvRefClass
			before := f.prevEffective(p - 1)
			switch {
			case before >= 0 && (f.toks[before].kind == token.OpenTag || f.toks[before].kind == token.AttributeStart):
			case f.is(before, ",") && level == 0:
			case n.isReferencedName(p):
				typ = n.referenceType(p, p)
			default:
				continue
			}
			out = append(out, slvReferencedName{f.toks[p].content, a, closer, typ})
		}
	}
	return out
}
