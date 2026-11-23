package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededControlParenthesesFixer.php
//
// NoUnneededControlParentheses removes redundant parentheses in one pass over
// every "(", like PHP-CS-Fixer's applyFix, each kind only when its statement is
// configured: "return ($x);", "foo(($x))", "($a) + 1", "(Foo::bar())->baz()".
// Without configuration it uses PHP-CS-Fixer's default, without others,
// negative_instanceof and yield_from.
type NoUnneededControlParentheses struct {
	// Statements limits the fixed statements; nil means the default set.
	Statements []string
}

func (f NoUnneededControlParentheses) WithConfig(config map[string]any) fixer.Fixer {
	switch list := config["statements"].(type) {
	case []string:
		f.Statements = append([]string{}, list...)
	case []any:
		f.Statements = []string{}
		for _, v := range list {
			if name, ok := v.(string); ok {
				f.Statements = append(f.Statements, name)
			}
		}
	}
	return f
}

var nucpDefaultStatements = []string{"break", "clone", "continue", "echo_print", "return", "switch_case", "yield"}

func (f NoUnneededControlParentheses) enabled(key string) bool {
	if f.Statements == nil {
		return slices.Contains(nucpDefaultStatements, key)
	}
	return slices.Contains(f.Statements, key)
}

func (NoUnneededControlParentheses) Name() string {
	return `PhpCsFixer\Fixer\ControlStructure\NoUnneededControlParenthesesFixer`
}

func (NoUnneededControlParentheses) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/ControlStructure/NoUnneededControlParenthesesFixer.php"
}

func (f NoUnneededControlParentheses) Fix(s *tokens.Stream) bool {
	changed := false
	for open := 0; open < s.Len(); open++ {
		if !csbIsPunct(s.At(open), "(") || nucpCastEnd(s, open) >= 0 {
			continue
		}
		if f.fixPair(s, open) {
			changed = true
			// the "(" is gone or now a space: look at what took its place
			open--
		}
	}
	return changed
}

// fixPair applies applyFix's checks, in its order, to the pair opening at open.
func (f NoUnneededControlParentheses) fixPair(s *tokens.Stream, open int) bool {
	closing := s.MatchForward(open)
	if closing < 0 {
		return false
	}
	before, after := nucpPrev(s, open), nucpNext(s, closing)
	if before < 0 || after < 0 {
		return false
	}
	if nucpNext(s, open) == closing {
		return nucpIsExit(s, before) && f.removePair(s, before, open, closing, after, "others")
	}
	if nucpKnownNegative(s, before) {
		return false
	}
	if nucpUselessWrapped(s, before, after) {
		return f.removePair(s, before, open, closing, after, nucpConfigType(s.At(before)))
	}
	if csbIsKeyword(s.At(before), "clone") {
		return nucpWrappedCloneArgument(s, before, open, closing, after) && f.removePair(s, before, open, closing, after, "clone")
	}
	if instanceOf := nucpInstanceOfIndex(s, open, closing); instanceOf >= 0 {
		if !nucpWrappedInstanceOf(s, instanceOf, before, open, closing, after) {
			return false
		}
		key := "others"
		if csbIsPunct(s.At(before), "!") {
			key = "negative_instanceof"
		}
		return f.removePair(s, before, open, closing, after, key)
	}
	return nucpWrappedPartOfOperation(s, before, open, closing, after) && f.removePair(s, before, open, closing, after, nucpConfigType(s.At(before)))
}

// removePair is removeUselessParenthesisPair: a parenthesis becomes a space where
// its removal would glue two tokens together.
func (f NoUnneededControlParentheses) removePair(s *tokens.Stream, before, open, closing, after int, key string) bool {
	if !f.enabled(key) {
		return false
	}
	b, a := s.At(before), s.At(after)
	spaceAfter := !nucpIsAccess(a) && !csbIsPunct(a, ",") && !endsStatement(a) && !nucpBlockEnd(s, after) &&
		(!csbIsPunct(a, ":") || !csbIsKeyword(b, "case"))
	spaceBefore := !nucpIsPreUnary(s, before) && !nucpIsPlainBraceClose(s, before) && !nucpIsExit(s, before) &&
		(b.Kind != token.OpenTag || b.Value == "<?=") && !nucpBlockStart(s, before)
	removeParenthesis(s, closing, spaceAfter)
	removeParenthesis(s, open, spaceBefore)
	return true
}

func nucpConfigType(t token.Token) string {
	if t.Kind != token.Keyword {
		return "others"
	}
	switch word := strings.ToLower(t.Value); word {
	case "break", "continue", "return", "yield":
		return word
	case "case":
		return "switch_case"
	case "echo", "print":
		return "echo_print"
	default:
		// yield from is one token that keeps its inner spacing and comments
		if strings.HasPrefix(word, "yield") {
			return "yield_from"
		}
	}
	return "others"
}

// nucpPrev returns the index of the meaningful token before i, or -1. PHP lexes a
// cast as one token, which its "(" stands for here.
func nucpPrev(s *tokens.Stream, i int) int {
	p := prevMeaningfulIndex(s, i)
	if p >= 0 && csbIsPunct(s.At(p), ")") && isCastCloseParen(s, p) {
		return s.MatchBackward(p)
	}
	return p
}

// nucpNext returns the index of the meaningful token after i, or -1, past the
// whole cast when i is one.
func nucpNext(s *tokens.Stream, i int) int {
	if end := nucpCastEnd(s, i); end >= 0 {
		i = end
	}
	return nextMeaningfulIndex(s, i)
}

// nucpCastEnd returns the ")" of the cast whose "(" is at open, or -1: what PHP's
// scanner reads as one cast token, with only spaces or tabs inside.
func nucpCastEnd(s *tokens.Stream, open int) int {
	if !csbIsPunct(s.At(open), "(") {
		return -1
	}
	j := open + 1
	if j < s.Len() && isCastPadding(s.At(j)) {
		j++
	}
	if j >= s.Len() || !isCastType(s.At(j)) {
		return -1
	}
	j++
	if j < s.Len() && isCastPadding(s.At(j)) {
		j++
	}
	if j >= s.Len() || !csbIsPunct(s.At(j), ")") {
		return -1
	}
	return j
}

// nucpKnownNegative mirrors KNOWN_NEGATIVE_PRE_TYPES: a "(" after these opens a
// call, a declaration or a construct's own parentheses.
func nucpKnownNegative(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.Ident:
		// T_STRING, and T_EXIT, T_EVAL, T_HALT_COMPILER, CT::T_CLASS_CONSTANT
		return true
	case token.Variable:
		return t.Value != "$"
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "array", "catch", "class", "declare", "elseif", "empty", "fn", "for", "foreach", "function", "if",
			"include", "include_once", "isset", "list", "match", "require", "require_once", "static", "switch",
			"unset", "use", "while":
			return true
		}
	case token.Punct:
		switch t.Value {
		case "&":
			// CT::T_RETURN_REF
			p := prevMeaningfulIndex(s, i)
			return p >= 0 && (csbIsKeyword(s.At(p), "function") || csbIsKeyword(s.At(p), "fn"))
		case "}":
			// CT::T_DYNAMIC_VAR_BRACE_CLOSE
			open := s.MatchBackward(i)
			return open >= 0 && nucpBraceKind(s, open) == nucpDynamicVarBrace
		}
	}
	return false
}

func nucpUselessWrapped(s *tokens.Stream, before, after int) bool {
	return nucpSingleStatement(s, before, after) || nucpWrappedFnBody(s, before, after) ||
		nucpWrappedForElement(s, before, after) || nucpWrappedConstructArgument(s, before, after) ||
		nucpWrappedSequenceElement(s, before, after)
}

func nucpWrappedCloneArgument(s *tokens.Stream, clone, open, closing, after int) bool {
	before := nucpPrev(s, clone)
	if before < 0 {
		return false
	}
	if csbIsPunct(s.At(before), "?") {
		// Only before ":" or around a plain variable. Anything else can then apply to
		// another expression, where PHP-CS-Fixer unwraps anyway: "clone ($o->b)()" would
		// call a method, and "clone (new Foo)->bar" and "clone (enum) and $b" become
		// parse errors.
		if !csbIsPunct(s.At(after), ":") && !cloneOperandIsVariable(s, open, closing) {
			return false
		}
	} else if !nucpSimpleAssignment(s, before, after) && !nucpSingleStatement(s, before, after) &&
		!nucpWrappedFnBody(s, before, after) && !nucpWrappedForElement(s, before, after) &&
		!nucpWrappedSequenceElement(s, before, after) {
		return false
	}
	// a leading new belongs to the operand
	if first := nucpNext(s, open); first >= 0 && csbIsKeyword(s.At(first), "new") {
		return !nucpContainsOperation(s, first, closing)
	}
	return !nucpContainsOperation(s, open, closing)
}

// cloneOperandIsVariable reports "($a)" or "(($a))" for the pair at open..closing.
func cloneOperandIsVariable(s *tokens.Stream, open, closing int) bool {
	i := nextMeaningfulIndex(s, open)
	for csbIsPunct(s.At(i), "(") {
		i = nextMeaningfulIndex(s, i)
	}
	if s.At(i).Kind != token.Variable {
		return false
	}
	for i = nextMeaningfulIndex(s, i); i != closing; i = nextMeaningfulIndex(s, i) {
		if !csbIsPunct(s.At(i), ")") {
			return false
		}
	}
	return true
}

// nucpInstanceOfIndex returns the index of the only instanceof between open and
// closing, nested ones included, or -1 when there are none or several.
func nucpInstanceOfIndex(s *tokens.Stream, open, closing int) int {
	found := -1
	for j := open; j < closing; j++ {
		if csbIsKeyword(s.At(j), "instanceof") {
			if found >= 0 {
				return -1
			}
			found = j
		}
	}
	return found
}

func nucpWrappedInstanceOf(s *tokens.Stream, instanceOf, before, open, closing, after int) bool {
	if nucpContainsOperation(s, open, instanceOf) || nucpContainsOperation(s, instanceOf, closing) {
		return false
	}
	if csbIsPunct(s.At(before), "!") {
		if before = nucpPrev(s, before); before < 0 {
			return false
		}
	}
	return nucpSimpleAssignment(s, before, after) || nucpSingleStatement(s, before, after) ||
		nucpWrappedFnBody(s, before, after) || nucpWrappedForElement(s, before, after) ||
		nucpWrappedSequenceElement(s, before, after)
}

// nucpWrappedPartOfOperation is isWrappedPartOfOperation: an operand without
// operation, past its unary operators and access chain, between two operators
// or an operator and a block, comma or statement edge.
func nucpWrappedPartOfOperation(s *tokens.Stream, before, open, closing, after int) bool {
	if nucpContainsOperation(s, open, closing) {
		return false
	}
	moved := false
	if nucpIsPreUnary(s, before) {
		for {
			if before = nucpPrev(s, before); before < 0 || !nucpIsPreUnary(s, before) {
				break
			}
		}
		moved = true
	}
	if nucpIsAccess(s.At(after)) {
		after = nucpAfterAccess(s, after)
		moved = true
		if after >= 0 && unaryOperatorSpacesIsSuccessor(s, after) {
			after = nucpNext(s, after)
		}
	}
	if before < 0 || after < 0 {
		return false
	}
	if moved {
		if nucpKnownNegative(s, before) {
			return false
		}
		if nucpUselessWrapped(s, before, after) {
			return true
		}
	}
	beforeIsBinary, afterIsBinary := bosIsBinaryOperator(s, before), bosIsBinaryOperator(s, after)
	if beforeIsBinary && afterIsBinary {
		return true
	}
	beforeIsBlockOpenOrComma := csbIsPunct(s.At(before), ",") || nucpBlockStart(s, before)
	afterIsBlockEndOrComma := csbIsPunct(s.At(after), ",") || nucpBlockEnd(s, after)
	if beforeIsBlockOpenOrComma && afterIsBinary || beforeIsBinary && afterIsBlockEndOrComma {
		return true
	}
	beforeIsStatementOpen := nucpStatementOpen(s, before) || csbIsKeyword(s.At(before), "case")
	return beforeIsStatementOpen && afterIsBinary || beforeIsBinary && endsStatement(s.At(after))
}

// nucpWrappedConstructArgument matches print, yield, yield from, require and
// include (X) bounded like a sequence element.
func nucpWrappedConstructArgument(s *tokens.Stream, before, after int) bool {
	t := s.At(before)
	if t.Kind != token.Keyword {
		return false
	}
	switch word := strings.ToLower(t.Value); word {
	case "print", "require", "require_once", "include", "include_once":
	default:
		if !strings.HasPrefix(word, "yield") {
			return false
		}
	}
	before = nucpPrev(s, before)
	return before >= 0 && nucpWrappedSequenceElement(s, before, after)
}

// nucpSingleStatement matches a statement start or control keyword before the
// pair and ";" or "?>" after it, or a case and its ":" or ";".
func nucpSingleStatement(s *tokens.Stream, before, after int) bool {
	if csbIsKeyword(s.At(before), "case") {
		return csbIsPunct(s.At(after), ":") || csbIsPunct(s.At(after), ";")
	}
	return endsStatement(s.At(after)) && nucpStatementOpen(s, before)
}

func nucpSimpleAssignment(s *tokens.Stream, before, after int) bool {
	return csbIsPunct(s.At(before), "=") && endsStatement(s.At(after))
}

func endsStatement(t token.Token) bool {
	return csbIsPunct(t, ";") || t.Kind == token.CloseTag
}

// nucpStatementOpen is BEFORE_TYPES, with a "}" that does not close a dynamic
// element.
func nucpStatementOpen(s *tokens.Stream, i int) bool {
	t := s.At(i)
	switch t.Kind {
	case token.OpenTag:
		return true
	case token.Punct:
		switch t.Value {
		case ";":
			return true
		case "{":
			return nucpBraceKind(s, i) == nucpPlainBrace
		case "}":
			return nucpIsPlainBraceClose(s, i) && !closesDynamicElement(s, i)
		}
		return false
	case token.Keyword:
		word := strings.ToLower(t.Value)
		return strings.HasPrefix(word, "yield") || slices.Contains([]string{
			"echo", "print", "return", "throw", "break", "continue", "require", "require_once", "include", "include_once",
		}, word)
	}
	return false
}

// closesDynamicElement follows PHP-CS-Fixer's closeCurlyBelongsToDynamicElement: a
// "}" whose "{" follows "::", or a ":" with no case closer to it than a "?".
func closesDynamicElement(s *tokens.Stream, closing int) bool {
	open := s.MatchBackward(closing)
	if open < 0 {
		return false
	}
	prev := prevMeaningfulIndex(s, open)
	if prev < 0 {
		return false
	}
	if csbIsPunct(s.At(prev), "::") {
		return true
	}
	if !csbIsPunct(s.At(prev), ":") {
		return false
	}
	for j := prev - 1; j >= 0; j-- {
		if csbIsPunct(s.At(j), "?") {
			return true
		}
		if csbIsKeyword(s.At(j), "case") {
			return false
		}
	}
	// PHP-CS-Fixer throws when it finds neither: keep the parentheses
	return true
}

// nucpWrappedFnBody matches the body of an arrow function: "fn () => (X);".
func nucpWrappedFnBody(s *tokens.Stream, before, after int) bool {
	a := s.At(after)
	return isFnBodyStart(s, before) && (endsStatement(a) || csbIsPunct(a, ","))
}

// isFnBodyStart reports whether the token at index i is the "=>" of an arrow
// function, with at most a plain return type: "fn () =>", "fn (): A|B =>".
func isFnBodyStart(s *tokens.Stream, i int) bool {
	if !csbIsPunct(s.At(i), "=>") {
		return false
	}
	j := prevMeaningfulIndex(s, i)
	if j >= 0 && s.At(j).Kind == token.Ident {
		for j >= 0 && (s.At(j).Kind == token.Ident || csbIsPunct(s.At(j), "|") || csbIsPunct(s.At(j), "&")) {
			j = prevMeaningfulIndex(s, j)
		}
		if j < 0 || !csbIsPunct(s.At(j), ":") {
			return false
		}
		j = prevMeaningfulIndex(s, j)
	}
	if j < 0 || !csbIsPunct(s.At(j), ")") {
		return false
	}
	open := s.MatchBackward(j)
	if open < 0 {
		return false
	}
	j = prevMeaningfulIndex(s, open)
	if j >= 0 && csbIsPunct(s.At(j), "&") {
		j = prevMeaningfulIndex(s, j)
	}
	return j >= 0 && csbIsKeyword(s.At(j), "fn")
}

// nucpWrappedForElement matches the first and last expressions of a for head; the
// middle one is a single statement already: for (X; Y; Z).
func nucpWrappedForElement(s *tokens.Stream, before, after int) bool {
	candidate := -1
	switch {
	case csbIsPunct(s.At(before), "(") && nucpCastEnd(s, before) < 0 && csbIsPunct(s.At(after), ";"):
		candidate = nucpPrev(s, before)
	case csbIsPunct(s.At(after), ")") && csbIsPunct(s.At(before), ";"):
		if open := s.MatchBackward(after); open >= 0 {
			candidate = nucpPrev(s, open)
		}
	}
	return candidate >= 0 && csbIsKeyword(s.At(candidate), "for")
}

// nucpWrappedSequenceElement matches any of ",(X),", "[(X)]", "{(X)}" and "((X))".
func nucpWrappedSequenceElement(s *tokens.Stream, start, end int) bool {
	return (csbIsPunct(s.At(start), ",") || nucpBlockStart(s, start)) &&
		(csbIsPunct(s.At(end), ",") || nucpBlockEnd(s, end))
}

// nucpIsPreUnary is isPreUnaryOperation: a unary prefix operator or a cast.
func nucpIsPreUnary(s *tokens.Stream, i int) bool {
	return s.At(i).Kind == token.Punct && unaryOperatorSpacesIsPredecessor(s, i) || nucpCastEnd(s, i) >= 0
}

func nucpIsAccess(t token.Token) bool {
	return csbIsPunct(t, "[") || csbIsPunct(t, "->") || csbIsPunct(t, "?->")
}

// nucpAfterAccess is getAfterAccess: the first token past the access chain
// starting at i, or -1.
func nucpAfterAccess(s *tokens.Stream, i int) int {
	for i >= 0 {
		t := s.At(i)
		switch {
		case nucpBlockStart(s, i):
			if i = s.MatchForward(i); i < 0 {
				return -1
			}
		case csbIsPunct(t, "->"), csbIsPunct(t, "?->"), csbIsPunct(t, "::"), t.Kind == token.Variable:
		case t.Kind == token.Ident && taIsTString(s, i) && !nucpIsClassConstant(s, i):
		default:
			return i
		}
		i = nucpNext(s, i)
	}
	return i
}

// nucpIsClassConstant reports the "class" of "Foo::class" (CT::T_CLASS_CONSTANT).
func nucpIsClassConstant(s *tokens.Stream, i int) bool {
	p := prevMeaningfulIndex(s, i)
	return strings.EqualFold(s.At(i).Value, "class") && p >= 0 && csbIsPunct(s.At(p), "::")
}

func nucpIsExit(s *tokens.Stream, i int) bool {
	return taIsConstruct(s, i, "exit") || taIsConstruct(s, i, "die")
}

// The kinds of "{" PHP-CS-Fixer's BraceTransformer tells apart that matter here.
const (
	nucpPlainBrace = iota
	// "->{" and "${": blocks, but no plain "{"
	nucpDynamicPropBrace
	nucpDynamicVarBrace
	// a group import, or a class constant fetch ended by ";", "?>" or "::": no blocks
	nucpOtherBrace
)

func nucpBraceKind(s *tokens.Stream, open int) int {
	if open > 0 && (csbIsPunct(s.At(open-1), "->") || csbIsPunct(s.At(open-1), "?->")) {
		return nucpDynamicPropBrace
	}
	p := prevMeaningfulIndex(s, open)
	if p < 0 {
		return nucpPlainBrace
	}
	switch t := s.At(p); {
	case t.Kind == token.Variable && t.Value == "$":
		return nucpDynamicVarBrace
	case csbIsPunct(t, `\`):
		return nucpOtherBrace
	case csbIsPunct(t, "::"):
		if closing := s.MatchForward(open); closing >= 0 {
			if n := nextMeaningfulIndex(s, closing); n >= 0 && (endsStatement(s.At(n)) || csbIsPunct(s.At(n), "::")) {
				return nucpOtherBrace
			}
		}
	}
	return nucpPlainBrace
}

func nucpIsPlainBraceClose(s *tokens.Stream, i int) bool {
	if !csbIsPunct(s.At(i), "}") {
		return false
	}
	open := s.MatchBackward(i)
	return open >= 0 && nucpBraceKind(s, open) == nucpPlainBrace
}

// nucpBlockStart reports an opening edge of BLOCK_TYPES, not a cast,
// "(new Foo)" parentheses, a group import nor a class constant fetch.
func nucpBlockStart(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case "[":
		return true
	case "(":
		return nucpCastEnd(s, i) < 0 && !nucpClassInstantiation(s, i)
	case "{":
		return nucpBraceKind(s, i) != nucpOtherBrace
	}
	return false
}

func nucpBlockEnd(s *tokens.Stream, i int) bool {
	t := s.At(i)
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case "]":
		return true
	case ")", "}":
		open := s.MatchBackward(i)
		return open >= 0 && nucpBlockStart(s, open)
	}
	return false
}

// nucpClassInstantiation reports the "(" of "(new Foo)" as PHP-CS-Fixer's
// BraceClassInstantiationTransformer sees it.
func nucpClassInstantiation(s *tokens.Stream, open int) bool {
	if n := nextMeaningfulIndex(s, open); n < 0 || !csbIsKeyword(s.At(n), "new") {
		return false
	}
	closing := s.MatchForward(open)
	return closing >= 0 && isClassInstantiationCloseParen(s, closing)
}

// nucpContainsOperation is containsOperation: whether a token between start and
// end, a nested block counting as one operand, is no plain operand part. Like
// PHP-CS-Fixer it walks past end when end sits in a nested block.
func nucpContainsOperation(s *tokens.Stream, start, end int) bool {
	for {
		start = nucpNext(s, start)
		switch {
		case start < 0:
			return true
		case start == end:
			return false
		case nucpCastEnd(s, start) >= 0:
			return true
		case csbIsPunct(s.At(start), "("), csbIsPunct(s.At(start), "["), csbIsPunct(s.At(start), "{"):
			if start = s.MatchForward(start); start < 0 {
				return true
			}
		case s.At(start).Kind == token.Ident:
			if nucpIsClassConstant(s, start) || !nucpNoopName(s, start) {
				return true
			}
		case !nucpNoopToken(s.At(start)):
			return true
		}
	}
}

// nucpNoopName rejects the names that are language constructs to PHP but plain
// names to the lexer, unless they are a member or a segment of a qualified name.
func nucpNoopName(s *tokens.Stream, i int) bool {
	word := strings.ToLower(s.At(i).Value)
	if prev := prevMeaningfulIndex(s, i); prev >= 0 && s.At(prev).Kind == token.Punct {
		switch s.At(prev).Value {
		case "->", "?->", "::", `\`:
			return true
		}
	}
	if next := nextMeaningfulIndex(s, i); next >= 0 && csbIsPunct(s.At(next), `\`) {
		return word != "namespace"
	}
	switch word {
	case "eval", "exit", "die", "__property__":
		return false
	}
	return true
}

// nucpNoopToken is noopTypes for the tokens that are no names.
func nucpNoopToken(t token.Token) bool {
	switch t.Kind {
	case token.Variable, token.Number:
		return true
	case token.String:
		return t.IsLiteral() && !t.StartsHeredoc() && !t.IsShellExec()
	case token.Punct:
		return t.Value == "::" || t.Value == `\` || t.Value == "->" || t.Value == "?->"
	case token.Keyword:
		return strings.EqualFold(t.Value, "static")
	}
	return false
}

func removeParenthesis(s *tokens.Stream, i int, needsSpace bool) {
	spaced := func(j int) bool {
		if j < 0 || j >= s.Len() {
			return false
		}
		k := s.At(j).Kind
		return k == token.Whitespace || k == token.Comment || k == token.DocComment
	}
	if needsSpace && !spaced(i-1) && !spaced(i+1) {
		s.Set(i, token.Token{Kind: token.Whitespace, Value: " "})
		return
	}
	s.RemoveAt(i)
	if i > 0 && i < s.Len() && s.Kind(i-1) == token.Whitespace && s.Kind(i) == token.Whitespace {
		s.SetValue(i-1, s.At(i-1).Value+s.At(i).Value)
		s.RemoveAt(i)
	}
}
