package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// symScan ports the PHP-CS-Fixer token analyzers these fixers use. PHP-CS-Fixer's
// transformers retag tokens by context (T_TYPE_COLON, T_USE_TRAIT, ...); blink's
// tokens are flatter, so the helpers below recompute those tags. It is built once
// per Fix call and is stale once the stream changes.
type symScan struct {
	s     *tokens.Stream
	n     int
	match []int  // index of the matching bracket for ( ) [ ] { } #[ ], else -1
	tstr  []bool // token is a T_STRING in PHP-CS-Fixer's token model
	// inClassBody marks tokens inside the braces of a class, interface, trait or
	// enum (at any depth), where a "use" is a trait import.
	inClassBody []bool
	importKW    []bool // "function"/"const" of an import (CT::T_FUNCTION_IMPORT / T_CONST_IMPORT)
	// namedArgKW marks a reserved word used as a named argument (foo(class: 1)):
	// PHP-CS-Fixer tokenizes with TOKEN_PARSE, which makes it a T_STRING (then a
	// CT::T_NAMED_ARGUMENT_NAME), never the keyword it spells.
	namedArgKW []bool
	hasEnum    bool
	hasAttr    bool

	nsDecls []nsDecl
	uses    []useDecl // import declarations, multi-imports filtered out
}

// nsDecl mirrors PHP-CS-Fixer's NamespaceAnalysis.
type nsDecl struct {
	name       string
	declEnd    int // the ";" or "{" ending the declaration
	scopeStart int // the namespace keyword, or the open tag for the implicit global one
	scopeEnd   int
}

func (d nsDecl) isGlobal() bool { return d.name == "" }

// useDecl mirrors PHP-CS-Fixer's NamespaceUseAnalysis for a non-multi import.
type useDecl struct {
	kind      string // "class", "function" or "const"
	fullName  string // as written, possibly with a leading "\"
	shortName string // alias or last segment
	aliased   bool
	start     int // the "use" token
	end       int // the ";" (or close tag) ending the statement
}

// symScanKey is sharedSymScan's key for Stream.Memo.
type symScanKey struct{}

// sharedSymScan returns the symScan of s, shared by the rules that run while
// the tokens stay the same.
func sharedSymScan(s *tokens.Stream) *symScan {
	return s.Memo(symScanKey{}, func() any { return newSymScan(s) }).(*symScan)
}

func newSymScan(s *tokens.Stream) *symScan {
	n := s.Len()
	sc := &symScan{s: s, n: n, match: make([]int, n), tstr: make([]bool, n), inClassBody: make([]bool, n), importKW: make([]bool, n), namedArgKW: make([]bool, n)}
	var stack []int
	for i := range n {
		sc.match[i] = -1
		t := s.At(i)
		switch t.Kind {
		case token.AttributeStart:
			sc.hasAttr = true
			stack = append(stack, i)
		case token.AttributeEnd:
			if len(stack) > 0 {
				o := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				sc.match[o], sc.match[i] = i, o
			}
		case token.Punct:
			switch t.Value {
			case "(", "[", "{":
				stack = append(stack, i)
			case ")", "]", "}":
				if len(stack) > 0 {
					o := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					sc.match[o], sc.match[i] = i, o
				}
			}
		}
	}
	for i := range n {
		if s.At(i).Kind != token.Keyword {
			continue
		}
		next, prev := sc.nextM(i), sc.prevM(i)
		if sc.isPunct(next, ":") && (sc.isPunct(prev, "(") || sc.isPunct(prev, ",")) {
			sc.namedArgKW[i] = true
		} else if strings.EqualFold(s.At(i).Value, "enum") {
			sc.hasEnum = true
		}
	}
	for i := range n {
		sc.tstr[i] = sc.computeTString(i)
	}
	// class-like bodies
	for i := range n {
		if !sc.isClassy(i) {
			continue
		}
		open := sc.classBodyOpen(i)
		if open < 0 || sc.match[open] < 0 {
			continue
		}
		for j := open + 1; j < sc.match[open]; j++ {
			sc.inClassBody[j] = true
		}
	}
	for i := range n {
		sc.importKW[i] = sc.computeImportKind(i)
	}
	sc.nsDecls = sc.computeNamespaces()
	sc.computeUses()
	return sc
}

func (sc *symScan) tok(i int) token.Token { return sc.s.At(i) }

func (sc *symScan) isPunct(i int, v string) bool {
	if i < 0 || i >= sc.n {
		return false
	}
	t := sc.s.At(i)
	return t.Kind == token.Punct && t.Value == v
}

func (sc *symScan) isKeyword(i int, words ...string) bool {
	if i < 0 || i >= sc.n {
		return false
	}
	t := sc.s.At(i)
	if t.Kind != token.Keyword || sc.namedArgKW[i] {
		return false
	}
	for _, w := range words {
		if strings.EqualFold(t.Value, w) {
			return true
		}
	}
	return false
}

func (sc *symScan) isKind(i int, k token.Kind) bool {
	return i >= 0 && i < sc.n && sc.s.At(i).Kind == k
}

func (sc *symScan) isTString(i int) bool { return i >= 0 && i < sc.n && sc.tstr[i] }

func (sc *symScan) prevM(i int) int { return prevMeaningfulIndex(sc.s, i) }
func (sc *symScan) nextM(i int) int { return nextMeaningfulIndex(sc.s, i) }

// nextPunct returns the first punctuation token equal to v after i, or -1.
func (sc *symScan) nextPunct(i int, v string) int {
	for j := i + 1; j < sc.n; j++ {
		if sc.isPunct(j, v) {
			return j
		}
	}
	return -1
}

var phpMagicConstants = map[string]bool{
	"__class__": true, "__dir__": true, "__file__": true, "__function__": true, "__line__": true,
	"__method__": true, "__namespace__": true, "__trait__": true, "__property__": true,
}

// phpNonStringNames are words blink lexes as identifiers that PHP tokenizes as
// their own token kind (T_EXIT, T_EVAL, T_HALT_COMPILER).
var phpNonStringNames = map[string]bool{"exit": true, "die": true, "eval": true, "__halt_compiler": true}

var phpCastNames = map[string]bool{
	"int": true, "integer": true, "bool": true, "boolean": true, "float": true, "double": true,
	"real": true, "string": true, "binary": true, "object": true, "unset": true, "void": true,
}

// computeTString reports whether token i is a T_STRING in PHP-CS-Fixer's model.
func (sc *symScan) computeTString(i int) bool {
	t := sc.s.At(i)
	if t.Kind != token.Ident {
		return false
	}
	lower := strings.ToLower(t.Value)
	prev, next := sc.prevM(i), sc.nextM(i)
	member := sc.isPunct(prev, "->") || sc.isPunct(prev, "?->") || sc.isPunct(prev, "::") ||
		sc.isKeyword(prev, "function", "const")
	nsPart := sc.isPunct(prev, `\`) || sc.isPunct(next, `\`)
	if lower == "namespace" && !sc.isPunct(prev, `\`) && sc.isPunct(next, `\`) {
		return false // the namespace operator (namespace\foo)
	}
	if !member && !nsPart && (phpMagicConstants[lower] || phpNonStringNames[lower]) {
		return false
	}
	if lower == "from" && sc.isKeyword(prev, "yield") {
		return false // part of T_YIELD_FROM
	}
	// "(int)" and friends are one cast token in PHP
	if phpCastNames[lower] && i > 0 && i+1 < sc.n {
		l, r := i-1, i+1
		if sc.isKind(l, token.Whitespace) && strings.Trim(sc.s.At(l).Value, " \t") == "" {
			l--
		}
		if sc.isKind(r, token.Whitespace) && strings.Trim(sc.s.At(r).Value, " \t") == "" {
			r++
		}
		if sc.isPunct(l, "(") && sc.isPunct(r, ")") {
			return false
		}
	}
	// a named argument name (foo(name: 1)) is CT::T_NAMED_ARGUMENT_NAME
	if sc.isPunct(next, ":") && (sc.isPunct(prev, "(") || sc.isPunct(prev, ",")) {
		return false
	}
	return true
}

// isNamespaceKeyword reports a T_NAMESPACE (a declaration, not the operator).
func (sc *symScan) isNamespaceKeyword(i int) bool {
	if !sc.isKeyword(i, "namespace") {
		return false
	}
	return !sc.isPunct(sc.nextM(i), `\`)
}

// isNamespaceOperator reports CT::T_NAMESPACE_OPERATOR ("namespace\").
func (sc *symScan) isNamespaceOperator(i int) bool {
	if i < 0 || i >= sc.n {
		return false
	}
	t := sc.s.At(i)
	if (t.Kind != token.Keyword && t.Kind != token.Ident) || !strings.EqualFold(t.Value, "namespace") {
		return false
	}
	return !sc.isPunct(sc.prevM(i), `\`) && sc.isPunct(sc.nextM(i), `\`)
}

// isClassy reports a class/interface/trait/enum declaration keyword (T_CLASS
// includes anonymous classes; Foo::class is never a keyword token here).
func (sc *symScan) isClassy(i int) bool {
	if sc.isKeyword(i, "class", "interface", "trait") {
		return !sc.isPunct(sc.prevM(i), "::")
	}
	if sc.isKeyword(i, "enum") {
		return sc.isKind(sc.nextM(i), token.Ident)
	}
	return false
}

// classBodyOpen returns the "{" opening the body of the classy keyword at i,
// stepping over an anonymous class's constructor arguments.
func (sc *symScan) classBodyOpen(i int) int {
	for j := i + 1; j < sc.n; j++ {
		if sc.isPunct(j, "(") {
			if sc.match[j] < 0 {
				return -1
			}
			j = sc.match[j]
			continue
		}
		if sc.isPunct(j, "{") {
			return j
		}
		if sc.isPunct(j, ";") {
			return -1
		}
	}
	return -1
}

func (sc *symScan) isObjectOperator(i int) bool {
	return sc.isPunct(i, "->") || sc.isPunct(i, "?->")
}

// isReturnRef reports CT::T_RETURN_REF: "&" right after function/fn.
func (sc *symScan) isReturnRef(i int) bool {
	return sc.isPunct(i, "&") && sc.isKeyword(sc.prevM(i), "function", "fn")
}

// isLambdaUse reports CT::T_USE_LAMBDA: a closure's "use (".
func (sc *symScan) isLambdaUse(i int) bool {
	return sc.isKeyword(i, "use") && sc.isPunct(sc.prevM(i), ")")
}

// isTraitUse reports CT::T_USE_TRAIT.
func (sc *symScan) isTraitUse(i int) bool {
	return sc.isKeyword(i, "use") && !sc.isLambdaUse(i) && sc.inClassBody[i]
}

// isImportUse reports a T_USE that imports a namespace symbol.
func (sc *symScan) isImportUse(i int) bool {
	return sc.isKeyword(i, "use") && !sc.isLambdaUse(i) && !sc.inClassBody[i]
}

// isImportKindKeyword reports CT::T_FUNCTION_IMPORT / CT::T_CONST_IMPORT: the
// "function"/"const" of "use function"/"use const" (or inside a group import).
func (sc *symScan) isImportKindKeyword(i int) bool { return i >= 0 && i < sc.n && sc.importKW[i] }

func (sc *symScan) computeImportKind(i int) bool {
	if !sc.isKeyword(i, "function", "const") {
		return false
	}
	if p := sc.prevM(i); sc.isImportUse(p) {
		return true
	}
	// inside a group import: the next "=", "(" or "}" is the group's close brace
	for j := i + 1; j < sc.n; j++ {
		if sc.isPunct(j, "=") || sc.isPunct(j, "(") || sc.isReturnRef(j) {
			return false
		}
		if sc.isPunct(j, "}") {
			return sc.isGroupImportOpen(sc.match[j])
		}
	}
	return false
}

// isFunctionKeyword reports a T_FUNCTION (not the "function" of "use function").
func (sc *symScan) isFunctionKeyword(i int) bool {
	return sc.isKeyword(i, "function") && !sc.isImportKindKeyword(i)
}

// isConstKeyword reports a T_CONST (not the "const" of "use const").
func (sc *symScan) isConstKeyword(i int) bool {
	return sc.isKeyword(i, "const") && !sc.isImportKindKeyword(i)
}

// isGroupImportOpen reports CT::T_GROUP_IMPORT_BRACE_OPEN ("use A\{").
func (sc *symScan) isGroupImportOpen(i int) bool {
	return sc.isPunct(i, "{") && sc.isPunct(sc.prevM(i), `\`)
}

// isArrayTypehint reports CT::T_ARRAY_TYPEHINT: "array" not followed by "(".
func (sc *symScan) isArrayTypehint(i int) bool {
	return sc.isKeyword(i, "array") && !sc.isPunct(sc.nextM(i), "(")
}

// isTypeColon mirrors TypeColonTransformer.
func (sc *symScan) isTypeColon(i int) bool {
	if !sc.isPunct(i, ":") {
		return false
	}
	end := sc.prevM(i)
	if end < 0 {
		return false
	}
	if sc.isKeyword(sc.prevM(end), "enum") {
		return true
	}
	if !sc.isPunct(end, ")") || sc.match[end] < 0 {
		return false
	}
	p := sc.prevM(sc.match[end])
	if sc.isTString(p) {
		p = sc.prevM(p)
	}
	return sc.isKeyword(p, "function", "fn") || sc.isReturnRef(p) || sc.isLambdaUse(p)
}

// typeToken reports AbstractTypeTransformer::TYPE_TOKENS membership.
func (sc *symScan) typeToken(j int) bool {
	t := sc.s.At(j)
	switch t.Kind {
	case token.Whitespace, token.Comment, token.DocComment:
		return true
	case token.Punct:
		switch t.Value {
		case "|", "&", "(", ")", `\`:
			return true
		}
		return false
	}
	return sc.isKeyword(j, "callable", "static") || sc.isTString(j) || sc.isArrayTypehint(j)
}

// isPartOfType mirrors AbstractTypeTransformer::isPartOfType for the "|" or "&"
// at index i.
func (sc *symScan) isPartOfType(i int) bool {
	b := i - 1
	for b >= 0 && sc.typeToken(b) {
		b--
	}
	if b >= 0 && (sc.isKeyword(b, "catch") || sc.isTypeColon(b) || sc.isConstKeyword(b)) {
		return true
	}
	a := i + 1
	for a < sc.n && sc.typeToken(a) {
		a++
	}
	if a >= sc.n {
		return false
	}
	if sc.isPunct(a, "...") {
		return true
	}
	if !sc.isKind(a, token.Variable) {
		return false
	}
	bv := sc.prevM(a)
	if sc.isPunct(bv, "&") {
		for j := i - 1; j >= 0; j-- {
			if sc.isPunct(j, "{") || sc.isPunct(j, "}") || sc.isPunct(j, ";") || sc.isKind(j, token.CloseTag) {
				return false
			}
			if sc.isKeyword(j, "fn") || sc.isFunctionKeyword(j) {
				return true
			}
		}
		return false
	}
	return sc.isPunct(bv, ")") || sc.isKeyword(bv, "callable", "static") || sc.isPunct(bv, `\`) ||
		sc.isTString(bv) || sc.isArrayTypehint(bv)
}

// isTypeAlternation reports CT::T_TYPE_ALTERNATION.
func (sc *symScan) isTypeAlternation(i int) bool {
	return sc.isPunct(i, "|") && sc.isPartOfType(i)
}

// isTypeIntersection reports CT::T_TYPE_INTERSECTION: an "&" not followed by a
// variable or "..." that is part of a type.
func (sc *symScan) isTypeIntersection(i int) bool {
	if !sc.isPunct(i, "&") || sc.isReturnRef(i) {
		return false
	}
	nx := sc.nextM(i)
	if sc.isKind(nx, token.Variable) || sc.isPunct(nx, "...") {
		return false
	}
	return sc.isPartOfType(i)
}

// isNullableType mirrors NullableTypeTransformer.
func (sc *symScan) isNullableType(i int) bool {
	if !sc.isPunct(i, "?") {
		return false
	}
	p := sc.prevM(i)
	if p < 0 {
		return false
	}
	switch {
	case sc.isPunct(p, "(") || sc.isPunct(p, ",") || sc.isTypeColon(p) || sc.isKind(p, token.AttributeEnd):
		return true
	case sc.isKeyword(p, "static"):
		return !sc.isKeyword(sc.prevM(p), "instanceof")
	case sc.isKeyword(p, "private", "protected", "public", "var", "abstract", "final", "readonly") || sc.isConstKeyword(p):
		return true
	case sc.isPunct(p, ")") && sc.match[p] >= 0:
		// asymmetric visibility: private(set)
		o := sc.match[p]
		in := sc.nextM(o)
		return in >= 0 && strings.EqualFold(sc.s.At(in).Value, "set") && sc.nextM(in) == p &&
			sc.isKeyword(sc.prevM(o), "private", "protected", "public")
	}
	return false
}

// isDNFClose reports CT::T_DISJUNCTIVE_NORMAL_FORM_TYPE_PARENTHESIS_CLOSE.
func (sc *symScan) isDNFClose(i int) bool {
	if !sc.isPunct(i, ")") {
		return false
	}
	if sc.isTypeAlternation(sc.nextM(i)) {
		return true
	}
	o := sc.match[i]
	return o >= 0 && sc.isTypeAlternation(sc.prevM(o))
}

// isAttribute mirrors AttributeAnalyzer::isAttribute: token i names an
// attribute class (#[Foo] or #[Foo(...)], not a token inside its arguments).
func (sc *symScan) isAttribute(i int) bool {
	if !sc.isTString(i) || !sc.hasAttr {
		return false
	}
	start := -1
	for j := i - 1; j >= 0; j-- {
		t := sc.s.At(j)
		if t.Kind == token.AttributeStart || t.Kind == token.AttributeEnd || t.Kind == token.Variable || t.Kind == token.OpenTag ||
			sc.isPunct(j, ";") || sc.isPunct(j, "{") ||
			sc.isFunctionKeyword(j) || sc.isKeyword(j, "private", "protected", "public", "return") {
			start = j
			break
		}
	}
	if start < 0 || sc.s.At(start).Kind != token.AttributeStart {
		return false
	}
	count := 0
	for j := start + 1; j < i; j++ {
		if sc.isPunct(j, "(") {
			count++
		} else if sc.isPunct(j, ")") {
			count--
		}
	}
	return count == 0
}

// belongsToGotoLabel mirrors GotoLabelAnalyzer for the ":" at index i.
func (sc *symScan) belongsToGotoLabel(i int) bool {
	if !sc.isPunct(i, ":") {
		return false
	}
	p := sc.prevM(i)
	if !sc.isTString(p) {
		return false
	}
	pp := sc.prevM(p)
	return sc.isPunct(pp, ":") || sc.isPunct(pp, ";") || sc.isPunct(pp, "{") || sc.isPunct(pp, "}") || sc.isKind(pp, token.OpenTag)
}

// isConstantInvocation mirrors TokensAnalyzer::isConstantInvocation.
func (sc *symScan) isConstantInvocation(i int) bool {
	if !sc.isTString(i) {
		return false
	}
	next := sc.nextM(i)
	if sc.isPunct(next, "(") || sc.isPunct(next, "{") || sc.isPunct(next, "::") || sc.isPunct(next, "...") ||
		sc.isPunct(next, `\`) || sc.isReturnRef(next) || sc.isTypeAlternation(next) || sc.isTypeIntersection(next) ||
		sc.isKind(next, token.Variable) {
		return false
	}
	prev := sc.prevM(i)
	if sc.isKeyword(next, "as") && !sc.isPunct(prev, "(") {
		return false
	}
	if sc.isClassy(prev) {
		return false
	}
	if sc.isKeyword(prev, "as", "goto") || sc.isConstKeyword(prev) || sc.isPunct(prev, "::") || sc.isFunctionKeyword(prev) ||
		sc.isGroupImportOpen(prev) || sc.isTypeColon(prev) || sc.isTypeAlternation(prev) || sc.isTypeIntersection(prev) ||
		sc.isObjectOperator(prev) {
		return false
	}
	if sc.isKeyword(prev, "case") && sc.hasEnum {
		for j := i - 1; j >= 0; j-- {
			if sc.isKeyword(j, "switch") {
				break
			}
			if sc.isKeyword(j, "enum") {
				return false
			}
		}
	}
	for prev >= 0 && (sc.isNamespaceOperator(prev) || sc.isPunct(prev, `\`) || sc.isTString(prev) || sc.isArrayTypehint(prev)) {
		prev = sc.prevM(prev)
	}
	if sc.isImportKindKeyword(prev) || sc.isKeyword(prev, "extends", "implements", "instanceof", "insteadof", "new") ||
		sc.isNamespaceKeyword(prev) || sc.isNullableType(prev) || sc.isTypeColon(prev) || sc.isImportUse(prev) ||
		sc.isTraitUse(prev) || sc.isTypeIntersection(prev) || sc.isTypeAlternation(prev) || sc.isConstKeyword(prev) ||
		sc.isDNFClose(prev) {
		return false
	}
	// FOO & $bar: a by-reference parameter type or a bitwise and
	if sc.isPunct(next, "&") && sc.isKind(sc.nextM(next), token.Variable) {
		for j := prev - 1; j >= 0; j-- {
			if sc.isPunct(j, ";") || sc.isPunct(j, "{") || sc.isPunct(j, "}") || sc.isKind(j, token.OpenTag) {
				break
			}
			if sc.isFunctionKeyword(j) {
				return false
			}
		}
	}
	// an extends/implements/use list
	if sc.isPunct(prev, ",") {
		c := prev
		for c >= 0 && (sc.isPunct(c, ",") || sc.isKeyword(c, "as") || sc.isNamespaceOperator(c) || sc.isPunct(c, `\`) || sc.isTString(c)) {
			c = sc.prevM(c)
		}
		if sc.isKeyword(c, "extends", "implements") || sc.isGroupImportOpen(c) || sc.isImportUse(c) || sc.isTraitUse(c) {
			return false
		}
	}
	// an array key inside a double-quoted string: "..$foo[bar].."
	if sc.isPunct(prev, "[") && sc.isPunct(next, "]") {
		if c := sc.nextM(next); sc.isKind(c, token.String) && !strings.HasPrefix(sc.s.At(c).Value, "}") {
			return false
		}
	}
	if sc.isAttribute(i) {
		return false
	}
	if sc.isPunct(next, ":") && sc.belongsToGotoLabel(next) {
		return false
	}
	// a non-capturing catch: catch (Foo)
	for prev >= 0 && (sc.isNamespaceOperator(prev) || sc.isPunct(prev, `\`) || sc.isTString(prev) || sc.isTypeAlternation(prev)) {
		prev = sc.prevM(prev)
	}
	if sc.isPunct(prev, "(") && sc.isKeyword(sc.prevM(prev), "catch") {
		return false
	}
	return true
}

var phpReservedTypes = map[string]bool{
	"array": true, "bool": true, "callable": true, "false": true, "float": true, "int": true, "iterable": true,
	"list": true, "mixed": true, "never": true, "null": true, "object": true, "parent": true, "resource": true,
	"self": true, "static": true, "string": true, "true": true, "void": true,
}

// isClassyInvocation mirrors ClassyAnalyzer::isClassyInvocation.
func (sc *symScan) isClassyInvocation(i int) bool {
	if !sc.isTString(i) || phpReservedTypes[strings.ToLower(sc.s.At(i).Value)] {
		return false
	}
	next := sc.nextM(i)
	if sc.isPunct(next, `\`) {
		return false
	}
	if sc.isPunct(next, "::") || sc.isPunct(next, "...") || sc.isTypeAlternation(next) || sc.isTypeIntersection(next) ||
		sc.isKind(next, token.Variable) {
		return true
	}
	prev := sc.prevM(i)
	for prev >= 0 && (sc.isNamespaceOperator(prev) || sc.isPunct(prev, `\`) || sc.isTString(prev)) {
		prev = sc.prevM(prev)
	}
	if sc.isKeyword(prev, "extends", "instanceof", "insteadof", "implements", "new") || sc.isNullableType(prev) ||
		sc.isTypeAlternation(prev) || sc.isTypeIntersection(prev) || sc.isTypeColon(prev) || sc.isTraitUse(prev) {
		return true
	}
	if sc.isPunct(next, ")") && sc.isPunct(prev, "(") && sc.isKeyword(sc.prevM(prev), "catch") {
		return true
	}
	if sc.isAttribute(i) {
		return true
	}
	if sc.isPunct(next, "&") && sc.isKind(sc.nextM(next), token.Variable) {
		for j := prev; j >= 0; j-- {
			if sc.isPunct(j, ";") || sc.isPunct(j, "{") || sc.isPunct(j, "}") || sc.isKind(j, token.OpenTag) {
				return false
			}
			if sc.isFunctionKeyword(j) {
				return true
			}
		}
		return false
	}
	if !sc.isPunct(prev, ",") {
		return false
	}
	for {
		prev = sc.prevM(prev)
		if !sc.isPunct(prev, ",") && !sc.isPunct(prev, `\`) && !sc.isTString(prev) && !sc.isNamespaceOperator(prev) {
			break
		}
	}
	return sc.isKeyword(prev, "implements") || sc.isTraitUse(prev)
}

// computeNamespaces mirrors NamespacesAnalyzer::getDeclarations.
func (sc *symScan) computeNamespaces() []nsDecl {
	var out []nsDecl
	for i := 1; i < sc.n; i++ {
		if !sc.isNamespaceKeyword(i) {
			continue
		}
		end := -1
		for j := i + 1; j < sc.n; j++ {
			if sc.isPunct(j, ";") || sc.isPunct(j, "{") {
				end = j
				break
			}
		}
		if end < 0 {
			break
		}
		var b strings.Builder
		for j := i + 1; j < end; j++ {
			b.WriteString(sc.s.At(j).Value)
		}
		name := strings.TrimSpace(b.String())
		scopeEnd := sc.n - 1
		if sc.isPunct(end, "{") {
			if sc.match[end] >= 0 {
				scopeEnd = sc.match[end]
			}
		} else {
			for j := end + 1; j < sc.n; j++ {
				if sc.isNamespaceKeyword(j) {
					scopeEnd = j - 1
					break
				}
			}
		}
		out = append(out, nsDecl{name: name, declEnd: end, scopeStart: i, scopeEnd: scopeEnd})
		i = scopeEnd
	}
	if len(out) == 0 {
		hasOpen := false
		for i := range sc.n {
			if sc.s.At(i).Kind == token.OpenTag && sc.s.At(i).Value != "<?=" {
				hasOpen = true
				break
			}
		}
		if hasOpen {
			start := 0
			if sc.n > 0 && sc.s.At(0).Kind == token.InlineHTML {
				start = 1
			}
			out = append(out, nsDecl{declEnd: start, scopeStart: start, scopeEnd: sc.n - 1})
		}
	}
	return out
}

// computeUses mirrors NamespaceUsesAnalyzer::getDeclarationsFromTokens with
// $allowMultiUses = false: "use A, B;" and group imports are left out.
func (sc *symScan) computeUses() {
	for i := range sc.n {
		if !sc.isImportUse(i) {
			continue
		}
		end := -1
		for j := i + 1; j < sc.n; j++ {
			if sc.isPunct(j, ";") || sc.isKind(j, token.CloseTag) {
				end = j
				break
			}
		}
		if end < 0 {
			continue
		}
		multi := false
		for j := i + 1; j < end; j++ {
			if sc.isPunct(j, ",") || sc.isGroupImportOpen(j) {
				multi = true
				break
			}
		}
		if multi {
			continue
		}
		kind := "class"
		if k := sc.nextM(i); sc.isKeyword(k, "function") {
			kind = "function"
		} else if sc.isKeyword(k, "const") {
			kind = "const"
		}
		start := -1
		for j := i + 1; j <= end; j++ {
			if sc.isTString(j) || sc.isPunct(j, `\`) {
				start = j
				break
			}
		}
		if start < 0 {
			continue
		}
		var full strings.Builder
		short, aliased := "", false
		for j := start; j >= 0 && j < end; j = sc.nextM(j) {
			switch {
			case sc.isTString(j):
				short = sc.s.At(j).Value
				if !aliased {
					full.WriteString(short)
				}
			case sc.isPunct(j, `\`):
				full.WriteString(`\`)
			case sc.isKeyword(j, "as"):
				aliased = true
			}
		}
		if full.Len() == 0 || short == "" {
			continue
		}
		sc.uses = append(sc.uses, useDecl{kind: kind, fullName: full.String(), shortName: short, aliased: aliased, start: i, end: end})
	}
}

// functionDeclarations returns the name-token indices of every named function
// declared outside class-like bodies (FunctionsAnalyzer::buildFunctionsAnalysis).
func (sc *symScan) functionDeclarations() []int {
	var out []int
	for i := 0; i < sc.n; i++ {
		if sc.isClassy(i) {
			j := i + 1
			for j < sc.n && !sc.isPunct(j, "(") && !sc.isPunct(j, "{") {
				j++
			}
			if j < sc.n && sc.isPunct(j, "(") {
				if sc.match[j] < 0 {
					break
				}
				j = sc.nextPunct(sc.match[j], "{")
			}
			if j < 0 || j >= sc.n || sc.match[j] < 0 {
				break
			}
			i = sc.match[j]
			continue
		}
		if !sc.isFunctionKeyword(i) {
			continue
		}
		k := sc.nextM(i)
		if sc.isReturnRef(k) {
			k = sc.nextM(k)
		}
		if sc.isTString(k) {
			out = append(out, k)
			i = k
		}
	}
	return out
}

// isPropertyHookBrace approximates CT::T_PROPERTY_HOOK_BRACE_OPEN: a "{" in a
// class body that follows a property declaration rather than a method header.
func (sc *symScan) isPropertyHookBrace(i int) bool {
	if !sc.isPunct(i, "{") || !sc.inClassBody[i] {
		return false
	}
	sawVar := false
	for j := sc.prevM(i); j >= 0; j = sc.prevM(j) {
		if sc.isPunct(j, ";") || sc.isPunct(j, "{") || sc.isPunct(j, "}") {
			break
		}
		if sc.isFunctionKeyword(j) {
			return false
		}
		if sc.isKind(j, token.Variable) {
			sawVar = true
		}
	}
	return sawVar
}

// funcScan answers FunctionsAnalyzer::isGlobalFunctionUsage.
type funcScan struct {
	*symScan
	decls   []int
	imports []useDecl
}

func newFuncScan(sc *symScan) *funcScan {
	f := &funcScan{symScan: sc, decls: sc.functionDeclarations()}
	for _, u := range sc.uses {
		if u.kind == "function" {
			f.imports = append(f.imports, u)
		}
	}
	return f
}

func (f *funcScan) isGlobalFunctionUsage(i int) bool { return f.globalFunction(i, false) }

func (f *funcScan) globalFunction(i int, callOnly bool) bool {
	sc := f.symScan
	if !sc.isTString(i) {
		return false
	}
	open := sc.nextM(i)
	if !sc.isPunct(open, "(") {
		return false
	}
	prevIsNs := false
	prev := sc.prevM(i)
	if sc.isPunct(prev, `\`) {
		prevIsNs = true
		prev = sc.prevM(prev)
	}
	if sc.isPunct(prev, "::") || sc.isFunctionKeyword(prev) || sc.isNamespaceOperator(prev) || sc.isKeyword(prev, "new") ||
		sc.isReturnRef(prev) || sc.isTString(prev) || sc.isObjectOperator(prev) || sc.isKind(prev, token.AttributeStart) {
		return false
	}
	if callOnly {
		if e := sc.nextM(open); sc.isPunct(e, "...") && sc.isPunct(sc.nextM(e), ")") {
			return false // first-class callable
		}
	}
	if prevIsNs {
		return true
	}
	name := strings.ToLower(sc.s.At(i).Value)
	if name == "set" {
		if !sc.isPropertyHookBrace(prev) && !sc.isPunct(prev, ";") && !sc.isPunct(prev, "}") {
			return true
		}
		if c := sc.match[open]; c >= 0 {
			if a := sc.nextM(c); sc.isPunct(a, "{") || sc.isPunct(a, "=>") {
				return false
			}
		}
	}
	scopeStart, scopeEnd, inGlobal := 0, sc.n-1, false
	for _, d := range sc.nsDecls {
		scopeStart, scopeEnd = d.scopeStart, d.scopeEnd
		if i >= scopeStart && i <= scopeEnd {
			inGlobal = d.isGlobal()
			break
		}
	}
	if !inGlobal {
		for _, d := range f.decls {
			if d < scopeStart || d > scopeEnd {
				continue
			}
			if strings.ToLower(sc.s.At(d).Value) == name {
				return false
			}
		}
	}
	for _, u := range f.imports {
		if u.start < scopeStart || u.end > scopeEnd {
			continue
		}
		if name != strings.ToLower(u.shortName) {
			continue
		}
		return u.shortName == strings.TrimLeft(u.fullName, `\`)
	}
	return !sc.isAttribute(i)
}

// streamEdits collects token edits computed on an unchanged stream and applies
// them in one go, highest index first, so the indices stay valid.
type streamEdits struct {
	removes map[int]bool
	inserts map[int][]token.Token // tokens inserted before the index
	sets    map[int]string
}

func newStreamEdits() *streamEdits {
	return &streamEdits{removes: map[int]bool{}, inserts: map[int][]token.Token{}, sets: map[int]string{}}
}

// apply performs the edits. A removed token whose neighbours are both
// whitespace merges them (Tokens::clearTokenAndMergeSurroundingWhitespace).
func (e *streamEdits) apply(s *tokens.Stream, mergeWhitespace bool) bool {
	changed := false
	for i, v := range e.sets {
		if s.At(i).Value != v {
			s.SetValue(i, v)
			changed = true
		}
	}
	for i := s.Len(); i >= 0; i-- {
		if e.removes[i] {
			s.RemoveAt(i)
			changed = true
			if mergeWhitespace && i > 0 && i < s.Len() && s.Kind(i-1) == token.Whitespace && s.Kind(i) == token.Whitespace {
				s.SetValue(i-1, s.At(i-1).Value+s.At(i).Value)
				s.RemoveAt(i)
			}
		}
		if ins, ok := e.inserts[i]; ok && len(ins) > 0 {
			s.InsertSliceAt(i, ins)
			changed = true
		}
	}
	return changed
}

// nsSeparatorToken is the "\" PHP-CS-Fixer inserts.
func nsSeparatorToken() token.Token { return token.Token{Kind: token.Punct, Value: `\`} }
