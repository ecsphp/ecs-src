package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// clearableTokens is a copy of the stream shaped like PHP-CS-Fixer's Tokens, so
// NoUnusedImportsFixer's analyzers and clean-up code port one to one:
//   - a removed token is cleared in place, so indices stay stable, and counts
//     as whitespace, as in PHP;
//   - the open tag carries its first whitespace character and the close tag its
//     line break, as T_OPEN_TAG and T_CLOSE_TAG do; writeBack splits them again;
//   - use statements are classified once and group import braces are marked,
//     standing in for PHP-CS-Fixer's transformers.
type clearableTokens struct {
	toks    []token.Token
	cleared []bool
	match   []int // index of the matching bracket for ( ) [ ] { }, else -1

	importUse  []bool // a T_USE importing a namespace symbol
	traitUse   []bool // CT::T_USE_TRAIT
	groupBrace []bool // CT::T_GROUP_IMPORT_BRACE_OPEN / _CLOSE

	hasAttribute, hasEnum bool
}

func newClearableTokens(s *tokens.Stream) *clearableTokens {
	src := s.Tokens()
	c := &clearableTokens{toks: make([]token.Token, 0, len(src))}
	var origUse []int // view index -> stream index, for the "use" keywords
	var viewUse []int
	for i := 0; i < len(src); i++ {
		t := src[i]
		if i+1 < len(src) {
			n := src[i+1]
			if t.Kind == token.OpenTag && !strings.HasPrefix(t.Value, "<?=") && n.Kind == token.Whitespace {
				head := firstBlankOrNewline(n.Value)
				t.Value += head
				c.toks = append(c.toks, t)
				if rest := n.Value[len(head):]; rest != "" {
					c.toks = append(c.toks, token.Token{Kind: token.Whitespace, Value: rest, Pos: n.Pos + len(head)})
				}
				i++
				continue
			}
			if t.Kind == token.CloseTag && n.Kind == token.InlineHTML {
				if head := leadingNewline(n.Value); head != "" {
					t.Value += head
					c.toks = append(c.toks, t)
					if rest := n.Value[len(head):]; rest != "" {
						c.toks = append(c.toks, token.Token{Kind: token.InlineHTML, Value: rest, Pos: n.Pos + len(head)})
					}
					i++
					continue
				}
			}
		}
		if t.Kind == token.Keyword && strings.EqualFold(t.Value, "use") {
			origUse = append(origUse, i)
			viewUse = append(viewUse, len(c.toks))
		}
		c.toks = append(c.toks, t)
	}
	n := len(c.toks)
	c.cleared = make([]bool, n)
	c.match = make([]int, n)
	c.importUse = make([]bool, n)
	c.traitUse = make([]bool, n)
	c.groupBrace = make([]bool, n)
	var stack []int
	for i, t := range c.toks {
		c.match[i] = -1
		if t.Value == "" {
			c.cleared[i] = true
		}
		switch t.Kind {
		case token.AttributeStart:
			c.hasAttribute = true
		case token.Keyword:
			if strings.EqualFold(t.Value, "enum") {
				c.hasEnum = true
			}
		case token.Punct:
			switch t.Value {
			case "(", "[", "{":
				stack = append(stack, i)
			case ")", "]", "}":
				if len(stack) > 0 {
					o := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					c.match[o], c.match[i] = i, o
				}
			}
		}
	}
	for k, vi := range viewUse {
		switch {
		case c.punct(c.nextMeaningful(vi), "("):
			// closure use: CT::T_USE_LAMBDA
		case inClassLikeBody(s, origUse[k]):
			c.traitUse[vi] = true
		default:
			c.importUse[vi] = true
			end := c.nextTokenOf(vi, func(j int) bool { return c.punct(j, ";") || c.kind(j, token.CloseTag) })
			for j := vi + 1; j >= 0 && j < end; j++ {
				if c.punct(j, "{") && c.match[j] >= 0 {
					c.groupBrace[j], c.groupBrace[c.match[j]] = true, true
				}
			}
		}
	}
	return c
}

// firstBlankOrNewline returns what T_OPEN_TAG takes of the whitespace after
// "<?php": one space or tab, or one line break.
func firstBlankOrNewline(v string) string {
	if strings.HasPrefix(v, "\r\n") {
		return "\r\n"
	}
	return v[:1]
}

// leadingNewline returns the line break T_CLOSE_TAG takes after "?>", if any.
func leadingNewline(v string) string {
	switch {
	case strings.HasPrefix(v, "\r\n"):
		return "\r\n"
	case strings.HasPrefix(v, "\n"), strings.HasPrefix(v, "\r"):
		return v[:1]
	}
	return ""
}

// writeBack replaces the stream's tokens with the non-cleared ones, splitting
// the open and close tags back into blink's shape and merging the whitespace
// (or inline HTML) runs that clearing left side by side.
func (c *clearableTokens) writeBack(s *tokens.Stream) {
	out := make([]token.Token, 0, len(c.toks))
	push := func(t token.Token) {
		if t.Value == "" {
			return
		}
		if l := len(out) - 1; l >= 0 && out[l].Kind == t.Kind && (t.Kind == token.Whitespace || t.Kind == token.InlineHTML) {
			out[l].Value += t.Value
			return
		}
		out = append(out, t)
	}
	for i, t := range c.toks {
		if c.cleared[i] {
			continue
		}
		switch t.Kind {
		case token.OpenTag:
			if tag := strings.TrimRight(t.Value, " \t\r\n"); tag != t.Value {
				push(token.Token{Kind: token.OpenTag, Value: tag, Pos: t.Pos})
				push(token.Token{Kind: token.Whitespace, Value: t.Value[len(tag):], Pos: t.Pos + len(tag)})
				continue
			}
		case token.CloseTag:
			if tag := strings.TrimRight(t.Value, "\r\n"); tag != t.Value {
				push(token.Token{Kind: token.CloseTag, Value: tag, Pos: t.Pos})
				push(token.Token{Kind: token.InlineHTML, Value: t.Value[len(tag):], Pos: t.Pos + len(tag)})
				continue
			}
		}
		push(t)
	}
	s.ReplaceRange(0, s.Len()-1, out)
}

func (c *clearableTokens) valid(i int) bool { return i >= 0 && i < len(c.toks) }

func (c *clearableTokens) clear(i int) { c.cleared[i] = true }

func (c *clearableTokens) kind(i int, k token.Kind) bool {
	return c.valid(i) && !c.cleared[i] && c.toks[i].Kind == k
}

func (c *clearableTokens) punct(i int, vals ...string) bool {
	return c.kind(i, token.Punct) && slices.Contains(vals, c.toks[i].Value)
}

func (c *clearableTokens) keyword(i int, words ...string) bool {
	if !c.kind(i, token.Keyword) {
		return false
	}
	for _, w := range words {
		if strings.EqualFold(c.toks[i].Value, w) {
			return true
		}
	}
	return false
}

// isWhitespace is Token::isWhitespace(): a whitespace token or a cleared one.
func (c *clearableTokens) isWhitespace(i int) bool {
	return c.valid(i) && (c.cleared[i] || c.toks[i].Kind == token.Whitespace)
}

func (c *clearableTokens) isComment(i int) bool {
	return c.kind(i, token.Comment) || c.kind(i, token.DocComment)
}

func (c *clearableTokens) value(i int) string {
	if !c.valid(i) || c.cleared[i] {
		return ""
	}
	return c.toks[i].Value
}

// hasID reports whether PHP-CS-Fixer's token has an id: everything but cleared
// tokens and the single-character tokens PHP leaves untyped (";", ",", ...).
func (c *clearableTokens) hasID(i int) bool {
	if !c.valid(i) || c.cleared[i] {
		return false
	}
	t := c.toks[i]
	return t.Kind != token.Punct || len(t.Value) > 1 || t.Value == `\` || c.groupBrace[i]
}

// setWhitespace is Tokens::ensureWhitespaceAtIndex($i, 0, $v) on a token that
// is (or was cleared to) whitespace: an empty value clears it.
func (c *clearableTokens) setWhitespace(i int, v string) {
	if v == "" {
		c.clear(i)
		return
	}
	c.toks[i] = token.Token{Kind: token.Whitespace, Value: v, Pos: c.toks[i].Pos}
	c.cleared[i] = false
}

// nonEmptySibling is Tokens::getNonEmptySibling(): the next index in direction
// dir that is not cleared, or -1.
func (c *clearableTokens) nonEmptySibling(i, dir int) int {
	for i += dir; c.valid(i); i += dir {
		if !c.cleared[i] {
			return i
		}
	}
	return -1
}

func (c *clearableTokens) meaningful(i int) bool {
	return c.valid(i) && !c.isWhitespace(i) && !c.isComment(i)
}

func (c *clearableTokens) nonWhitespace(i int) bool { return c.valid(i) && !c.isWhitespace(i) }

// nextMeaningful / prevMeaningful are getNextMeaningfulToken() and
// getPrevMeaningfulToken(), or -1.
func (c *clearableTokens) nextMeaningful(i int) int { return c.nextTokenOf(i, c.meaningful) }
func (c *clearableTokens) prevMeaningful(i int) int { return c.prevTokenOf(i, c.meaningful) }

// nextTokenOf / prevTokenOf are getNextTokenOfKind() / getPrevTokenOfKind():
// the first index after (before) i whose token satisfies is, or -1.
func (c *clearableTokens) nextTokenOf(i int, is func(int) bool) int {
	if i < 0 {
		return -1
	}
	for i++; c.valid(i); i++ {
		if is(i) {
			return i
		}
	}
	return -1
}

func (c *clearableTokens) prevTokenOf(i int, is func(int) bool) int {
	for i--; c.valid(i); i-- {
		if is(i) {
			return i
		}
	}
	return -1
}

// clearAndMergeSurroundingWhitespace is Tokens::clearTokenAndMergeSurroundingWhitespace().
func (c *clearableTokens) clearAndMergeSurroundingWhitespace(i int) {
	c.clear(i)
	next := c.nonEmptySibling(i, 1)
	if next < 0 || !c.isWhitespace(next) {
		return
	}
	prev := c.nonEmptySibling(i, -1)
	switch {
	case c.isWhitespace(prev):
		c.setWhitespace(prev, c.value(prev)+c.value(next))
	case c.valid(prev+1) && c.cleared[prev+1]:
		c.setWhitespace(prev+1, c.value(next))
	}
	c.clear(next)
}

// The token kinds below stand in for the ones PHP-CS-Fixer's tokenizer and
// transformers give; blink's lexer is flatter, so they are read off the
// neighbouring tokens.

// tString reports a T_STRING: a name, but not the namespace operator, the
// "class" of "::class" or a named argument's name.
func (c *clearableTokens) tString(i int) bool {
	if !c.kind(i, token.Ident) {
		return false
	}
	switch strings.ToLower(c.toks[i].Value) {
	case "namespace":
		if c.punct(c.nextMeaningful(i), `\`) {
			return false
		}
	case "class":
		if c.punct(c.prevMeaningful(i), "::") {
			return false
		}
	}
	return !c.isNamedArgumentName(i)
}

// isNamedArgumentName is CT::T_NAMED_ARGUMENT_NAME (NamedArgumentTransformer).
func (c *clearableTokens) isNamedArgumentName(i int) bool {
	return c.punct(c.nextMeaningful(i), ":") && c.punct(c.prevMeaningful(i), ",", "(")
}

// namespaceOperator is CT::T_NAMESPACE_OPERATOR ("namespace\Foo").
func (c *clearableTokens) namespaceOperator(i int) bool {
	return (c.kind(i, token.Ident) || c.kind(i, token.Keyword)) && strings.EqualFold(c.toks[i].Value, "namespace") &&
		c.punct(c.nextMeaningful(i), `\`)
}

// namespaceKeyword is T_NAMESPACE: a namespace declaration.
func (c *clearableTokens) namespaceKeyword(i int) bool {
	return c.keyword(i, "namespace") && !c.namespaceOperator(i) && !c.punct(c.prevMeaningful(i), "->", "?->", "::")
}

func (c *clearableTokens) objectOperator(i int) bool { return c.punct(i, "->", "?->") }

// classy is Token::isClassy(): class, trait, interface or enum.
func (c *clearableTokens) classy(i int) bool {
	return c.keyword(i, "class", "trait", "interface", "enum")
}

// returnRef is CT::T_RETURN_REF: the "&" of "function &foo()".
func (c *clearableTokens) returnRef(i int) bool {
	return c.punct(i, "&") && c.keyword(c.prevMeaningful(i), "function", "fn")
}

// isImportUse is a T_USE that imports a namespace symbol.
func (c *clearableTokens) isImportUse(i int) bool {
	return c.valid(i) && !c.cleared[i] && c.importUse[i]
}

// isTraitUse is CT::T_USE_TRAIT.
func (c *clearableTokens) isTraitUse(i int) bool { return c.valid(i) && !c.cleared[i] && c.traitUse[i] }

// groupOpen / groupClose are CT::T_GROUP_IMPORT_BRACE_OPEN / _CLOSE.
func (c *clearableTokens) groupOpen(i int) bool  { return c.punct(i, "{") && c.groupBrace[i] }
func (c *clearableTokens) groupClose(i int) bool { return c.punct(i, "}") && c.groupBrace[i] }

// lambdaUse is CT::T_USE_LAMBDA.
func (c *clearableTokens) lambdaUse(i int) bool {
	return c.keyword(i, "use") && c.punct(c.nextMeaningful(i), "(")
}

// arrayTypehint is CT::T_ARRAY_TYPEHINT: "array" not followed by "(".
func (c *clearableTokens) arrayTypehint(i int) bool {
	return c.keyword(i, "array") && !c.punct(c.nextMeaningful(i), "(")
}

// typeColon is CT::T_TYPE_COLON (TypeColonTransformer).
func (c *clearableTokens) typeColon(i int) bool {
	if !c.punct(i, ":") {
		return false
	}
	end := c.prevMeaningful(i)
	if c.keyword(c.prevMeaningful(end), "enum") {
		return true
	}
	if !c.punct(end, ")") || c.match[end] < 0 {
		return false
	}
	p := c.prevMeaningful(c.match[end])
	if c.tString(p) {
		p = c.prevMeaningful(p)
	}
	return c.keyword(p, "function", "fn") || c.returnRef(p) || c.lambdaUse(p)
}

// typeToken is AbstractTypeTransformer::TYPE_TOKENS (cleared tokens included,
// as getTokenNotOfKindSibling skips them).
func (c *clearableTokens) typeToken(i int) bool {
	if !c.valid(i) {
		return false
	}
	if c.isWhitespace(i) || c.isComment(i) {
		return true
	}
	return c.punct(i, "|", "&", "(", ")", `\`) || c.keyword(i, "callable", "static") || c.tString(i) || c.arrayTypehint(i)
}

// partOfType is AbstractTypeTransformer::isPartOfType() for the "|" or "&" at i.
func (c *clearableTokens) partOfType(i int) bool {
	b := i - 1
	for c.typeToken(b) {
		b--
	}
	if c.keyword(b, "catch", "const") || c.typeColon(b) {
		return true
	}
	a := i + 1
	for c.typeToken(a) {
		a++
	}
	if c.punct(a, "...") {
		return true
	}
	if !c.kind(a, token.Variable) {
		return false
	}
	bv := c.prevMeaningful(a)
	if c.punct(bv, "&") {
		p := c.prevTokenOf(i, func(j int) bool {
			return c.punct(j, "{", "}", ";") || c.kind(j, token.CloseTag) || c.keyword(j, "fn", "function")
		})
		return c.keyword(p, "fn", "function")
	}
	return c.punct(bv, ")", `\`) || c.keyword(bv, "callable", "static") || c.tString(bv) || c.arrayTypehint(bv)
}

// typeAlternation is CT::T_TYPE_ALTERNATION.
func (c *clearableTokens) typeAlternation(i int) bool { return c.punct(i, "|") && c.partOfType(i) }

// typeIntersection is CT::T_TYPE_INTERSECTION: only an "&" not followed by a
// variable or "..." (T_AMPERSAND_NOT_FOLLOWED_BY_VAR_OR_VARARG) is a candidate.
func (c *clearableTokens) typeIntersection(i int) bool {
	if !c.punct(i, "&") || c.returnRef(i) {
		return false
	}
	if n := c.nextMeaningful(i); c.kind(n, token.Variable) || c.punct(n, "...") {
		return false
	}
	return c.partOfType(i)
}

// nullableType is CT::T_NULLABLE_TYPE (NullableTypeTransformer).
func (c *clearableTokens) nullableType(i int) bool {
	if !c.punct(i, "?") {
		return false
	}
	p := c.prevMeaningful(i)
	switch {
	case c.punct(p, "(", ",") || c.typeColon(p) || c.kind(p, token.AttributeEnd):
		return true
	case c.keyword(p, "static"):
		return !c.keyword(c.prevMeaningful(p), "instanceof")
	case c.keyword(p, "var", "const", "abstract", "final", "readonly"):
		return true
	case c.kind(p, token.Keyword):
		// visibility, also the asymmetric "private(set)" when lexed as one keyword
		v := strings.ToLower(c.toks[p].Value)
		return strings.HasPrefix(v, "private") || strings.HasPrefix(v, "protected") || strings.HasPrefix(v, "public")
	case c.punct(p, ")") && c.match[p] >= 0:
		// asymmetric visibility lexed as "private" "(" "set" ")"
		o := c.match[p]
		in := c.nextMeaningful(o)
		return c.kind(in, token.Ident) && strings.EqualFold(c.toks[in].Value, "set") && c.nextMeaningful(in) == p &&
			c.keyword(c.prevMeaningful(o), "private", "protected", "public")
	}
	return false
}

// dnfClose is CT::T_DISJUNCTIVE_NORMAL_FORM_TYPE_PARENTHESIS_CLOSE.
func (c *clearableTokens) dnfClose(i int) bool {
	if !c.punct(i, ")") {
		return false
	}
	if c.typeAlternation(c.nextMeaningful(i)) {
		return true
	}
	return c.match[i] >= 0 && c.typeAlternation(c.prevMeaningful(c.match[i]))
}

// isAttribute is AttributeAnalyzer::isAttribute(): the name at i is an
// attribute's class name.
func (c *clearableTokens) isAttribute(i int) bool {
	if !c.tString(i) || !c.hasAttribute {
		return false
	}
	start := c.prevTokenOf(i, func(j int) bool {
		return c.punct(j, ";", "{") || c.kind(j, token.AttributeStart) || c.kind(j, token.AttributeEnd) ||
			c.kind(j, token.OpenTag) || c.kind(j, token.Variable) ||
			c.keyword(j, "function", "private", "protected", "public", "return")
	})
	if !c.kind(start, token.AttributeStart) {
		return false
	}
	count := 0
	for j := start + 1; j < i; j++ {
		if c.punct(j, "(") {
			count++
		} else if c.punct(j, ")") {
			count--
		}
	}
	return count == 0
}

// belongsToGotoLabel is GotoLabelAnalyzer::belongsToGoToLabel() for the ":" at i.
func (c *clearableTokens) belongsToGotoLabel(i int) bool {
	if !c.punct(i, ":") {
		return false
	}
	p := c.prevMeaningful(i)
	if !c.tString(p) {
		return false
	}
	pp := c.prevMeaningful(p)
	return c.punct(pp, ":", ";", "{", "}") || c.kind(pp, token.OpenTag)
}

// isConstantInvocation is TokensAnalyzer::isConstantInvocation() for the
// T_STRING at i.
func (c *clearableTokens) isConstantInvocation(i int) bool {
	next := c.nextMeaningful(i)
	if c.punct(next, "(", "{", "::", "...", `\`) || c.returnRef(next) || c.typeAlternation(next) ||
		c.typeIntersection(next) || c.kind(next, token.Variable) {
		return false
	}
	prev := c.prevMeaningful(i)
	// foreach (FOO as $_) {}
	if c.keyword(next, "as") && !c.punct(prev, "(") {
		return false
	}
	if c.classy(prev) {
		return false
	}
	if c.keyword(prev, "as", "const", "function", "goto") || c.punct(prev, "::") || c.groupOpen(prev) ||
		c.typeColon(prev) || c.typeAlternation(prev) || c.typeIntersection(prev) || c.objectOperator(prev) {
		return false
	}
	if c.keyword(prev, "case") && c.hasEnum {
		sw := c.prevTokenOf(i, func(j int) bool { return c.keyword(j, "switch", "enum") })
		if !c.keyword(sw, "switch") {
			return false
		}
	}
	for c.namespaceOperator(prev) || c.punct(prev, `\`) || c.tString(prev) || c.arrayTypehint(prev) {
		prev = c.prevMeaningful(prev)
	}
	if c.keyword(prev, "extends", "implements", "instanceof", "insteadof", "namespace", "new", "const") ||
		c.isImportUse(prev) || c.isTraitUse(prev) ||
		c.nullableType(prev) || c.typeColon(prev) || c.typeIntersection(prev) || c.typeAlternation(prev) || c.dnfClose(prev) {
		return false
	}
	// FOO & $bar: a by-reference parameter's type, or a bitwise and
	if c.punct(next, "&") && c.kind(c.nextMeaningful(next), token.Variable) {
		check := c.prevTokenOf(prev, func(j int) bool {
			return c.punct(j, ";", "{", "}") || c.keyword(j, "function") || c.kind(j, token.OpenTag)
		})
		if c.keyword(check, "function") {
			return false
		}
	}
	// an extends / implements / use list
	if c.punct(prev, ",") {
		check := prev
		for c.punct(check, ",") || c.keyword(check, "as") || c.namespaceOperator(check) || c.punct(check, `\`) || c.tString(check) {
			check = c.prevMeaningful(check)
		}
		if c.keyword(check, "extends", "implements") || c.groupOpen(check) || c.isImportUse(check) || c.isTraitUse(check) {
			return false
		}
	}
	// an array key inside a double-quoted string: "..$foo[bar].."; blink keeps
	// the literal parts of a split string as String tokens, and the one after a
	// "{$...}" interpolation starts with its closing "}" (CT::T_CURLY_CLOSE)
	if c.punct(prev, "[") && c.punct(next, "]") {
		check := c.nextMeaningful(next)
		if c.kind(check, token.Variable) || c.kind(check, token.String) && !strings.HasPrefix(c.toks[check].Value, "}") {
			return false
		}
	}
	if c.isAttribute(i) {
		return false
	}
	if c.punct(next, ":") && c.belongsToGotoLabel(next) {
		return false
	}
	// a non-capturing catch: catch (Foo)
	for c.namespaceOperator(prev) || c.punct(prev, `\`) || c.tString(prev) || c.typeAlternation(prev) {
		prev = c.prevMeaningful(prev)
	}
	if c.punct(prev, "(") && c.keyword(c.prevMeaningful(prev), "catch") {
		return false
	}
	return true
}
