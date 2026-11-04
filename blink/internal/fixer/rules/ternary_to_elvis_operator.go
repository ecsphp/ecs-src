package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/TernaryToElvisOperatorFixer.php
//
// TernaryToElvisOperator turns "$a ? $a : $b" into "$a ?: $b" when the
// condition and the middle operand are the same tokens.
type TernaryToElvisOperator struct{}

func (TernaryToElvisOperator) Name() string {
	return `PhpCsFixer\Fixer\Operator\TernaryToElvisOperatorFixer`
}

func (TernaryToElvisOperator) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Operator/TernaryToElvisOperatorFixer.php"
}

// elvis bracket classes: PHP-CS-Fixer's block types, which tell a plain "(",
// "[" or "{" from the custom tokens its transformers make of some of them.
const (
	elvisNone         uint8 = iota
	elvisParen              // ( ) BLOCK_TYPE_PARENTHESIS
	elvisNewParen           // (new Foo) BLOCK_TYPE_CLASS_INSTANTIATION_PARENTHESIS
	elvisIndex              // $a[...] BLOCK_TYPE_INDEX_BRACKET
	elvisArray              // [...] BLOCK_TYPE_ARRAY_BRACKET
	elvisDestructure        // [$a] = BLOCK_TYPE_DESTRUCTURING_BRACKET
	elvisBrace              // { } BLOCK_TYPE_BRACE
	elvisDynamicBrace       // $a->{...}, ${...}, Foo::{...}, use A\{...}
	elvisAttribute          // #[ ]
)

// elvisView holds the bracket pairs, bracket classes and casts of the stream.
// PHP lexes a cast like "(int)" as one token, so a cast is handled as one unit,
// named by the index of its "(".
type elvisView struct {
	s         *tokens.Stream
	match     []int
	class     []uint8
	castEnd   []int // at a cast "(": its ")"; else -1
	castStart []int // at a cast ")": its "("; else -1
	phpPos    []int // PHP-CS-Fixer token index of each blink token
	phpCount  int
}

var elvisCastTypes = map[string]bool{
	"int": true, "integer": true, "bool": true, "boolean": true, "float": true,
	"double": true, "string": true, "binary": true, "array": true, "object": true,
	"unset": true, "void": true,
}

func newElvisView(s *tokens.Stream) *elvisView {
	n := s.Len()
	v := &elvisView{
		s: s, match: make([]int, n), class: make([]uint8, n),
		castEnd: make([]int, n), castStart: make([]int, n), phpPos: make([]int, n),
	}
	for i := range n {
		v.match[i], v.castEnd[i], v.castStart[i] = -1, -1, -1
	}
	var stack []int
	for i := range n {
		t := s.At(i)
		switch {
		case t.Kind == token.AttributeStart || (t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{")):
			stack = append(stack, i)
		case t.Kind == token.AttributeEnd || (t.Kind == token.Punct && (t.Value == ")" || t.Value == "]" || t.Value == "}")):
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			if !elvisPairs(s.At(open), t) {
				continue
			}
			stack = stack[:len(stack)-1]
			v.match[open], v.match[i] = i, open
		}
	}
	for i := range n {
		if isPunctVal(s, i, "(") {
			if end := elvisCastAt(s, i); end >= 0 {
				v.castEnd[i], v.castStart[end] = end, i
			}
		}
	}
	for i := range n {
		t := s.At(i)
		if t.Kind == token.AttributeStart {
			v.setClass(i, elvisAttribute)
			continue
		}
		if t.Kind != token.Punct || v.class[i] != elvisNone || v.castEnd[i] >= 0 {
			continue
		}
		switch t.Value {
		case "(":
			v.setClass(i, v.parenClass(i))
		case "[":
			v.classifySquare(i)
		case "{":
			v.setClass(i, v.braceClass(i))
		}
	}
	// PHP token positions, for the fixer's "count - 5" loop bound
	pos := 0
	for i := 0; i < n; i++ {
		v.phpPos[i] = pos
		t := s.At(i)
		switch {
		case v.castEnd[i] >= 0:
			for j := i + 1; j <= v.castEnd[i]; j++ {
				v.phpPos[j] = pos
			}
			i = v.castEnd[i]
		case t.Kind == token.OpenTag && t.Value == "<?php" && i+1 < n && elvisAbsorbedByTag(s.At(i+1).Value):
			v.phpPos[i+1] = pos // "<?php " owns its single whitespace character
			i++
		case t.Kind == token.CloseTag && i+1 < n && s.At(i+1).Kind == token.InlineHTML &&
			(s.At(i+1).Value == "\n" || s.At(i+1).Value == "\r\n"):
			v.phpPos[i+1] = pos // "?>\n" owns its newline
			i++
		}
		pos++
	}
	v.phpCount = pos
	return v
}

func elvisAbsorbedByTag(ws string) bool {
	return ws == " " || ws == "\t" || ws == "\n" || ws == "\r\n" || ws == "\r"
}

func elvisPairs(open, closeTok token.Token) bool {
	if open.Kind == token.AttributeStart || closeTok.Kind == token.AttributeEnd {
		return open.Kind == token.AttributeStart && closeTok.Kind == token.AttributeEnd
	}
	return open.Value+closeTok.Value == "()" || open.Value+closeTok.Value == "[]" || open.Value+closeTok.Value == "{}"
}

// elvisCastAt returns the ")" of the cast starting at the "(" at open (PHP:
// "(" [ \t]* type [ \t]* ")"), or -1.
func elvisCastAt(s *tokens.Stream, open int) int {
	j := open + 1
	if j < s.Len() && s.At(j).Kind == token.Whitespace && strings.Trim(s.At(j).Value, " \t") == "" {
		j++
	}
	if j >= s.Len() || (s.At(j).Kind != token.Ident && s.At(j).Kind != token.Keyword) || !elvisCastTypes[strings.ToLower(s.At(j).Value)] {
		return -1
	}
	j++
	if j < s.Len() && s.At(j).Kind == token.Whitespace && strings.Trim(s.At(j).Value, " \t") == "" {
		j++
	}
	if !isPunctVal(s, j, ")") {
		return -1
	}
	return j
}

func (v *elvisView) setClass(open int, c uint8) {
	v.class[open] = c
	if m := v.match[open]; m >= 0 {
		v.class[m] = c
	}
}

// parenClass mirrors BraceClassInstantiationTransformer.
func (v *elvisView) parenClass(i int) uint8 {
	s := v.s
	if !gfcIsKeyword(s, sigNext(s, i), "new") {
		return elvisParen
	}
	p := sigPrev(s, i)
	if p < 0 {
		return elvisNewParen
	}
	t := s.At(p)
	switch t.Kind {
	case token.Punct:
		if t.Value == ")" || t.Value == "]" {
			return elvisParen
		}
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "array", "class", "elseif", "for", "foreach", "if", "static", "switch", "while":
			return elvisParen
		}
	case token.Ident:
		if gfcIsTString(s, p) {
			return elvisParen
		}
	case token.Variable:
		if t.Value != "$" {
			return elvisParen
		}
	}
	return elvisNewParen
}

// braceClass tells a code-block brace from the custom brace tokens
// (BraceTransformer).
func (v *elvisView) braceClass(i int) uint8 {
	s := v.s
	// transformIntoDynamicPropBraces: "{" right after "->" or "?->"
	if i > 0 && (isPunctVal(s, i-1, "->") || isPunctVal(s, i-1, "?->")) {
		return elvisDynamicBrace
	}
	p := sigPrev(s, i)
	if p < 0 {
		return elvisBrace
	}
	t := s.At(p)
	// transformIntoDynamicVarBraces and transformIntoGroupUseBraces
	if (t.Kind == token.Variable && t.Value == "$") || (t.Kind == token.Punct && t.Value == `\`) {
		return elvisDynamicBrace
	}
	if v.isDynamicClassConstantFetchBrace(i, p) {
		return elvisDynamicBrace
	}
	return elvisBrace
}

// isDynamicClassConstantFetchBrace mirrors
// BraceTransformer::transformIntoDynamicClassConstantFetchBraces for the "{"
// at i, whose previous meaningful token is p.
func (v *elvisView) isDynamicClassConstantFetchBrace(i, p int) bool {
	s := v.s
	for !isPunctVal(s, p, "::") {
		if !isPunctVal(s, p, ")") || v.match[p] < 0 {
			return false
		}
		p = sigPrev(s, v.match[p])
		if p < 0 || !isPunctVal(s, p, "}") || v.match[p] < 0 {
			return false
		}
		p = sigPrev(s, v.match[p])
		if p < 0 {
			return false
		}
	}
	end := v.match[i]
	if end < 0 {
		return false
	}
	next := sigNext(s, end)
	if next < 0 {
		return false
	}
	return s.At(next).Kind == token.CloseTag || isPunctVal(s, next, ";") || isPunctVal(s, next, "::")
}

// classifySquare mirrors SquareBraceTransformer for the "[" at i.
func (v *elvisView) classifySquare(i int) {
	s := v.s
	p := sigPrev(s, i)
	valueBefore := func(withBrace bool) bool {
		if p < 0 {
			return false
		}
		t := s.At(p)
		switch t.Kind {
		case token.Punct:
			switch t.Value {
			case ")", "]":
				return true
			case "}":
				return withBrace || v.class[p] == elvisDynamicBrace
			}
		case token.String:
			return !strings.HasPrefix(t.Value, "<<<")
		case token.Ident:
			return gfcIsTString(s, p)
		case token.Variable:
			return t.Value != "$"
		}
		return false
	}
	if !valueBefore(false) && v.isDestructuring(i, p) {
		v.markDestructuring(i)
		return
	}
	if valueBefore(true) {
		v.setClass(i, elvisIndex)
		return
	}
	v.setClass(i, elvisArray)
}

func (v *elvisView) isDestructuring(i, p int) bool {
	s := v.s
	if p >= 0 && kwIs(s.At(p), "as") {
		return true
	}
	if isPunctVal(s, p, "=>") {
		variable := sigPrev(s, p)
		if gfcIsTVariable(s, variable) && gfcIsKeyword(s, sigPrev(s, variable), "as") {
			return true
		}
		if !gfcIsTVariable(s, variable) {
			return false
		}
	}
	end := v.match[i]
	return end >= 0 && isPunctVal(s, sigNext(s, end), "=")
}

func (v *elvisView) markDestructuring(i int) {
	v.setClass(i, elvisDestructure)
	end := v.match[i]
	prev := i
	for j := sigNext(v.s, i); j >= 0 && j < end; j = sigNext(v.s, j) {
		if isPunctVal(v.s, j, "[") && (prev == i || isPunctVal(v.s, prev, ",")) {
			v.markDestructuring(j)
		}
		prev = j
	}
}

// unitEnd is the last blink token of the unit starting at i.
func (v *elvisView) unitEnd(i int) int {
	if v.castEnd[i] >= 0 {
		return v.castEnd[i]
	}
	return i
}

func (v *elvisView) prev(i int) int {
	j := sigPrev(v.s, i)
	if j >= 0 && v.castStart[j] >= 0 {
		return v.castStart[j]
	}
	return j
}

func (v *elvisView) next(i int) int {
	return sigNext(v.s, v.unitEnd(i))
}

// blockType is Tokens::detectBlockType: the bracket class and whether the
// token opens the block.
func (v *elvisView) blockType(i int) (uint8, bool) {
	t := v.s.At(i)
	switch t.Kind {
	case token.AttributeStart:
		return elvisAttribute, true
	case token.AttributeEnd:
		return elvisAttribute, false
	case token.Punct:
		if v.castEnd[i] >= 0 || v.castStart[i] >= 0 {
			return elvisNone, false
		}
		switch t.Value {
		case "(", "[", "{":
			return v.class[i], true
		case ")", "]", "}":
			return v.class[i], false
		}
	}
	return elvisNone, false
}

var elvisValidBeforeOps = map[string]bool{
	"=": true, ",": true, ";": true, "&=": true, ".=": true, "/=": true, "-=": true,
	"%=": true, "*=": true, "|=": true, "+=": true, "**=": true, "<<=": true,
	">>=": true, "^=": true,
}

// validBefore mirrors VALID_BEFORE_ENDTYPES: the tokens ending the condition.
func (v *elvisView) validBefore(i int) bool {
	t := v.s.At(i)
	switch t.Kind {
	case token.OpenTag:
		return true
	case token.Punct:
		if elvisValidBeforeOps[t.Value] {
			return true
		}
		cls, _ := v.blockType(i)
		switch t.Value {
		case "(":
			return cls == elvisParen
		case "[":
			return cls == elvisIndex
		case "{", "}":
			return cls == elvisBrace
		}
	}
	return false
}

func (v *elvisView) isIncDec(i int) bool {
	return isPunctVal(v.s, i, "++") || isPunctVal(v.s, i, "--")
}

// isNullableType approximates NullableTypeTransformer: a "?" before a type in
// a parameter, property, constant or return type is not a ternary.
func (v *elvisView) isNullableType(i int) bool {
	p := sigPrev(v.s, i)
	if p < 0 {
		return false
	}
	t := v.s.At(p)
	switch t.Kind {
	case token.Punct:
		return t.Value == "(" || t.Value == "," || t.Value == ":" || t.Value == "|"
	case token.AttributeEnd:
		return true
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "public", "protected", "private", "var", "static", "readonly", "final", "abstract", "const":
			return true
		}
	}
	return false
}

// isElseColon reports a plain ":" (not a return type colon or a named
// argument colon, which are custom tokens in PHP-CS-Fixer).
func (v *elvisView) isElseColon(i int) bool {
	s := v.s
	if !isPunctVal(s, i, ":") {
		return false
	}
	p := sigPrev(s, i)
	if isPunctVal(s, p, ")") && v.match[p] >= 0 {
		before := sigPrev(s, v.match[p])
		if gfcIsTString(s, before) {
			before = sigPrev(s, before)
		}
		if isPunctVal(s, before, "&") {
			before = sigPrev(s, before)
		}
		if gfcIsKeyword(s, before, "fn") || gfcIsKeyword(s, before, "function") || gfcIsKeyword(s, before, "use") {
			return false
		}
	}
	if p >= 0 && (s.At(p).Kind == token.Ident || s.At(p).Kind == token.Keyword) {
		if pp := sigPrev(s, p); isPunctVal(s, pp, "(") || isPunctVal(s, pp, ",") {
			return false
		}
	}
	return true
}

type elvisRange struct{ start, end int }

// before is getBeforeOperator: the condition, or false when it holds "++" or
// "--" or is empty.
func (v *elvisView) before(index int) (elvisRange, bool) {
	index = v.prev(index)
	r := elvisRange{start: -1, end: index}
	for index >= 0 && !v.validBefore(index) {
		if v.isIncDec(index) {
			return r, false
		}
		cls, isStart := v.blockType(index)
		if cls == elvisNone || isStart {
			r.start = index
			index = v.prev(index)
			continue
		}
		openCount := 1
		for {
			index = v.prev(index)
			if index < 0 || v.isIncDec(index) {
				return r, false
			}
			if c, st := v.blockType(index); c == cls && st {
				openCount++
			} else if c == cls && !st {
				openCount--
			}
			if openCount > 1 {
				break
			}
		}
		r.start = index
		index = v.prev(index)
	}
	return r, r.start >= 0
}

// after is getAfterOperator: the operand between "?" and ":".
func (v *elvisView) after(index int) (elvisRange, bool) {
	index = v.next(index)
	r := elvisRange{start: index}
	for {
		if index < 0 {
			return r, false
		}
		if cls, isStart := v.blockType(index); cls != elvisNone {
			if !isStart || v.match[index] < 0 {
				return r, false // findBlockEnd on a closing token throws upstream
			}
			index = v.match[index]
		}
		r.end = index
		index = v.next(index)
		if index < 0 {
			return r, false
		}
		if v.isElseColon(index) {
			return r, true
		}
	}
}

func (v *elvisView) unitText(i int) string {
	if v.castEnd[i] < 0 {
		return v.s.At(i).Value
	}
	var b strings.Builder
	for j := i; j <= v.castEnd[i]; j++ {
		b.WriteString(v.s.At(j).Value)
	}
	return b.String()
}

func (v *elvisView) unitEquals(a, b int) bool {
	if (v.castEnd[a] >= 0) != (v.castEnd[b] >= 0) {
		return false
	}
	ta, tb := v.s.At(a), v.s.At(b)
	if ta.Kind != tb.Kind || v.unitText(a) != v.unitText(b) {
		return false
	}
	ca, _ := v.blockType(a)
	cb, _ := v.blockType(b)
	return ca == cb
}

// isPlain reports a plain "(" or ")" (not a cast, not a class instantiation).
func (v *elvisView) isPlain(i int, val string) bool {
	if !isPunctVal(v.s, i, val) {
		return false
	}
	cls, _ := v.blockType(i)
	return cls == elvisParen
}

// rangeEquals is RangeAnalyzer::rangeEqualsRange: a meaningful-token compare
// of two ranges, each stripped of wrapping parentheses.
func (v *elvisView) rangeEquals(r1, r2 elvisRange) bool {
	strip := func(r elvisRange) elvisRange {
		for r.start >= 0 && r.end >= 0 && v.isPlain(r.start, "(") && v.isPlain(r.end, ")") {
			r.start, r.end = v.next(r.start), v.prev(r.end)
		}
		return r
	}
	l, rr := strip(r1), strip(r2)
	if l.start < 0 || rr.start < 0 {
		return false
	}
	ls, rs := l.start, rr.start
	for {
		if !v.unitEquals(ls, rs) {
			return false
		}
		ls, rs = v.next(ls), v.next(rs)
		reachedLeft := ls < 0 || ls > l.end
		reachedRight := rs < 0 || rs > rr.end
		if !reachedLeft && !reachedRight {
			continue
		}
		return reachedLeft && reachedRight
	}
}

// clearUnit is Tokens::clearTokenAndMergeSurroundingWhitespace for a unit.
func (v *elvisView) clearUnit(a int) {
	for j := a + 1; j <= v.unitEnd(a); j++ {
		clearTokenAt(v.s, j)
	}
	clearTokenAndMergeWS(v.s, a)
}

func (TernaryToElvisOperator) Fix(s *tokens.Stream) bool {
	// indices below assume no cleared placeholders, as after PHP-CS-Fixer's
	// clearEmptyTokens between fixers
	s.ClearEmptyTokens()
	var v *elvisView
	changed := false
	for index := s.Len() - 1; index > 1; index-- {
		if !isPunctVal(s, index, "?") {
			continue
		}
		if v == nil {
			v = newElvisView(s)
		}
		if p := v.phpPos[index]; p > v.phpCount-5 || p <= 1 || v.isNullableType(index) {
			continue
		}
		if isPunctVal(s, v.next(index), ":") {
			continue
		}
		before, ok := v.before(index)
		if !ok {
			continue // contains something we cannot fix because of priorities
		}
		after, ok := v.after(index)
		if !ok || !v.rangeEquals(before, after) {
			continue
		}
		for i := after.end; i >= after.start && i >= 0; i = v.prev(i) {
			v.clearUnit(i)
		}
		changed = true
	}
	if changed {
		s.ClearEmptyTokens()
	}
	return changed
}
