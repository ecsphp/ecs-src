package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/lexer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// yodaMode is how a comparison group is normalized. The zero value is
// yodaNonYoda (option false).
type yodaMode int

const (
	yodaNonYoda yodaMode = iota // false: move the variable to the left
	yodaLeave                   // null: leave the comparison as it is
	yodaYoda                    // true: move the constant to the left
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/YodaStyleFixer.php
//
// YodaStyle is a port of YodaStyleFixer. It rewrites comparisons to the
// configured style per operator group:
// false is non-yoda (a constant on the left and a variable on the right are
// swapped so the variable comes first, "null === $x" -> "$x === null"), null
// leaves the group alone, true is yoda. The struct zero value is non-yoda for
// all three groups (ECS's common set); the registry hands out the rule with
// PHP-CS-Fixer's defaults instead (equal=true, identical=true,
// less_and_greater=null), see phpDefaultsJSON.
type YodaStyle struct {
	equal       yodaMode
	identical   yodaMode
	lessGreater yodaMode
	// alwaysMoveVariable is always_move_variable: the strict variable check,
	// and a parenthesized side counts as a variable
	alwaysMoveVariable bool
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
	if v, ok := config["always_move_variable"].(bool); ok {
		f.alwaysMoveVariable = v
	}
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

// yodaMirror flips a relational operator when its operands are swapped, so
// "5 > $z" becomes "$z < 5" (YodaStyleFixer's candidatesMap). Equality
// operators are symmetric and unchanged.
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

// isInterpolatedString reports whether t is a double-quoted string that contains
// a variable interpolation, making it a runtime value rather than a constant.
func isInterpolatedString(t token.Token) bool {
	if t.Kind != token.String {
		return false
	}
	_, v := token.SplitBinaryPrefix(t.Value)
	if len(v) == 0 || v[0] != '"' {
		return false
	}
	for i := 1; i < len(v); i++ {
		if v[i] == '\\' {
			i++
			continue
		}
		if v[i] == '$' || (v[i] == '{' && i+1 < len(v) && v[i+1] == '$') {
			return true
		}
	}
	return false
}

// candidate returns the mode of the comparison at i, and whether it is one of
// the configured candidate operators.
func (f YodaStyle) candidate(t token.Token) (bool, bool) {
	if t.Kind != token.Punct {
		return false, false
	}
	var m yodaMode
	switch t.Value {
	case "==", "!=", "<>":
		m = f.equal
	case "===", "!==":
		m = f.identical
	case "<", ">", "<=", ">=":
		m = f.lessGreater
	default:
		return false, false
	}
	if m == yodaLeave {
		return false, false
	}
	return m == yodaYoda, true
}

func (f YodaStyle) Fix(s *tokens.Stream) bool {
	before := s.Render()
	f.fixTokens(s)
	return s.Render() != before
}

// fixTokens is YodaStyleFixer::fixTokens: comparisons right to left; after a
// swap the scan resumes left of the comparison's left side.
func (f YodaStyle) fixTokens(s *tokens.Stream) {
	for i := s.Len() - 1; i > 1; i-- {
		yoda, ok := f.candidate(s.At(i))
		if !ok {
			continue
		}
		ls, le, rs, re, ok := f.compareFixableInfo(s, i, yoda)
		if !ok {
			continue
		}
		i = f.fixTokensCompare(s, ls, le, i, rs, re)
	}
}

// fixTokensCompare swaps the two sides of the comparison at op (each side
// first fixed on its own, as fixTokensComparePart does) and returns the start
// of the left side.
func (f YodaStyle) fixTokensCompare(s *tokens.Stream, ls, le, op, rs, re int) int {
	right := f.fixComparePart(s, rs, re)
	left := f.fixComparePart(s, ls, le)
	mid := append([]token.Token(nil), s.Tokens()[le+1:rs]...)
	mid[op-le-1].Value = yodaMirror(mid[op-le-1].Value)
	repl := make([]token.Token, 0, len(right)+len(mid)+len(left))
	repl = append(append(append(repl, right...), mid...), left...)
	s.ReplaceRange(ls, re, repl)
	return ls
}

// fixComparePart re-tokenizes one side as "<?php SIDE;", fixes it on its own
// and returns its tokens.
func (f YodaStyle) fixComparePart(s *tokens.Stream, start, end int) []token.Token {
	var b strings.Builder
	b.WriteString("<?php ")
	for k := start; k <= end; k++ {
		b.WriteString(s.At(k).Value)
	}
	b.WriteString(";")
	part := tokens.New(lexer.Lex(b.String()))
	f.fixTokens(part)
	toks := part.Tokens()
	first := 1 // the open tag; its trailing space is a token of its own here
	if first < len(toks) && toks[first].Kind == token.Whitespace && toks[first].Value == " " {
		first++
	}
	return append([]token.Token(nil), toks[first:len(toks)-1]...)
}

// compareFixableInfo is getCompareFixableInfo.
func (f YodaStyle) compareFixableInfo(s *tokens.Stream, op int, yoda bool) (ls, le, rs, re int, ok bool) {
	rs, re = nextMeaningfulIndex(s, op), yodaComparisonEnd(s, op)
	if rs < 0 || re < rs {
		return 0, 0, 0, 0, false
	}
	if !yoda {
		if n := nextMeaningfulIndex(s, re); n >= 0 && yodaIsLowerPrecedenceAssignment(s.At(n)) {
			return 0, 0, 0, 0, false
		}
	}
	ls, le = yodaComparisonStart(s, op), prevMeaningfulIndex(s, op)
	if ls < 0 || le < ls {
		return 0, 0, 0, 0, false
	}
	if yodaIsListStatement(s, ls, le) || yodaIsListStatement(s, rs, re) {
		return 0, 0, 0, 0, false
	}
	strict := f.alwaysMoveVariable
	leftVar := yodaIsVariable(s, ls, le, strict)
	rightVar := yodaIsVariable(s, rs, re, strict)
	if leftVar == rightVar {
		return 0, 0, 0, 0, false
	}
	if !strict {
		leftVar = leftVar && !yodaIsGroupParen(s, ls)
		rightVar = rightVar && !yodaIsGroupParen(s, rs)
	}
	if (yoda && !leftVar) || (!yoda && !rightVar) {
		return 0, 0, 0, 0, false
	}
	return ls, le, rs, re, true
}

// yodaIsGroupParen reports whether the token at i is a "(" token in PHP's
// sense: a cast "(int)" is a single token there, and the parenthesis around
// "(new Foo())" is CT::T_CLASS_INSTANTIATION_PARENTHESIS_OPEN.
func yodaIsGroupParen(s *tokens.Stream, i int) bool {
	t := s.At(i)
	return t.Kind == token.Punct && t.Value == "(" && !isCastParen(s, i) && !yodaIsClassInstantiationParen(s, i)
}

// yodaIsClassInstantiationParen is BraceClassInstantiationTransformer's test
// for the "(" at i.
func yodaIsClassInstantiationParen(s *tokens.Stream, i int) bool {
	if n := nextMeaningfulIndex(s, i); n < 0 || !kwIs(s.At(n), "new") {
		return false
	}
	p := prevMeaningfulIndex(s, i)
	if p < 0 {
		return true
	}
	t := s.At(p)
	switch t.Kind {
	case token.Variable, token.Ident:
		return false
	case token.Punct:
		return t.Value != ")" && t.Value != "]" && t.Value != "}"
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "array", "class", "elseif", "for", "foreach", "if", "static", "switch", "while":
			return false
		}
	}
	return true
}

// yodaBlock reports whether the token at i opens or closes a block
// (Tokens::detectBlockType): parentheses, brackets, braces and attributes. A
// cast is a single token in PHP, not a block.
func yodaBlock(s *tokens.Stream, i int) (isBlock, isStart bool) {
	t := s.At(i)
	switch t.Kind {
	case token.AttributeStart:
		return true, true
	case token.AttributeEnd:
		return true, false
	case token.Punct:
		switch t.Value {
		case "(":
			return !isCastParen(s, i), true
		case "[", "{":
			return true, true
		case ")":
			if o := s.MatchBackward(i); o >= 0 && isCastParen(s, o) {
				return false, false
			}
			return true, false
		case "]", "}":
			return true, false
		}
	}
	return false, false
}

func yodaIsLowerPrecedenceAssignment(t token.Token) bool {
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case "=", "&=", ".=", "/=", "-=", "%=", "*=", "|=", "+=", "**=", "<<=", ">>=", "^=", "??=":
		return true
	}
	return false
}

// yodaIsLowerPrecedence is isOfLowerPrecedence for the token at i.
func yodaIsLowerPrecedence(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if yodaIsLowerPrecedenceAssignment(t) {
		return true
	}
	switch t.Kind {
	case token.OpenTag:
		return true
	case token.Punct:
		switch t.Value {
		case "&&", "||", "=>", "??", "&", "|", "^", "?", ":", ",", ";":
			return true
		}
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "case", "echo", "goto", "and", "or", "xor", "print", "return", "throw", "yield",
			"require", "require_once", "include", "include_once":
			return true
		}
	case token.Ident:
		// "from" of "yield from" (a single T_YIELD_FROM token in PHP)
		if strings.EqualFold(t.Value, "from") {
			if p := sigPrev(s, i); p >= 0 && kwIs(s.At(p), "yield") {
				return true
			}
		}
	}
	return false
}

// yodaComparisonEnd is findComparisonEnd: the last token of the right side of
// the comparison at op.
func yodaComparisonEnd(s *tokens.Stream, op int) int {
	index := op + 1
	for index < s.Len() {
		t := s.At(index)
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			index++
			continue
		}
		if yodaIsLowerPrecedence(s, index) {
			break
		}
		isBlock, isStart := yodaBlock(s, index)
		if !isBlock {
			index++
			continue
		}
		if !isStart {
			break
		}
		end := s.MatchForward(index)
		if t.Kind == token.AttributeStart {
			end = s.AttributeEnd(index)
		}
		if end < 0 {
			return -1
		}
		index = end + 1
	}
	prev := prevMeaningfulIndex(s, index)
	// PHP's T_CLOSE_TAG holds the newline that follows "?>"
	if prev > 0 && s.At(prev).Kind == token.InlineHTML && s.At(prev-1).Kind == token.CloseTag &&
		(s.At(prev).Value == "\n" || s.At(prev).Value == "\r\n") {
		prev--
	}
	if prev >= 0 && s.At(prev).Kind == token.CloseTag {
		return prevMeaningfulIndex(s, prev)
	}
	return prev
}

// yodaIsPlainBraceClose reports whether the "}" at i closes a code-block brace
// (BLOCK_TYPE_BRACE), not a dynamic property, variable or index brace.
func yodaIsPlainBraceClose(s *tokens.Stream, i int) bool {
	t := s.At(i)
	return t.Kind == token.Punct && t.Value == "}" && !unaryOperatorSpacesIsDynamicBraceClose(s, i)
}

// yodaComparisonStart is findComparisonStart: the first token of the left side
// of the comparison at op.
func yodaComparisonStart(s *tokens.Stream, op int) int {
	index := op - 1
	nonBlockFound := false
	for index >= 0 {
		t := s.At(index)
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			index--
			continue
		}
		if yodaIsLowerPrecedence(s, index) {
			break
		}
		isBlock, isStart := yodaBlock(s, index)
		if !isBlock {
			index--
			nonBlockFound = true
			continue
		}
		if isStart || (nonBlockFound && yodaIsPlainBraceClose(s, index)) {
			break
		}
		start := s.MatchBackward(index)
		if t.Kind == token.AttributeEnd {
			start = s.AttributeStartOf(index)
		}
		if start < 0 {
			return -1
		}
		index = start - 1
	}
	return nextMeaningfulIndex(s, index)
}

// yodaIsDestructuringBracket reports whether the "[" or "]" at i belongs to a
// destructuring "[...]" (CT::T_DESTRUCTURING_BRACKET_OPEN/CLOSE).
func yodaIsDestructuringBracket(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Punct {
		return false
	}
	open := i
	switch t.Value {
	case "[":
	case "]":
		if open = s.MatchBackward(i); open < 0 {
			return false
		}
	default:
		return false
	}
	return !isOffsetOpen(s, open) && arrayNotationIsDestructuring(s, open)
}

// yodaIsListStatement is isListStatement.
func yodaIsListStatement(s *tokens.Stream, from, to int) bool {
	for i := from; i <= to; i++ {
		if kwIs(s.At(i), "list") || yodaIsDestructuringBracket(s, i) {
			return true
		}
	}
	return false
}

func yodaIsObjectOperator(t token.Token) bool {
	return t.Kind == token.Punct && (t.Value == "->" || t.Value == "?->")
}

// yodaIsIndexOpen reports whether the token at i is "[" or an index "{"
// (the bracket list isVariable accepts after a variable or a property name).
func yodaIsIndexOpen(t token.Token) bool {
	return t.Kind == token.Punct && (t.Value == "[" || t.Value == "{")
}

// yodaIsBinaryOperator approximates TokensAnalyzer::isBinaryOperator for the
// strict (always_move_variable) variable check.
func yodaIsBinaryOperator(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "and", "or", "xor":
			return true
		}
		return false
	case token.Punct:
	default:
		return false
	}
	switch t.Value {
	case "+", "-", "&":
		return !unaryOperatorSpacesIsPredecessor(s, i)
	case "=", "*", "/", "%", "<", ">", "|", "^", ".", "**", "<<", ">>", "==", "!=", "<>", "===", "!==",
		"<=", ">=", "<=>", "&&", "||", "??", "+=", "-=", "*=", "/=", ".=", "%=", "**=", "&=", "|=", "^=",
		"<<=", ">>=", "??=":
		return true
	}
	return false
}

// yodaIsVariable is isVariable: whether [start, end] describes a variable.
func yodaIsVariable(s *tokens.Stream, start, end int, strict bool) bool {
	if start == end {
		return s.At(start).Kind == token.Variable
	}
	if yodaIsGroupParen(s, start) {
		return true
	}
	if strict {
		for index := start; index <= end; index++ {
			t := s.At(index)
			if (t.Kind == token.Punct && t.Value == "(" && isCastParen(s, index)) || kwIs(t, "instanceof") ||
				(t.Kind == token.Punct && t.Value == "!") || yodaIsBinaryOperator(s, index) {
				return false
			}
		}
	}
	index := start
	// multiple braces around the statement: ((($a === 1)))
	for yodaIsGroupParen(s, index) && s.MatchForward(index) == end {
		index = nextMeaningfulIndex(s, index)
		end = prevMeaningfulIndex(s, end)
	}
	expectString := false
	for index <= end {
		current := s.At(index)
		if current.Kind == token.Whitespace || current.Kind == token.Comment || current.Kind == token.DocComment {
			index++
			continue
		}
		if index == end {
			if expectString {
				return current.Kind == token.Ident
			}
			return current.Kind == token.Variable
		}
		if kwIs(current, "list") || yodaIsDestructuringBracket(s, index) {
			return false
		}
		nextIndex := nextMeaningfulIndex(s, index)
		if nextIndex < 0 {
			break
		}
		next := s.At(nextIndex)
		// self:: or ClassName::
		if current.Kind == token.Ident && next.Kind == token.Punct && next.Value == "::" {
			index = nextMeaningfulIndex(s, nextIndex)
			if index < 0 {
				break
			}
			continue
		}
		// \ClassName
		if current.Kind == token.Punct && current.Value == `\` && next.Kind == token.Ident {
			index = nextIndex
			continue
		}
		// ClassName\
		if current.Kind == token.Ident && next.Kind == token.Punct && next.Value == `\` {
			index = nextIndex
			continue
		}
		// $a-> or a-> (as in $b->a->c)
		if (current.Kind == token.Ident || current.Kind == token.Variable) && yodaIsObjectOperator(next) {
			index = nextMeaningfulIndex(s, nextIndex)
			if index < 0 {
				break
			}
			expectString = true
			continue
		}
		// $a[...], a[...] (as in $c->a[$b]), $a{...} or a{...}
		wantKind := token.Variable
		if expectString {
			wantKind = token.Ident
		}
		if current.Kind == wantKind && yodaIsIndexOpen(next) {
			index = s.MatchForward(nextIndex)
			if index < 0 {
				return false
			}
			if index == end {
				return true
			}
			index = nextMeaningfulIndex(s, index)
			if index < 0 || (!yodaIsIndexOpen(s.At(index)) && !yodaIsObjectOperator(s.At(index))) {
				return false
			}
			index = nextMeaningfulIndex(s, index)
			if index < 0 {
				return false
			}
			expectString = true
			continue
		}
		// $a(...) or $a->b(...)
		if strict && (current.Kind == token.Ident || current.Kind == token.Variable) && next.Kind == token.Punct && next.Value == "(" {
			return false
		}
		// {...} (as in $a->{$b})
		if expectString && current.Kind == token.Punct && current.Value == "{" {
			index = s.MatchForward(index)
			if index < 0 {
				return false
			}
			if index == end {
				return true
			}
			index = nextMeaningfulIndex(s, index)
			if index < 0 || !yodaIsObjectOperator(s.At(index)) {
				return false
			}
			index = nextMeaningfulIndex(s, index)
			if index < 0 {
				return false
			}
			expectString = true
			continue
		}
		break
	}
	return !yodaIsConstant(s, start, end)
}

// yodaIsPlainString reports whether t is a T_CONSTANT_ENCAPSED_STRING: a quoted
// string without interpolation (heredocs and nowdocs are not).
func yodaIsPlainString(t token.Token) bool {
	if t.Kind != token.String || t.Value == "" {
		return false
	}
	switch t.Value[0] {
	case '\'':
		return true
	case '"':
		for i := 1; i < len(t.Value)-1; i++ {
			if t.Value[i] == '\\' {
				i++
				continue
			}
			if t.Value[i] == '$' || (t.Value[i] == '{' && t.Value[i+1] == '$') {
				return false
			}
		}
		return true
	}
	return false
}

// yodaIsConstant is isConstant.
func yodaIsConstant(s *tokens.Stream, index, end int) bool {
	expectArrayOnly, expectNumberOnly, expectNothing := false, false, false
	for ; index <= end; index++ {
		t := s.At(index)
		if t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			continue
		}
		if expectNothing {
			return false
		}
		if expectArrayOnly {
			if t.Kind == token.Punct && (t.Value == "(" || t.Value == ")" || t.Value == "]") {
				continue
			}
			return false
		}
		if kwIs(t, "array") || (t.Kind == token.Punct && t.Value == "[" && !isOffsetOpen(s, index)) {
			expectArrayOnly = true
			continue
		}
		if expectNumberOnly && t.Kind != token.Number {
			return false
		}
		if t.Kind == token.Punct && t.Value == "-" {
			expectNumberOnly = true
			continue
		}
		if t.Kind == token.Number || yodaIsPlainString(t) ||
			(t.Kind == token.Ident && (t.Value == "true" || t.Value == "false" || t.Value == "null")) {
			expectNothing = true
			continue
		}
		return false
	}
	return true
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

// isCastParen reports whether the "(" at i opens a cast like "(int)", "(string)"
// rather than a grouping parenthesis.
func isCastParen(s *tokens.Stream, i int) bool {
	n := nextSignificantIndex(s, i)
	if n < 0 {
		return false
	}
	t := s.At(n)
	if t.Kind != token.Ident && t.Kind != token.Keyword {
		return false
	}
	switch strings.ToLower(t.Value) {
	case "int", "integer", "bool", "boolean", "float", "double", "real",
		"string", "array", "object", "unset", "binary":
	default:
		return false
	}
	c := nextSignificantIndex(s, n)
	return c >= 0 && s.At(c).Kind == token.Punct && s.At(c).Value == ")"
}
