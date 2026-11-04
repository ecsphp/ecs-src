package rules

import (
	"slices"
	"strings"

	"blink/internal/lexer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/StringLengthToEmptyFixer.php
//
// StringLengthToEmpty replaces a comparison of a global strlen() call with 0
// or 1 by a comparison with the empty string:
//
//	0 === strlen($a)  ->  '' === $a
//	strlen($a) > 0    ->  $a !== ''
type StringLengthToEmpty struct{}

func (StringLengthToEmpty) Name() string {
	return `PhpCsFixer\Fixer\StringNotation\StringLengthToEmptyFixer`
}

func (StringLengthToEmpty) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/StringNotation/StringLengthToEmptyFixer.php"
}

func (StringLengthToEmpty) Fix(s *tokens.Stream) bool {
	calls := strlenEmptyFindCalls(s)
	changed := false
	// last call first, as PHP-CS-Fixer does; cleared tokens keep indexes stable
	for _, call := range slices.Backward(calls) {
		fn, open, closeIdx := call[0], call[1], call[2]
		if !strlenEmptySingleArgument(s, open, closeIdx) {
			continue
		}
		next := nextMeaningfulIndex(s, closeIdx)
		prev := prevMeaningfulIndex(s, fn)
		nsSep := -1
		if isPunctVal(s, prev, `\`) {
			nsSep = prev
			prev = prevMeaningfulIndex(s, prev)
		}
		yoda := strlenEmptyIsOperator(s, prev) // "0 === strlen($a)"
		operator, operand, beyond := next, nextMeaningfulIndex(s, next), prev
		if yoda {
			operator, operand, beyond = prev, prevMeaningfulIndex(s, prev), next
		} else if !strlenEmptyIsOperator(s, next) {
			continue
		}
		if !strlenEmptyIsOperand(s, operand) {
			continue
		}
		op := s.At(operator).Value
		if !yoda {
			op = strlenEmptyFlipped[op]
		}
		identical, ok := strlenEmptyReplacement(op, s.At(operand).Value)
		if !ok {
			continue
		}
		beyondOperand := nextMeaningfulIndex(s, operand)
		if yoda {
			beyondOperand = prevMeaningfulIndex(s, operand)
		}
		if strlenEmptyHigherPrecedence(s, beyond) || strlenEmptyHigherPrecedence(s, beyondOperand) {
			continue
		}

		keepParentheses := strlenEmptyKeepParentheses(s, open, closeIdx)
		s.Set(operand, token.Token{Kind: token.String, Value: "''"})
		if identical {
			s.Set(operator, token.Token{Kind: token.Punct, Value: "==="})
		} else {
			s.Set(operator, token.Token{Kind: token.Punct, Value: "!=="})
		}
		if !keepParentheses {
			clearTokenAndMergeWS(s, closeIdx)
			clearTokenAndMergeWS(s, open)
		}
		clearTokenAndMergeWS(s, fn)
		if nsSep >= 0 {
			clearTokenAndMergeWS(s, nsSep)
		}
		changed = true
	}
	if changed {
		// drop the cleared placeholders
		s.ReplaceRange(0, s.Len()-1, lexer.Lex(s.Render()))
	}
	return changed
}

// strlenEmptyFlipped maps the operator of "strlen(...) op n" to that of "n op strlen(...)".
var strlenEmptyFlipped = map[string]string{"===": "===", "!==": "!==", "<": ">", ">": "<", "<=": ">=", ">=": "<="}

// strlenEmptyReplacement maps "<operand> <operator> strlen(...)" to whether
// the result compares identical to the empty string or not, or ok false when
// the comparison has no such equivalent.
func strlenEmptyReplacement(operator, operand string) (identical, ok bool) {
	if operand == "0" {
		switch operator {
		case "===", ">=":
			return true, true
		case "!==", "<":
			return false, true
		}
		return false, false
	}
	switch operator {
	case "<=":
		return false, true
	case ">":
		return true, true
	}
	return false, false
}

func strlenEmptyIsOperand(s *tokens.Stream, i int) bool {
	return i >= 0 && s.At(i).Kind == token.Number && (s.At(i).Value == "0" || s.At(i).Value == "1")
}

func strlenEmptyIsOperator(s *tokens.Stream, i int) bool {
	if i < 0 || s.At(i).Kind != token.Punct {
		return false
	}
	switch s.At(i).Value {
	case "===", "!==", "<=", ">=", "<", ">":
		return true
	}
	return false
}

// strlenEmptyHigherPrecedence reports an operator that binds tighter than the
// comparison (instanceof, **, <<, >>, !, %, *, +, -, ., /, ~, ?).
func strlenEmptyHigherPrecedence(s *tokens.Stream, i int) bool {
	if i < 0 {
		return false
	}
	t := s.At(i)
	if kwIs(t, "instanceof") {
		return true
	}
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case "**", "<<", ">>", "!", "%", "*", "+", "-", ".", "/", "~", "?":
		return true
	}
	return false
}

// strlenEmptySingleArgument reports whether the call parentheses hold exactly
// one argument (ArgumentsAnalyzer::getArguments, a trailing comma allowed).
func strlenEmptySingleArgument(s *tokens.Stream, open, closeIdx int) bool {
	if isPunctVal(s, nextMeaningfulIndex(s, open), ")") {
		return false
	}
	for i := open + 1; i < closeIdx; i++ {
		if end := strlenEmptyBlockEnd(s, i); end > i {
			i = end
			continue
		}
		if isPunctVal(s, i, ",") {
			return nextMeaningfulIndex(s, i) == closeIdx
		}
	}
	return true
}

// strlenEmptyKeepParentheses reports whether the argument needs its
// parentheses once strlen is gone: anything but variables, names, object
// operators, whitespace, comments and nested blocks (a leading cast aside).
func strlenEmptyKeepParentheses(s *tokens.Stream, open, closeIdx int) bool {
	// a leading "(type)" cast is a block here, skipped like the cast token
	for i := nextMeaningfulIndex(s, open); i >= 0 && i < closeIdx; i++ {
		t := s.At(i)
		switch t.Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		case token.Variable:
			if t.Value != "$" { // a lone "$" (variable variable) is no T_VARIABLE
				continue
			}
		case token.Ident:
			if !strlenEmptyMagicConstants[strings.ToLower(t.Value)] { // T_DIR & co. are no T_STRING
				continue
			}
		case token.Punct:
			if t.Value == "->" || t.Value == "?->" {
				continue
			}
		}
		if end := strlenEmptyBlockEnd(s, i); end > i {
			i = end
			continue
		}
		return true
	}
	return false
}

// strlenEmptyMagicConstants are the magic constants PHP lexes as their own
// token kinds where the lexer keeps an Ident.
var strlenEmptyMagicConstants = map[string]bool{
	"__line__": true, "__file__": true, "__dir__": true, "__function__": true,
	"__class__": true, "__trait__": true, "__method__": true, "__namespace__": true,
	"__property__": true,
}

// strlenEmptyBlockEnd returns the end of the (), [], {} or attribute block
// opened at i, or -1.
func strlenEmptyBlockEnd(s *tokens.Stream, i int) int {
	t := s.At(i)
	if t.Kind == token.AttributeStart {
		return s.AttributeEnd(i)
	}
	if t.Kind == token.Punct && (t.Value == "(" || t.Value == "[" || t.Value == "{") {
		return s.MatchForward(i)
	}
	return -1
}

// strlenEmptyFindCalls returns every global strlen() call as its name, "(" and
// ")" indexes (AbstractFunctionReferenceFixer::find in a loop).
func strlenEmptyFindCalls(s *tokens.Stream) [][3]int {
	var calls [][3]int
	var scope *strlenEmptyScope
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Ident || !strings.EqualFold(s.At(i).Value, "strlen") {
			continue
		}
		open := nextMeaningfulIndex(s, i)
		if !isPunctVal(s, open, "(") {
			continue
		}
		if scope == nil {
			scope = newStrlenEmptyScope(s)
		}
		if scope.isGlobalFunctionCall(s, i, open) {
			if closeIdx := s.MatchForward(open); closeIdx > open {
				calls = append(calls, [3]int{i, open, closeIdx})
			}
		}
		i = open
	}
	return calls
}

// strlenEmptyScope holds the parts of FunctionsAnalyzer's analysis that decide
// whether a "strlen(" calls the global function: the namespaces, and the
// function declarations and "use function" imports named strlen.
type strlenEmptyScope struct {
	namespaces   []strlenEmptyNamespace
	declarations []int
	imports      []strlenEmptyImport
	hasAttribute bool
}

type strlenEmptyNamespace struct {
	start, end int
	global     bool
}

type strlenEmptyImport struct {
	start, end      int
	short, fullName string
}

// isGlobalFunctionCall ports FunctionsAnalyzer::isGlobalFunctionCall for the
// strlen name at i, whose "(" is at open.
func (sc *strlenEmptyScope) isGlobalFunctionCall(s *tokens.Stream, i, open int) bool {
	prev := prevMeaningfulIndex(s, i)
	nsSep := false
	if isPunctVal(s, prev, `\`) {
		nsSep = true
		prev = prevMeaningfulIndex(s, prev)
	}
	if strlenEmptyNotACallBefore(s, prev) {
		return false
	}
	// first-class callable "strlen(...)"
	if n := nextMeaningfulIndex(s, open); isPunctVal(s, n, "...") && isPunctVal(s, nextMeaningfulIndex(s, n), ")") {
		return false
	}
	if nsSep {
		return true
	}

	scopeStart, scopeEnd := 0, s.Len()-1
	inGlobal := false
	for _, ns := range sc.namespaces {
		scopeStart, scopeEnd = ns.start, ns.end
		if i >= scopeStart && i <= scopeEnd {
			inGlobal = ns.global
			break
		}
	}
	if !inGlobal {
		for _, d := range sc.declarations {
			if d >= scopeStart && d <= scopeEnd {
				return false
			}
		}
	}
	for _, im := range sc.imports {
		if im.start < scopeStart || im.end > scopeEnd {
			continue
		}
		return im.short == strings.TrimLeft(im.fullName, `\`)
	}
	return !sc.hasAttribute || !strlenEmptyInAttribute(s, i)
}

// strlenEmptyNotACallBefore mirrors FunctionsAnalyzer::POSSIBLE_KINDS: a token
// before the name that makes it a method, static, namespaced or declared name.
func strlenEmptyNotACallBefore(s *tokens.Stream, i int) bool {
	if i < 0 {
		return false
	}
	t := s.At(i)
	switch t.Kind {
	case token.Ident, token.AttributeStart:
		return true
	case token.Keyword:
		switch strings.ToLower(t.Value) {
		case "function", "new":
			return true
		case "namespace":
			return isPunctVal(s, nextMeaningfulIndex(s, i), `\`)
		}
	case token.Punct:
		switch t.Value {
		case "::", "->", "?->":
			return true
		case "&":
			return kwIs(s.At(max(prevMeaningfulIndex(s, i), 0)), "function")
		}
	}
	return false
}

// strlenEmptyInAttribute ports AttributeAnalyzer::isAttribute: the name at i is
// an attribute class right inside "#[...]".
func strlenEmptyInAttribute(s *tokens.Stream, i int) bool {
	start := -1
	for j := i - 1; j >= 0 && start < 0; j-- {
		t := s.At(j)
		switch t.Kind {
		case token.AttributeStart, token.AttributeEnd, token.OpenTag:
			start = j
		case token.Variable:
			if t.Value != "$" {
				start = j
			}
		case token.Punct:
			if t.Value == ";" || t.Value == "{" {
				start = j
			}
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "private", "protected", "public", "return":
				start = j
			case "function":
				if !kwIs(s.At(max(prevMeaningfulIndex(s, j), 0)), "use") {
					start = j
				}
			}
		}
	}
	if start < 0 || s.At(start).Kind != token.AttributeStart {
		return false
	}
	depth := 0
	for j := start + 1; j < i; j++ {
		if isPunctVal(s, j, "(") {
			depth++
		} else if isPunctVal(s, j, ")") {
			depth--
		}
	}
	return depth == 0
}

func newStrlenEmptyScope(s *tokens.Stream) *strlenEmptyScope {
	sc := &strlenEmptyScope{}
	n := s.Len()
	hasOpenTag := false
	for i := range n {
		t := s.At(i)
		switch {
		case t.Kind == token.AttributeStart:
			sc.hasAttribute = true
		case t.Kind == token.OpenTag && t.Value != "<?=":
			hasOpenTag = true
		}
	}

	// NamespacesAnalyzer::getDeclarations
	isDeclaration := func(i int) bool {
		return kwIs(s.At(i), "namespace") && !isPunctVal(s, nextMeaningfulIndex(s, i), `\`)
	}
	for i := 1; i < n; i++ {
		if !isDeclaration(i) {
			continue
		}
		declEnd := -1
		for j := i + 1; j < n; j++ {
			if isPunctVal(s, j, ";") || isPunctVal(s, j, "{") {
				declEnd = j
				break
			}
		}
		if declEnd < 0 {
			break
		}
		var name strings.Builder
		for j := i + 1; j < declEnd; j++ {
			name.WriteString(s.At(j).Value)
		}
		end := n
		if isPunctVal(s, declEnd, "{") {
			end = s.MatchForward(declEnd)
			if end < 0 {
				end = n - 1
			}
		} else {
			for j := declEnd + 1; j < n; j++ {
				if isDeclaration(j) {
					end = j
					break
				}
			}
			end--
		}
		sc.namespaces = append(sc.namespaces, strlenEmptyNamespace{start: i, end: end, global: strings.TrimSpace(name.String()) == ""})
		i = end
	}
	if len(sc.namespaces) == 0 && hasOpenTag {
		start := 0
		if s.At(0).Kind == token.InlineHTML {
			start = 1
		}
		sc.namespaces = append(sc.namespaces, strlenEmptyNamespace{start: start, end: n - 1, global: true})
	}

	// function declarations named strlen, outside class-like bodies
	for i := 0; i < n; i++ {
		if isClassyDeclaration(s, i) {
			if end := strlenEmptyClassyEnd(s, i); end > i {
				i = end
				continue
			}
			break
		}
		if !kwIs(s.At(i), "function") || kwIs(s.At(max(prevMeaningfulIndex(s, i), 0)), "use") {
			continue
		}
		j := nextMeaningfulIndex(s, i)
		if isPunctVal(s, j, "&") {
			j = nextMeaningfulIndex(s, j)
		}
		if j >= 0 && s.At(j).Kind == token.Ident && strings.EqualFold(s.At(j).Value, "strlen") {
			sc.declarations = append(sc.declarations, j)
		}
	}

	// single "use function" imports whose short name is strlen
	classyEnd := -1
	for i := 0; i < n; i++ {
		if i > classyEnd && isClassyDeclaration(s, i) {
			if classyEnd = strlenEmptyClassyEnd(s, i); classyEnd < 0 {
				break
			}
			continue
		}
		if i <= classyEnd || !kwIs(s.At(i), "use") || isPunctVal(s, prevMeaningfulIndex(s, i), ")") {
			continue
		}
		if !kwIs(s.At(max(nextMeaningfulIndex(s, i), 0)), "function") {
			continue
		}
		end := -1
		for j := i + 1; j < n; j++ {
			if isPunctVal(s, j, ";") || s.At(j).Kind == token.CloseTag {
				end = j
				break
			}
		}
		if end < 0 {
			continue
		}
		var full strings.Builder
		short, aliased, multi := "", false, false
		for j := nextMeaningfulIndex(s, nextMeaningfulIndex(s, i)); j >= 0 && j < end; j = nextMeaningfulIndex(s, j) {
			t := s.At(j)
			switch {
			case isPunctVal(s, j, ",") || isPunctVal(s, j, "{"):
				multi = true
			case kwIs(t, "as"):
				aliased = true
			case t.Kind == token.Ident || t.Kind == token.Keyword:
				short = t.Value
				if !aliased {
					full.WriteString(t.Value)
				}
			case isPunctVal(s, j, `\`):
				full.WriteString(t.Value)
			}
		}
		if !multi && strings.EqualFold(short, "strlen") {
			sc.imports = append(sc.imports, strlenEmptyImport{start: i, end: end, short: short, fullName: full.String()})
		}
		i = end
	}
	return sc
}

// isClassyDeclaration reports a class, interface, trait or enum keyword.
func isClassyDeclaration(s *tokens.Stream, i int) bool {
	if kwIs(s.At(i), "enum") {
		j := nextMeaningfulIndex(s, i)
		return j >= 0 && s.At(j).Kind == token.Ident
	}
	return isClassyKeyword(s.At(i))
}

// strlenEmptyClassyEnd returns the "}" closing the body of the class-like
// declared at i (past an anonymous class's arguments), or -1.
func strlenEmptyClassyEnd(s *tokens.Stream, i int) int {
	j := i + 1
	for j < s.Len() && !isPunctVal(s, j, "(") && !isPunctVal(s, j, "{") {
		j++
	}
	if isPunctVal(s, j, "(") {
		if j = s.MatchForward(j); j < 0 {
			return -1
		}
		for j < s.Len() && !isPunctVal(s, j, "{") {
			j++
		}
	}
	if j >= s.Len() {
		return -1
	}
	return s.MatchForward(j)
}
