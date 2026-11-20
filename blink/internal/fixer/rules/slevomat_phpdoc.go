package rules

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// This file ports what slevomat's annotation helpers use of
// phpstan/phpdoc-parser 2.4.1: its Lexer, TokenIterator, PhpDocParser,
// TypeParser and ConstExprParser with the "lines" and "indexes" attributes,
// the node traversal, and its Printer.

type pdTokType uint8

const (
	pdReference pdTokType = iota
	pdUnion
	pdIntersection
	pdNullable
	pdOpenParentheses
	pdCloseParentheses
	pdOpenAngleBracket
	pdCloseAngleBracket
	pdOpenSquareBracket
	pdCloseSquareBracket
	pdComma
	pdVariadic
	pdDoubleColon
	pdDoubleArrow
	pdEqual
	pdOpenPhpdoc
	pdClosePhpdoc
	pdPhpdocTag
	pdDoctrineTag
	pdFloat
	pdInteger
	pdSingleQuotedString
	pdDoubleQuotedString
	pdDoctrineAnnotationString
	pdIdentifier
	pdThisVariable
	pdVariable
	pdHorizontalWS
	pdPhpdocEOL
	pdOther
	pdEnd
	pdColon
	pdWildcard
	pdOpenCurlyBracket
	pdCloseCurlyBracket
	pdNegated
	pdArrow
	pdComment
)

type pdToken struct {
	v    string
	t    pdTokType
	line int
}

var (
	pdFloatRe = regexp.MustCompile(`^(?i)[+\-]?(?:(?:[0-9]+(?:_[0-9]+)*\.[0-9]*(?:_[0-9]+)*(?:e[+\-]?[0-9]+(?:_[0-9]+)*)?)|(?:[0-9]*(?:_[0-9]+)*\.[0-9]+(?:_[0-9]+)*(?:e[+\-]?[0-9]+(?:_[0-9]+)*)?)|(?:[0-9]+(?:_[0-9]+)*e[+\-]?[0-9]+(?:_[0-9]+)*))`)
	pdIntRe   = regexp.MustCompile(`^(?i)[+\-]?(?:(?:0b[0-1]+(?:_[0-1]+)*)|(?:0o[0-7]+(?:_[0-7]+)*)|(?:0x[0-9a-f]+(?:_[0-9a-f]+)*)|(?:[0-9]+(?:_[0-9]+)*))`)
)

func pdIsLetter(c byte) bool { return c|0x20 >= 'a' && c|0x20 <= 'z' }
func pdIsDigit(c byte) bool  { return c >= '0' && c <= '9' }

// pdIsNameStart is [a-z_\x80-\xFF] under the lexer's "i" flag.
func pdIsNameStart(c byte) bool { return pdIsLetter(c) || c == '_' || c >= 0x80 }

func pdIsNameChar(c byte) bool { return pdIsNameStart(c) || pdIsDigit(c) }

// pdIsSpace is PCRE's \s.
func pdIsSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// pdLex is Lexer::tokenize: the first pattern matching at the current offset
// wins, and lexing stops where none does.
func pdLex(s string) []pdToken {
	var out []pdToken
	line := 1
	for p := 0; p < len(s); {
		n, t := pdLexOne(s, p)
		if n == 0 {
			break
		}
		out = append(out, pdToken{s[p : p+n], t, line})
		if t == pdPhpdocEOL {
			line++
		}
		p += n
	}
	if len(out) == 0 {
		return []pdToken{{"", pdEnd, 1}}
	}
	return append(out, pdToken{"", pdEnd, line})
}

func pdLexOne(s string, p int) (int, pdTokType) {
	at := func(i int) byte {
		if i < len(s) {
			return s[i]
		}
		return 0
	}
	c := s[p]
	if c == ' ' || c == '\t' {
		e := p
		for e < len(s) && (s[e] == ' ' || s[e] == '\t') {
			e++
		}
		return e - p, pdHorizontalWS
	}
	if n := pdLexIdentifier(s, p); n > 0 {
		return n, pdIdentifier
	}
	if c == '$' {
		if len(s)-p >= 5 && strings.EqualFold(s[p+1:p+5], "this") && !pdIsNameChar(at(p+5)) {
			return 5, pdThisVariable
		}
		if pdIsNameStart(at(p + 1)) {
			e := p + 2
			for e < len(s) && pdIsNameChar(s[e]) {
				e++
			}
			return e - p, pdVariable
		}
	}
	if c == '&' {
		if pdIsReference(s, p+1) {
			return 1, pdReference
		}
		return 1, pdIntersection
	}
	switch c {
	case '|':
		return 1, pdUnion
	case '?':
		return 1, pdNullable
	case '!':
		return 1, pdNegated
	case '(':
		return 1, pdOpenParentheses
	case ')':
		return 1, pdCloseParentheses
	case '<':
		return 1, pdOpenAngleBracket
	case '>':
		return 1, pdCloseAngleBracket
	case '[':
		return 1, pdOpenSquareBracket
	case ']':
		return 1, pdCloseSquareBracket
	case '{':
		return 1, pdOpenCurlyBracket
	case '}':
		return 1, pdCloseCurlyBracket
	case ',':
		return 1, pdComma
	}
	if c == '/' && at(p+1) == '/' {
		if n := pdLexComment(s, p); n > 0 {
			return n, pdComment
		}
	}
	if strings.HasPrefix(s[p:], "...") {
		return 3, pdVariadic
	}
	if strings.HasPrefix(s[p:], "::") {
		return 2, pdDoubleColon
	}
	if strings.HasPrefix(s[p:], "=>") {
		return 2, pdDoubleArrow
	}
	if strings.HasPrefix(s[p:], "->") {
		return 2, pdArrow
	}
	if c == '=' {
		return 1, pdEqual
	}
	if c == ':' {
		return 1, pdColon
	}
	if strings.HasPrefix(s[p:], "/**") && pdIsSpace(at(p+3)) {
		if at(p+3) == ' ' {
			return 4, pdOpenPhpdoc
		}
		return 3, pdOpenPhpdoc
	}
	if strings.HasPrefix(s[p:], "*/") {
		return 2, pdClosePhpdoc
	}
	if c == '@' {
		if n := pdLexTag(s, p); n > 0 {
			return n, pdPhpdocTag
		}
		if n := pdLexDoctrineTag(s, p); n > 0 {
			return n, pdDoctrineTag
		}
	}
	if c == '\r' || c == '\n' {
		e := p
		if c == '\r' {
			e++
		}
		if at(e) == '\n' {
			e++
			for e < len(s) && (s[e] == ' ' || s[e] == '\t') {
				e++
			}
			if at(e) == '*' && at(e+1) != '/' {
				e++
				if at(e) == ' ' {
					e++
				}
			}
			return e - p, pdPhpdocEOL
		}
	}
	if m := pdFloatRe.FindString(s[p:]); m != "" {
		return len(m), pdFloat
	}
	if m := pdIntRe.FindString(s[p:]); m != "" {
		return len(m), pdInteger
	}
	if c == '\'' || c == '"' {
		if n := pdLexQuoted(s, p, c); n > 0 {
			if c == '\'' {
				return n, pdSingleQuotedString
			}
			return n, pdDoubleQuotedString
		}
	}
	if c == '"' {
		e := p + 1
		for e < len(s) {
			if strings.HasPrefix(s[e:], `""`) {
				e += 2
			} else if s[e] != '"' {
				e++
			} else {
				break
			}
		}
		if at(e) == '"' {
			return e + 1 - p, pdDoctrineAnnotationString
		}
	}
	if c == '*' {
		return 1, pdWildcard
	}
	e := p
	for e < len(s) && !pdIsSpace(s[e]) && !strings.HasPrefix(s[e:], "*/") {
		e++
	}
	return e - p, pdOther
}

// pdLexIdentifier is (?:[\\]?+[a-z_\x80-\xFF][0-9a-z_\x80-\xFF-]*+)++.
func pdLexIdentifier(s string, p int) int {
	e := p
	for {
		q := e
		if q < len(s) && s[q] == '\\' {
			q++
		}
		if q >= len(s) || !pdIsNameStart(s[q]) {
			return e - p
		}
		q++
		for q < len(s) && (pdIsNameChar(s[q]) || s[q] == '-') {
			q++
		}
		e = q
	}
}

// pdIsReference is the lookahead of "&" as TOKEN_REFERENCE.
func pdIsReference(s string, p int) bool {
	for p < len(s) && pdIsSpace(s[p]) {
		p++
	}
	if p >= len(s) {
		return false
	}
	switch {
	case s[p] == ',' || s[p] == '=' || s[p] == ')':
		return true
	case strings.HasPrefix(s[p:], "..."):
		return true
	case s[p] == '$':
		if len(s)-p >= 5 && strings.EqualFold(s[p+1:p+5], "this") && (p+5 >= len(s) || !pdIsNameChar(s[p+5])) {
			return false
		}
		return true
	}
	return false
}

// pdLexComment is \/\/[^\r\n]*(?=\n|\r|\*/).
func pdLexComment(s string, p int) int {
	e := p + 2
	for e < len(s) && s[e] != '\r' && s[e] != '\n' {
		e++
	}
	if e < len(s) {
		return e - p
	}
	for k := e - 2; k >= p+2; k-- {
		if strings.HasPrefix(s[k:], "*/") {
			return k - p
		}
	}
	return 0
}

// pdLexTag is @(?:[a-z][a-z0-9-\\]+:)?[a-z][a-z0-9_\\-]*+.
func pdLexTag(s string, p int) int {
	tail := func(q int) int {
		if q >= len(s) || !pdIsLetter(s[q]) {
			return 0
		}
		q++
		for q < len(s) && (pdIsLetter(s[q]) || pdIsDigit(s[q]) || s[q] == '_' || s[q] == '\\' || s[q] == '-') {
			q++
		}
		return q
	}
	if p+1 < len(s) && pdIsLetter(s[p+1]) {
		j := p + 2
		for j < len(s) && (pdIsLetter(s[j]) || pdIsDigit(s[j]) || s[j] == '-' || s[j] == '\\') {
			j++
		}
		if j > p+2 && j < len(s) && s[j] == ':' {
			if e := tail(j + 1); e > 0 {
				return e - p
			}
		}
	}
	if e := tail(p + 1); e > 0 {
		return e - p
	}
	return 0
}

// pdLexDoctrineTag is @[a-z_\\][a-z0-9_\:\\]*[a-z_][a-z0-9_]*.
func pdLexDoctrineTag(s string, p int) int {
	if p+1 >= len(s) || !pdIsLetter(s[p+1]) && s[p+1] != '_' && s[p+1] != '\\' {
		return 0
	}
	j := p + 2
	for j < len(s) && (pdIsLetter(s[j]) || pdIsDigit(s[j]) || s[j] == '_' || s[j] == ':' || s[j] == '\\') {
		j++
	}
	for k := j - 1; k >= p+2; k-- {
		if pdIsLetter(s[k]) || s[k] == '_' {
			e := k + 1
			for e < len(s) && (pdIsLetter(s[e]) || pdIsDigit(s[e]) || s[e] == '_') {
				e++
			}
			return e - p
		}
	}
	return 0
}

// pdLexQuoted is a single or double quoted string on one line.
func pdLexQuoted(s string, p int, q byte) int {
	e := p + 1
	for e < len(s) {
		if s[e] == '\\' && e+1 < len(s) && s[e+1] != '\r' && s[e+1] != '\n' {
			e += 2
			continue
		}
		if s[e] != q && s[e] != '\r' && s[e] != '\n' && s[e] != '\\' {
			e++
			continue
		}
		break
	}
	if e < len(s) && s[e] == q {
		return e + 1 - p
	}
	return 0
}

// pdError is a ParserException.
type pdError struct{}

// pdIter is TokenIterator; comments only need skipping.
type pdIter struct {
	toks    []pdToken
	i       int
	saves   []int
	skipEOL bool
	newline string
}

func newPdIter(toks []pdToken) *pdIter {
	it := &pdIter{toks: toks}
	it.skipIrrelevant()
	return it
}

func (it *pdIter) skipped(t pdTokType) bool {
	return t == pdHorizontalWS || it.skipEOL && t == pdPhpdocEOL
}

func (it *pdIter) skipIrrelevant() {
	if it.i >= len(it.toks) {
		return
	}
	for it.skipped(it.toks[it.i].t) {
		if it.i+1 >= len(it.toks) {
			break
		}
		it.i++
	}
}

func (it *pdIter) next() {
	it.i++
	it.skipIrrelevant()
}

func (it *pdIter) cur() pdToken { return it.toks[it.i] }

func (it *pdIter) is(types ...pdTokType) bool {
	t := it.toks[it.i].t
	return slices.Contains(types, t)
}

func (it *pdIter) isValue(v string) bool { return it.toks[it.i].v == v }

func (it *pdIter) consume(t pdTokType) {
	if it.toks[it.i].t != t {
		panic(pdError{})
	}
	if t == pdPhpdocEOL && it.newline == "" {
		it.detectNewline()
	}
	it.next()
}

func (it *pdIter) consumeValue(t pdTokType, v string) {
	if it.toks[it.i].t != t || it.toks[it.i].v != v {
		panic(pdError{})
	}
	it.next()
}

func (it *pdIter) tryConsume(t pdTokType) bool {
	if it.toks[it.i].t != t {
		return false
	}
	if t == pdPhpdocEOL && it.newline == "" {
		it.detectNewline()
	}
	it.next()
	return true
}

func (it *pdIter) tryConsumeValue(v string) bool {
	if it.toks[it.i].v != v {
		return false
	}
	it.next()
	return true
}

func (it *pdIter) detectNewline() {
	v := it.toks[it.i].v
	if strings.HasPrefix(v, "\r\n") {
		it.newline = "\r\n"
	} else if strings.HasPrefix(v, "\n") {
		it.newline = "\n"
	}
}

func (it *pdIter) nl() string {
	if it.newline == "" {
		return "\n"
	}
	return it.newline
}

func (it *pdIter) skipNewLinesAndComments() {
	if it.cur().t == pdComment {
		it.next()
	}
	if !it.is(pdPhpdocEOL) {
		return
	}
	for {
		found := it.tryConsume(pdPhpdocEOL)
		if it.cur().t == pdComment {
			it.next()
		}
		if !found {
			return
		}
	}
}

func (it *pdIter) precededByWS() bool {
	return it.i > 0 && it.toks[it.i-1].t == pdHorizontalWS
}

func (it *pdIter) skippedWS() string {
	if it.i > 0 && it.toks[it.i-1].t == pdHorizontalWS {
		return it.toks[it.i-1].v
	}
	return ""
}

func (it *pdIter) joinUntil(types ...pdTokType) string {
	var b strings.Builder
	for !it.is(types...) {
		b.WriteString(it.toks[it.i].v)
		it.i++
	}
	return b.String()
}

func (it *pdIter) endIndexOfLastRelevant() int {
	e := it.i - 1
	for e >= 0 && it.skipped(it.toks[e].t) {
		if e-1 < 0 {
			break
		}
		e--
	}
	return e
}

func (it *pdIter) push()     { it.saves = append(it.saves, it.i) }
func (it *pdIter) drop()     { it.saves = it.saves[:len(it.saves)-1] }
func (it *pdIter) rollback() { it.i = it.saves[len(it.saves)-1]; it.drop() }

// pdTry runs fn, reporting false on a ParserException.
func pdTry(fn func()) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, is := r.(pdError); !is {
				panic(r)
			}
			ok = false
		}
	}()
	fn()
	return true
}

// pdNode is a phpdoc-parser AST node. kind is its class name and fields its
// node-valued properties in declaration order, which is the order
// NodeTraverser and the printer walk them in.
type pdNode struct {
	kind       string
	start, end int
	fields     []pdField
	// name is IdentifierTypeNode::$name, ConstFetchNode::$name, PhpDocTagNode::$name,
	// TemplateTagValueNode::$name, TypeAliasTagValueNode::$alias and
	// TypeAliasImportTagValueNode::$importedAlias.
	name string
	// className is ConstFetchNode::$className, importedAs TypeAliasImportTagValueNode::$importedAs.
	className, importedAs string
	// text is what a ConstExprNode prints as, ArrayShapeNode::$kind,
	// CallableTypeParameterNode::$parameterName and ConditionalTypeForParameterNode::$parameterName.
	text      string
	flags     pdFlags
	variances []string // GenericTypeNode::$variances
	// orig is the Attribute::ORIGINAL_NODE of a clone.
	orig *pdNode
}

type pdFlags uint8

const (
	pdFlagNegated pdFlags = 1 << iota
	pdFlagOptional
	pdFlagReference
	pdFlagVariadic
	pdFlagUnsealed
)

func (n *pdNode) has(f pdFlags) bool { return n.flags&f != 0 }

type pdField struct {
	node   *pdNode
	list   []*pdNode
	isList bool
}

func pdNew(kind string, fields ...pdField) *pdNode {
	return &pdNode{kind: kind, start: -1, end: -1, fields: fields}
}

func pdF(n *pdNode) pdField       { return pdField{node: n} }
func pdL(l []*pdNode) pdField     { return pdField{list: l, isList: true} }
func pdIdent(name string) *pdNode { n := pdNew("IdentifierTypeNode"); n.name = name; return n }

// value is the tag value of a PhpDocTagNode.
func (n *pdNode) value() *pdNode { return n.fields[0].node }

type pdParser struct {
	it *pdIter
}

func (p *pdParser) enrich(n *pdNode, start int) *pdNode {
	n.start = start
	n.end = p.it.endIndexOfLastRelevant()
	return n
}

// pdParseWithNewline is PhpDocParser::parse of a doc comment, or nil when it
// throws, with the newline its TokenIterator detects.
func pdParseWithNewline(doc string) (root *pdNode, toks []pdToken, newline string) {
	toks = pdLex(doc)
	p := &pdParser{it: newPdIter(toks)}
	if !pdTry(func() { root = p.parse() }) {
		return nil, toks, p.it.newline
	}
	return root, toks, p.it.newline
}

func (p *pdParser) parse() *pdNode {
	it := p.it
	it.consume(pdOpenPhpdoc)
	it.tryConsume(pdPhpdocEOL)
	var children []*pdNode
	if !it.is(pdClosePhpdoc) {
		last := p.parseChild()
		children = append(children, last)
		for !it.is(pdClosePhpdoc) {
			if last.kind == "PhpDocTagNode" && (last.value().kind == "DoctrineTagValueNode" || last.value().kind == "GenericTagValueNode") {
				it.tryConsume(pdPhpdocEOL)
				if it.is(pdClosePhpdoc) {
					break
				}
				last = p.parseChild()
				children = append(children, last)
				continue
			}
			if !it.tryConsume(pdPhpdocEOL) {
				break
			}
			if it.is(pdClosePhpdoc) {
				break
			}
			last = p.parseChild()
			children = append(children, last)
		}
	}
	if !pdTry(func() { it.consume(pdClosePhpdoc) }) {
		name := ""
		start := it.i
		if len(children) > 0 && children[len(children)-1].kind == "PhpDocTagNode" {
			name = children[len(children)-1].name
		}
		tag := pdNew("PhpDocTagNode", pdF(p.enrich(pdNew("InvalidTagValueNode"), start)))
		tag.name = name
		it.i = len(it.toks) - 1
		return p.enrich(pdNew("PhpDocNode", pdL([]*pdNode{p.enrich(tag, start)})), 0)
	}
	return p.enrich(pdNew("PhpDocNode", pdL(children)), 0)
}

func (p *pdParser) parseChild() *pdNode {
	it := p.it
	start := it.i
	if it.is(pdPhpdocTag) {
		return p.enrich(p.parseTag(), start)
	}
	if it.is(pdDoctrineTag) {
		tag := it.cur().v
		it.next()
		tagStart := it.i
		var value *pdNode
		it.push()
		if pdTry(func() { value = p.parseDoctrineTagValue(tag) }) {
			it.drop()
		} else {
			it.rollback()
			p.parseOptionalDescription(false)
			value = pdNew("InvalidTagValueNode")
		}
		n := pdNew("PhpDocTagNode", pdF(p.enrich(value, tagStart)))
		n.name = tag
		return p.enrich(n, start)
	}
	p.parseText()
	return p.enrich(pdNew("PhpDocTextNode"), start)
}

func (p *pdParser) parseText() string {
	it := p.it
	text := ""
	saved := false
	for {
		tmp := it.skippedWS() + it.joinUntil(pdPhpdocEOL, pdClosePhpdoc, pdEnd)
		text += tmp
		if !it.is(pdPhpdocEOL, pdClosePhpdoc) {
			break
		}
		if !saved {
			it.push()
			saved = true
		} else if tmp != "" {
			it.drop()
			it.push()
		}
		it.push()
		it.next()
		if it.is(pdPhpdocTag, pdDoctrineTag, pdClosePhpdoc, pdEnd) {
			it.rollback()
			break
		}
		it.drop()
		text += it.nl()
	}
	if saved {
		it.rollback()
		text = strings.TrimRight(text, it.nl())
	}
	return strings.Trim(text, " \t")
}

func (p *pdParser) parseOptionalDescriptionAfterDoctrineTag() string {
	it := p.it
	text := ""
	saved := false
	for {
		tmp := it.skippedWS() + it.joinUntil(pdPhpdocTag, pdDoctrineTag, pdPhpdocEOL, pdClosePhpdoc, pdEnd)
		text += tmp
		if !it.is(pdPhpdocEOL, pdClosePhpdoc) {
			if !it.precededByWS() {
				return strings.Trim(text+p.parseText(), " \t")
			}
			if it.is(pdPhpdocTag) {
				it.push()
				child := p.parseChild()
				if child.kind == "PhpDocTagNode" {
					switch child.value().kind {
					case "GenericTagValueNode", "DoctrineTagValueNode":
						it.rollback()
						goto done
					case "InvalidTagValueNode":
						it.rollback()
						it.push()
						it.next()
						if it.is(pdOpenParentheses) {
							it.rollback()
							goto done
						}
						it.rollback()
						return strings.Trim(text+p.parseText(), " \t")
					}
				}
				it.rollback()
				return strings.Trim(text+p.parseText(), " \t")
			}
			break
		}
		if !saved {
			it.push()
			saved = true
		} else if tmp != "" {
			it.drop()
			it.push()
		}
		it.push()
		it.next()
		if it.is(pdPhpdocTag, pdDoctrineTag, pdClosePhpdoc, pdEnd) {
			it.rollback()
			break
		}
		it.drop()
		text += it.nl()
	}
done:
	if saved {
		it.rollback()
		text = strings.TrimRight(text, it.nl())
	}
	return strings.Trim(text, " \t")
}

func (p *pdParser) parseTag() *pdNode {
	tag := p.it.cur().v
	p.it.next()
	n := pdNew("PhpDocTagNode", pdF(p.parseTagValue(tag)))
	n.name = tag
	return n
}

func (p *pdParser) parseTagValue(tag string) *pdNode {
	it := p.it
	start := it.i
	var value *pdNode
	it.push()
	if pdTry(func() { value = p.parseTagValueBody(tag) }) {
		it.drop()
	} else {
		it.rollback()
		p.parseOptionalDescription(false)
		value = pdNew("InvalidTagValueNode")
	}
	return p.enrich(value, start)
}

func (p *pdParser) parseTagValueBody(tag string) *pdNode {
	it := p.it
	typed := func(kind string, limit bool) *pdNode {
		t := p.parseType()
		p.parseOptionalDescription(limit)
		return pdNew(kind, pdF(t))
	}
	switch tag {
	case "@param", "@phpstan-param", "@psalm-param", "@phan-param":
		var t *pdNode
		if !it.is(pdReference, pdVariadic, pdVariable) {
			t = p.parseType()
		}
		it.tryConsume(pdReference)
		it.tryConsume(pdVariadic)
		name := it.cur().v
		it.consume(pdVariable)
		p.parseOptionalDescription(false)
		if t != nil {
			n := pdNew("ParamTagValueNode", pdF(t))
			n.text = name
			return n
		}
		return pdNew("TypelessParamTagValueNode")
	case "@param-immediately-invoked-callable", "@phpstan-param-immediately-invoked-callable",
		"@param-later-invoked-callable", "@phpstan-param-later-invoked-callable",
		"@pure-unless-callable-is-impure", "@phpstan-pure-unless-callable-is-impure",
		"@pure-unless-parameter-passed", "@phpstan-pure-unless-parameter-passed":
		it.consume(pdVariable)
		p.parseOptionalDescription(false)
		return pdNew("ParamCallableTagValueNode")
	case "@param-closure-this", "@phpstan-param-closure-this", "@param-out", "@phpstan-param-out", "@psalm-param-out",
		"@property", "@property-read", "@property-write", "@phpstan-property", "@phpstan-property-read",
		"@phpstan-property-write", "@psalm-property", "@psalm-property-read", "@psalm-property-write",
		"@phan-property", "@phan-property-read", "@phan-property-write":
		t := p.parseType()
		it.consume(pdVariable)
		p.parseOptionalDescription(false)
		return pdNew("TypedVariableTagValueNode", pdF(t))
	case "@var", "@phpstan-var", "@psalm-var", "@phan-var":
		t := p.parseType()
		hasName := false
		if it.is(pdVariable, pdThisVariable) {
			hasName = true
			it.next()
		}
		p.parseOptionalDescription(!hasName)
		return pdNew("VarTagValueNode", pdF(t))
	case "@return", "@phpstan-return", "@psalm-return", "@phan-return", "@phan-real-return":
		return typed("ReturnTagValueNode", true)
	case "@throws", "@phpstan-throws":
		return typed("ThrowsTagValueNode", true)
	case "@mixin", "@phan-mixin":
		return typed("MixinTagValueNode", true)
	case "@psalm-require-extends", "@phpstan-require-extends":
		return typed("RequireExtendsTagValueNode", true)
	case "@psalm-require-implements", "@phpstan-require-implements":
		return typed("RequireImplementsTagValueNode", true)
	case "@psalm-inheritors", "@phpstan-sealed":
		return typed("SealedTagValueNode", true)
	case "@phpstan-this-out", "@phpstan-self-out", "@psalm-this-out", "@psalm-self-out":
		return typed("SelfOutTagValueNode", true)
	case "@deprecated":
		p.parseOptionalDescription(false)
		return pdNew("DeprecatedTagValueNode")
	case "@method", "@phpstan-method", "@psalm-method", "@phan-method":
		return p.parseMethodTagValue()
	case "@template", "@phpstan-template", "@psalm-template", "@phan-template", "@template-covariant",
		"@phpstan-template-covariant", "@psalm-template-covariant", "@template-contravariant",
		"@phpstan-template-contravariant", "@psalm-template-contravariant":
		return p.parseTemplateTagValue(true)
	case "@extends", "@phpstan-extends", "@phan-extends", "@phan-inherits", "@template-extends",
		"@implements", "@phpstan-implements", "@template-implements",
		"@use", "@phpstan-use", "@template-use":
		start := it.i
		base := pdIdent(it.cur().v)
		it.consume(pdIdentifier)
		t := p.parseGeneric(p.enrich(base, start))
		p.parseOptionalDescription(true)
		return pdNew("ExtendsTagValueNode", pdF(t))
	case "@phpstan-type", "@psalm-type", "@phan-type":
		alias := it.cur().v
		it.consume(pdIdentifier)
		it.tryConsume(pdEqual)
		start := it.i
		var t *pdNode
		if !pdTry(func() {
			t = p.parseType()
			if !it.is(pdClosePhpdoc) && !it.is(pdPhpdocEOL) {
				panic(pdError{})
			}
		}) {
			p.parseOptionalDescription(false)
			t = p.enrich(pdNew("InvalidTypeNode"), start)
		}
		n := pdNew("TypeAliasTagValueNode", pdF(t))
		n.name = alias
		return n
	case "@phpstan-import-type", "@psalm-import-type":
		imported := it.cur().v
		it.consume(pdIdentifier)
		it.consumeValue(pdIdentifier, "from")
		start := it.i
		from := pdIdent(it.cur().v)
		it.consume(pdIdentifier)
		p.enrich(from, start)
		n := pdNew("TypeAliasImportTagValueNode", pdF(from))
		n.name = imported
		if it.tryConsumeValue("as") {
			n.importedAs = it.cur().v
			it.consume(pdIdentifier)
		}
		return n
	case "@phpstan-assert", "@phpstan-assert-if-true", "@phpstan-assert-if-false", "@psalm-assert",
		"@psalm-assert-if-true", "@psalm-assert-if-false", "@phan-assert", "@phan-assert-if-true", "@phan-assert-if-false":
		it.tryConsume(pdNegated)
		it.tryConsume(pdEqual)
		t := p.parseType()
		if it.is(pdThisVariable) {
			it.next()
		} else {
			it.consume(pdVariable)
		}
		if it.is(pdArrow) {
			it.consume(pdArrow)
			it.consume(pdIdentifier)
			if it.tryConsume(pdOpenParentheses) {
				it.consume(pdCloseParentheses)
			}
		}
		p.parseOptionalDescription(false)
		return pdNew("AssertTagValueNode", pdF(t))
	}
	if it.is(pdOpenParentheses) {
		return p.parseDoctrineTagValue(tag)
	}
	p.parseOptionalDescriptionAfterDoctrineTag()
	return pdNew("GenericTagValueNode")
}

func (p *pdParser) parseDoctrineTagValue(tag string) *pdNode {
	start := p.it.i
	annotation := pdNew("DoctrineAnnotation", pdL(p.parseDoctrineArguments(false)))
	annotation.name = tag
	p.enrich(annotation, start)
	p.parseOptionalDescriptionAfterDoctrineTag()
	return pdNew("DoctrineTagValueNode", pdF(annotation))
}

func (p *pdParser) parseDoctrineArguments(deep bool) []*pdNode {
	it := p.it
	if !it.is(pdOpenParentheses) {
		return nil
	}
	if !deep {
		it.skipEOL = true
	}
	var args []*pdNode
	ok := pdTry(func() {
		it.consume(pdOpenParentheses)
		for !it.is(pdCloseParentheses) {
			args = append(args, p.parseDoctrineArgument())
			if !it.tryConsume(pdComma) {
				break
			}
		}
	})
	if !deep {
		it.skipEOL = false
	}
	if !ok {
		panic(pdError{})
	}
	it.consume(pdCloseParentheses)
	return args
}

func (p *pdParser) parseDoctrineArgument() *pdNode {
	it := p.it
	start := it.i
	if !it.is(pdIdentifier, pdDoubleQuotedString, pdDoctrineAnnotationString) {
		return p.enrich(pdNew("DoctrineArgument", pdF(nil), pdF(p.parseDoctrineArgumentValue())), start)
	}
	var n *pdNode
	it.push()
	if pdTry(func() {
		isIdentifier := it.is(pdIdentifier)
		value := it.cur().v
		it.next()
		if !isIdentifier {
			p.parseDoctrineString()
		}
		key := p.enrich(pdIdent(value), start)
		it.consume(pdEqual)
		v := p.parseDoctrineArgumentValue()
		n = pdNew("DoctrineArgument", pdF(key), pdF(v))
	}) {
		it.drop()
		return p.enrich(n, start)
	}
	it.rollback()
	return p.enrich(pdNew("DoctrineArgument", pdF(nil), pdF(p.parseDoctrineArgumentValue())), start)
}

func (p *pdParser) parseDoctrineArgumentValue() *pdNode {
	it := p.it
	start := it.i
	if it.is(pdPhpdocTag, pdDoctrineTag) {
		name := it.cur().v
		it.next()
		n := pdNew("DoctrineAnnotation", pdL(p.parseDoctrineArguments(true)))
		n.name = name
		return p.enrich(n, start)
	}
	if it.tryConsume(pdOpenCurlyBracket) {
		var items []*pdNode
		for !it.is(pdCloseCurlyBracket) {
			items = append(items, p.parseDoctrineArrayItem())
			if !it.tryConsume(pdComma) {
				break
			}
		}
		it.consume(pdCloseCurlyBracket)
		return p.enrich(pdNew("DoctrineArray", pdL(items)), start)
	}
	value := it.cur().v
	it.push()
	if it.tryConsume(pdIdentifier) {
		identifier := p.enrich(pdIdent(value), start)
		if !it.is(pdDoubleColon) {
			it.drop()
			return identifier
		}
		it.rollback()
	} else {
		it.drop()
	}
	expr := p.parseConstExpr(true)
	if expr.kind == "ConstExprArrayNode" {
		panic(pdError{})
	}
	return expr
}

func (p *pdParser) parseDoctrineArrayItem() *pdNode {
	it := p.it
	start := it.i
	var n *pdNode
	it.push()
	if pdTry(func() {
		key := p.parseDoctrineArrayKey()
		if !it.tryConsume(pdEqual) && !it.tryConsume(pdColon) {
			it.consume(pdEqual)
		}
		n = pdNew("DoctrineArrayItem", pdF(key), pdF(p.parseDoctrineArgumentValue()))
	}) {
		it.drop()
		return p.enrich(n, start)
	}
	it.rollback()
	return p.enrich(pdNew("DoctrineArrayItem", pdF(nil), pdF(p.parseDoctrineArgumentValue())), start)
}

func (p *pdParser) parseDoctrineArrayKey() *pdNode {
	it := p.it
	start := it.i
	var key *pdNode
	switch {
	case it.is(pdInteger):
		key = pdNew("ConstExprIntegerNode")
		it.next()
	case it.is(pdDoctrineAnnotationString):
		it.next()
		p.parseDoctrineString()
		key = pdNew("DoctrineConstExprStringNode")
	case it.is(pdSingleQuotedString):
		key = pdNew("ConstExprStringNode")
		it.next()
	case it.is(pdDoubleQuotedString):
		it.next()
		p.parseDoctrineString()
		key = pdNew("DoctrineConstExprStringNode")
	default:
		value := it.cur().v
		it.push()
		if !it.tryConsume(pdIdentifier) {
			it.drop()
			panic(pdError{})
		}
		if !it.is(pdDoubleColon) {
			it.drop()
			return p.enrich(pdIdent(value), start)
		}
		it.rollback()
		expr := p.parseConstExpr(true)
		if expr.kind != "ConstFetchNode" {
			panic(pdError{})
		}
		return expr
	}
	return p.enrich(key, start)
}

func (p *pdParser) parseDoctrineString() {
	for p.it.is(pdDoubleQuotedString, pdDoctrineAnnotationString) {
		p.it.next()
	}
}

func (p *pdParser) parseMethodTagValue() *pdNode {
	it := p.it
	first := p.parseType()
	isStatic := false
	second := first
	if first.kind == "IdentifierTypeNode" && first.name == "static" {
		isStatic = true
		second = p.parseType()
	}
	var returnType *pdNode
	if it.is(pdIdentifier) {
		returnType = second
		it.next()
	} else if second.kind == "IdentifierTypeNode" {
		if isStatic {
			returnType = first
		}
	} else {
		panic(pdError{})
	}
	var templates []*pdNode
	if it.tryConsume(pdOpenAngleBracket) {
		for {
			start := it.i
			templates = append(templates, p.enrich(p.parseTemplateTagValue(false), start))
			if !it.tryConsume(pdComma) {
				break
			}
		}
		it.consume(pdCloseAngleBracket)
	}
	var params []*pdNode
	it.consume(pdOpenParentheses)
	if !it.is(pdCloseParentheses) {
		params = append(params, p.parseMethodTagValueParameter())
		for it.tryConsume(pdComma) {
			params = append(params, p.parseMethodTagValueParameter())
		}
	}
	it.consume(pdCloseParentheses)
	p.parseOptionalDescription(false)
	return pdNew("MethodTagValueNode", pdF(returnType), pdL(templates), pdL(params))
}

func (p *pdParser) parseMethodTagValueParameter() *pdNode {
	it := p.it
	start := it.i
	var t, def *pdNode
	if it.is(pdIdentifier, pdOpenParentheses, pdNullable) {
		t = p.parseType()
	}
	it.tryConsume(pdReference)
	it.tryConsume(pdVariadic)
	it.consume(pdVariable)
	if it.tryConsume(pdEqual) {
		def = p.parseConstExpr(false)
	}
	return p.enrich(pdNew("MethodTagValueParameterNode", pdF(t), pdF(def)), start)
}

func (p *pdParser) parseOptionalDescription(limitStartToken bool) {
	it := p.it
	if limitStartToken {
		if it.is(pdUnion) || it.is(pdIntersection) {
			it.consume(pdOther)
		}
		if !it.is(pdPhpdocEOL, pdClosePhpdoc, pdEnd) && !it.precededByWS() {
			it.consume(pdHorizontalWS)
		}
	}
	p.parseText()
}

// TypeParser

func (p *pdParser) parseType() *pdNode {
	it := p.it
	start := it.i
	var t *pdNode
	if it.is(pdNullable) {
		t = p.parseNullable()
	} else {
		t = p.parseAtomic()
		it.push()
		it.skipNewLinesAndComments()
		var enriched *pdNode
		if !pdTry(func() { enriched = p.unionOrIntersection(t) }) {
			enriched = nil
		}
		if enriched != nil {
			t = enriched
			it.drop()
		} else {
			it.rollback()
			if e := p.unionOrIntersection(t); e != nil {
				t = e
			}
		}
	}
	return p.enrich(t, start)
}

func (p *pdParser) unionOrIntersection(t *pdNode) *pdNode {
	if p.it.is(pdUnion) {
		return p.parseList(t, pdUnion, "UnionTypeNode")
	}
	if p.it.is(pdIntersection) {
		return p.parseList(t, pdIntersection, "IntersectionTypeNode")
	}
	return nil
}

// parseList is parseUnion and parseIntersection.
func (p *pdParser) parseList(t *pdNode, sep pdTokType, kind string) *pdNode {
	it := p.it
	types := []*pdNode{t}
	for it.tryConsume(sep) {
		types = append(types, p.parseAtomic())
		it.push()
		it.skipNewLinesAndComments()
		if !it.is(sep) {
			it.rollback()
			break
		}
		it.drop()
	}
	return pdNew(kind, pdL(types))
}

// subParseList is subParseUnion and subParseIntersection.
func (p *pdParser) subParseList(t *pdNode, sep pdTokType, kind string) *pdNode {
	it := p.it
	types := []*pdNode{t}
	for it.tryConsume(sep) {
		it.skipNewLinesAndComments()
		types = append(types, p.parseAtomic())
		it.skipNewLinesAndComments()
	}
	return pdNew(kind, pdL(types))
}

func (p *pdParser) subParse() *pdNode {
	it := p.it
	start := it.i
	var t *pdNode
	switch {
	case it.is(pdNullable):
		t = p.parseNullable()
	case it.is(pdVariable):
		t = p.parseConditional(nil)
	default:
		t = p.parseAtomic()
		if it.isValue("is") {
			t = p.parseConditional(t)
		} else {
			it.skipNewLinesAndComments()
			if it.is(pdUnion) {
				t = p.subParseList(t, pdUnion, "UnionTypeNode")
			} else if it.is(pdIntersection) {
				t = p.subParseList(t, pdIntersection, "IntersectionTypeNode")
			}
		}
	}
	return p.enrich(t, start)
}

func (p *pdParser) parseAtomic() *pdNode {
	it := p.it
	start := it.i
	if it.tryConsume(pdOpenParentheses) {
		it.skipNewLinesAndComments()
		t := p.subParse()
		it.skipNewLinesAndComments()
		it.consume(pdCloseParentheses)
		if it.is(pdOpenSquareBracket) {
			t = p.tryParseArrayOrOffsetAccess(t)
		}
		return p.enrich(t, start)
	}
	if it.tryConsume(pdThisVariable) {
		t := p.enrich(pdNew("ThisTypeNode"), start)
		if it.is(pdOpenSquareBracket) {
			t = p.tryParseArrayOrOffsetAccess(t)
		}
		return p.enrich(t, start)
	}
	value := it.cur().v
	it.push()
	if it.tryConsume(pdIdentifier) {
		t := p.enrich(pdIdent(value), start)
		if !it.is(pdDoubleColon) {
			it.drop()
			switch {
			case it.is(pdOpenAngleBracket):
				it.push()
				isHTML := p.isHTML()
				it.rollback()
				if isHTML {
					return t
				}
				orig := t
				t = p.tryParseCallable(t, true)
				if t == orig {
					t = p.parseGeneric(t)
					if it.is(pdOpenSquareBracket) {
						t = p.tryParseArrayOrOffsetAccess(t)
					}
				}
			case it.is(pdOpenParentheses):
				t = p.tryParseCallable(t, false)
			case it.is(pdOpenSquareBracket):
				t = p.tryParseArrayOrOffsetAccess(t)
			case pdIsShapeKind(t.name) && it.is(pdOpenCurlyBracket) && !it.precededByWS():
				if t.name == "object" {
					t = p.parseObjectShape()
				} else {
					t = p.parseArrayShape(t.name)
				}
				if it.is(pdOpenSquareBracket) {
					t = p.tryParseArrayOrOffsetAccess(p.enrich(t, start))
				}
			}
			return p.enrich(t, start)
		}
		it.rollback()
	} else {
		it.drop()
	}
	expr := p.parseConstExpr(false)
	if expr.kind == "ConstExprArrayNode" {
		panic(pdError{})
	}
	t := p.enrich(pdNew("ConstTypeNode", pdF(expr)), start)
	if it.is(pdOpenSquareBracket) {
		t = p.tryParseArrayOrOffsetAccess(t)
	}
	return t
}

func pdIsShapeKind(name string) bool {
	switch name {
	case "array", "list", "non-empty-array", "non-empty-list", "object":
		return true
	}
	return false
}

func (p *pdParser) parseConditional(subject *pdNode) *pdNode {
	it := p.it
	param := ""
	if subject == nil {
		param = it.cur().v
		it.consume(pdVariable)
		it.consumeValue(pdIdentifier, "is")
	} else {
		it.consume(pdIdentifier)
	}
	var flags pdFlags
	if it.isValue("not") {
		flags = pdFlagNegated
		it.consume(pdIdentifier)
	}
	target := p.parseType()
	it.skipNewLinesAndComments()
	it.consume(pdNullable)
	it.skipNewLinesAndComments()
	ifType := p.parseType()
	it.skipNewLinesAndComments()
	it.consume(pdColon)
	it.skipNewLinesAndComments()
	elseType := p.subParse()
	var n *pdNode
	if subject == nil {
		n = pdNew("ConditionalTypeForParameterNode", pdF(target), pdF(ifType), pdF(elseType))
		n.text = param
	} else {
		n = pdNew("ConditionalTypeNode", pdF(subject), pdF(target), pdF(ifType), pdF(elseType))
	}
	n.flags = flags
	return n
}

func (p *pdParser) parseNullable() *pdNode {
	p.it.consume(pdNullable)
	return pdNew("NullableTypeNode", pdF(p.parseAtomic()))
}

func (p *pdParser) isHTML() bool {
	it := p.it
	it.consume(pdOpenAngleBracket)
	if !it.is(pdIdentifier) {
		return false
	}
	name := it.cur().v
	it.next()
	if !it.tryConsume(pdCloseAngleBracket) {
		return false
	}
	if !p.hasEndTagCandidateAhead(name) {
		return false
	}
	endTag := "</" + name + ">"
	for !it.is(pdEnd) {
		if it.tryConsume(pdOpenAngleBracket) && strings.Contains(it.cur().v, "/"+name+">") ||
			strings.HasSuffix(it.cur().v, endTag) {
			return true
		}
		it.next()
	}
	return false
}

var pdEndTagRe = regexp.MustCompile(`/([^/>]+)>`)

func (p *pdParser) hasEndTagCandidateAhead(name string) bool {
	last := -1
	for i, t := range p.it.toks {
		if !strings.Contains(t.v, "/") {
			continue
		}
		for _, m := range pdEndTagRe.FindAllStringSubmatch(t.v, -1) {
			if m[1] == name {
				last = i
			}
		}
	}
	return last >= p.it.i
}

func (p *pdParser) parseGeneric(base *pdNode) *pdNode {
	it := p.it
	it.consume(pdOpenAngleBracket)
	it.skipNewLinesAndComments()
	var types []*pdNode
	var variances []string
	first := true
	for first || it.tryConsume(pdComma) {
		it.skipNewLinesAndComments()
		if !first && it.is(pdCloseAngleBracket) {
			break
		}
		first = false
		t, variance := p.parseGenericTypeArgument()
		types = append(types, t)
		variances = append(variances, variance)
		it.skipNewLinesAndComments()
	}
	t := pdNew("GenericTypeNode", pdF(base), pdL(types))
	t.variances = variances
	if base.start >= 0 {
		p.enrich(t, base.start)
	}
	it.consume(pdCloseAngleBracket)
	return t
}

func (p *pdParser) parseGenericTypeArgument() (*pdNode, string) {
	it := p.it
	start := it.i
	if it.tryConsume(pdWildcard) {
		return p.enrich(pdIdent("mixed"), start), "bivariant"
	}
	variance := "invariant"
	if it.tryConsumeValue("contravariant") {
		variance = "contravariant"
	} else if it.tryConsumeValue("covariant") {
		variance = "covariant"
	}
	return p.parseType(), variance
}

func (p *pdParser) parseTemplateTagValue(description bool) *pdNode {
	it := p.it
	name := it.cur().v
	it.consume(pdIdentifier)
	var bound, lower, def *pdNode
	if it.tryConsumeValue("of") || it.tryConsumeValue("as") {
		bound = p.parseType()
	}
	if it.tryConsumeValue("super") {
		lower = p.parseType()
	}
	if it.tryConsumeValue("=") {
		def = p.parseType()
	}
	if description {
		p.parseOptionalDescription(true)
	}
	n := pdNew("TemplateTagValueNode", pdF(bound), pdF(def), pdF(lower))
	n.name = name
	return n
}

func (p *pdParser) tryParseCallable(identifier *pdNode, hasTemplate bool) *pdNode {
	var t *pdNode
	p.it.push()
	if pdTry(func() { t = p.parseCallable(identifier, hasTemplate) }) {
		p.it.drop()
		return t
	}
	p.it.rollback()
	return identifier
}

func (p *pdParser) parseCallable(identifier *pdNode, hasTemplate bool) *pdNode {
	it := p.it
	var templates []*pdNode
	if hasTemplate {
		it.consume(pdOpenAngleBracket)
		first := true
		for first || it.tryConsume(pdComma) {
			it.skipNewLinesAndComments()
			if !first && it.is(pdCloseAngleBracket) {
				break
			}
			first = false
			start := it.i
			templates = append(templates, p.enrich(p.parseTemplateTagValue(false), start))
			it.skipNewLinesAndComments()
		}
		it.consume(pdCloseAngleBracket)
	}
	it.consume(pdOpenParentheses)
	it.skipNewLinesAndComments()
	var params []*pdNode
	if !it.is(pdCloseParentheses) {
		params = append(params, p.parseCallableParameter())
		it.skipNewLinesAndComments()
		for it.tryConsume(pdComma) {
			it.skipNewLinesAndComments()
			if it.is(pdCloseParentheses) {
				break
			}
			params = append(params, p.parseCallableParameter())
			it.skipNewLinesAndComments()
		}
	}
	it.consume(pdCloseParentheses)
	it.consume(pdColon)
	start := it.i
	ret := p.enrich(p.parseCallableReturnType(), start)
	return pdNew("CallableTypeNode", pdF(identifier), pdL(templates), pdL(params), pdF(ret))
}

func (p *pdParser) parseCallableParameter() *pdNode {
	it := p.it
	start := it.i
	t := p.parseType()
	n := pdNew("CallableTypeParameterNode", pdF(t))
	if it.tryConsume(pdReference) {
		n.flags |= pdFlagReference
	}
	if it.tryConsume(pdVariadic) {
		n.flags |= pdFlagVariadic
	}
	if it.is(pdVariable) {
		n.text = it.cur().v
		it.consume(pdVariable)
	}
	if it.tryConsume(pdEqual) {
		n.flags |= pdFlagOptional
	}
	return p.enrich(n, start)
}

func (p *pdParser) parseCallableReturnType() *pdNode {
	it := p.it
	start := it.i
	switch {
	case it.is(pdNullable):
		return p.parseNullable()
	case it.tryConsume(pdOpenParentheses):
		t := p.subParse()
		it.consume(pdCloseParentheses)
		if it.is(pdOpenSquareBracket) {
			t = p.tryParseArrayOrOffsetAccess(t)
		}
		return t
	case it.tryConsume(pdThisVariable):
		t := pdNew("ThisTypeNode")
		if it.is(pdOpenSquareBracket) {
			t = p.tryParseArrayOrOffsetAccess(p.enrich(t, start))
		}
		return t
	}
	value := it.cur().v
	it.push()
	if it.tryConsume(pdIdentifier) {
		t := pdIdent(value)
		if !it.is(pdDoubleColon) {
			// the save point stays on the stack, as upstream leaves it there
			switch {
			case it.is(pdOpenAngleBracket):
				t = p.parseGeneric(p.enrich(t, start))
				if it.is(pdOpenSquareBracket) {
					t = p.tryParseArrayOrOffsetAccess(p.enrich(t, start))
				}
			case it.is(pdOpenSquareBracket):
				t = p.tryParseArrayOrOffsetAccess(p.enrich(t, start))
			case pdIsShapeKind(t.name) && it.is(pdOpenCurlyBracket) && !it.precededByWS():
				if t.name == "object" {
					t = p.parseObjectShape()
				} else {
					p.enrich(t, start)
					t = p.parseArrayShape(t.name)
				}
				if it.is(pdOpenSquareBracket) {
					t = p.tryParseArrayOrOffsetAccess(p.enrich(t, start))
				}
			}
			return t
		}
		it.rollback()
	} else {
		it.drop()
	}
	expr := p.parseConstExpr(false)
	if expr.kind == "ConstExprArrayNode" {
		panic(pdError{})
	}
	t := p.enrich(pdNew("ConstTypeNode", pdF(expr)), start)
	if it.is(pdOpenSquareBracket) {
		t = p.tryParseArrayOrOffsetAccess(t)
	}
	return t
}

func (p *pdParser) tryParseArrayOrOffsetAccess(t *pdNode) *pdNode {
	it := p.it
	start := t.start
	if !pdTry(func() {
		for it.is(pdOpenSquareBracket) {
			it.push()
			canBeOffset := !it.precededByWS()
			it.consume(pdOpenSquareBracket)
			if canBeOffset && !it.is(pdCloseSquareBracket) {
				offset := p.parseType()
				it.consume(pdCloseSquareBracket)
				it.drop()
				t = pdNew("OffsetAccessTypeNode", pdF(t), pdF(offset))
			} else {
				it.consume(pdCloseSquareBracket)
				it.drop()
				t = pdNew("ArrayTypeNode", pdF(t))
			}
			if start >= 0 {
				p.enrich(t, start)
			}
		}
	}) {
		it.rollback()
	}
	return t
}

func (p *pdParser) parseArrayShape(kind string) *pdNode {
	it := p.it
	it.consume(pdOpenCurlyBracket)
	var items []*pdNode
	var unsealed *pdNode
	sealed := true
	for done := false; !done; {
		it.skipNewLinesAndComments()
		if it.tryConsume(pdCloseCurlyBracket) {
			n := pdNew("ArrayShapeNode", pdL(items), pdF(nil))
			n.text = kind
			return n
		}
		if it.tryConsume(pdVariadic) {
			sealed = false
			it.skipNewLinesAndComments()
			if it.is(pdOpenAngleBracket) {
				unsealed = p.parseUnsealedType(kind == "array")
				it.skipNewLinesAndComments()
			}
			it.tryConsume(pdComma)
			break
		}
		items = append(items, p.parseShapeItem(true))
		it.skipNewLinesAndComments()
		if !it.tryConsume(pdComma) {
			done = true
		}
		if it.cur().t == pdComment {
			it.next()
		}
	}
	it.skipNewLinesAndComments()
	it.consume(pdCloseCurlyBracket)
	n := pdNew("ArrayShapeNode", pdL(items), pdF(unsealed))
	n.text = kind
	if !sealed {
		n.flags = pdFlagUnsealed
	}
	return n
}

func (p *pdParser) parseUnsealedType(withKey bool) *pdNode {
	it := p.it
	start := it.i
	it.consume(pdOpenAngleBracket)
	it.skipNewLinesAndComments()
	value := p.parseType()
	it.skipNewLinesAndComments()
	var key *pdNode
	if withKey && it.tryConsume(pdComma) {
		it.skipNewLinesAndComments()
		key = value
		value = p.parseType()
		it.skipNewLinesAndComments()
	}
	it.consume(pdCloseAngleBracket)
	return p.enrich(pdNew("ArrayShapeUnsealedTypeNode", pdF(value), pdF(key)), start)
}

// parseShapeItem is parseArrayShapeItem, or parseObjectShapeItem when array is false.
func (p *pdParser) parseShapeItem(array bool) *pdNode {
	it := p.it
	start := it.i
	it.skipNewLinesAndComments()
	kind := "ObjectShapeItemNode"
	if array {
		kind = "ArrayShapeItemNode"
	}
	if !array {
		key := p.parseShapeKey(false)
		optional := it.tryConsume(pdNullable)
		it.consume(pdColon)
		n := pdNew(kind, pdF(key), pdF(p.parseType()))
		if optional {
			n.flags = pdFlagOptional
		}
		return p.enrich(n, start)
	}
	var n *pdNode
	it.push()
	if pdTry(func() {
		key := p.parseShapeKey(true)
		optional := it.tryConsume(pdNullable)
		it.consume(pdColon)
		n = pdNew(kind, pdF(key), pdF(p.parseType()))
		if optional {
			n.flags = pdFlagOptional
		}
	}) {
		it.drop()
		return p.enrich(n, start)
	}
	it.rollback()
	return p.enrich(pdNew(kind, pdF(nil), pdF(p.parseType())), start)
}

func (p *pdParser) parseShapeKey(array bool) *pdNode {
	it := p.it
	start := it.i
	var key *pdNode
	switch {
	case array && it.is(pdInteger):
		key = pdNew("ConstExprIntegerNode")
		key.text = strings.ReplaceAll(it.cur().v, "_", "")
		it.next()
	case it.is(pdSingleQuotedString, pdDoubleQuotedString):
		key = pdNew("ConstExprStringNode")
		key.text = pdStringText(it.cur())
		it.next()
	default:
		identifier := it.cur().v
		it.consume(pdIdentifier)
		key = pdIdent(identifier)
		if array && it.tryConsume(pdDoubleColon) {
			key = pdNew("ConstFetchNode")
			key.className = identifier
			key.name = it.cur().v
			it.consume(pdIdentifier)
		}
	}
	return p.enrich(key, start)
}

func (p *pdParser) parseObjectShape() *pdNode {
	it := p.it
	it.consume(pdOpenCurlyBracket)
	var items []*pdNode
	for {
		it.skipNewLinesAndComments()
		if it.tryConsume(pdCloseCurlyBracket) {
			return pdNew("ObjectShapeNode", pdL(items))
		}
		items = append(items, p.parseShapeItem(false))
		it.skipNewLinesAndComments()
		if !it.tryConsume(pdComma) {
			break
		}
	}
	it.skipNewLinesAndComments()
	it.consume(pdCloseCurlyBracket)
	return pdNew("ObjectShapeNode", pdL(items))
}

// ConstExprParser

func (p *pdParser) parseConstExpr(doctrine bool) *pdNode {
	it := p.it
	start := it.i
	switch {
	case it.is(pdFloat):
		n := pdNew("ConstExprFloatNode")
		n.text = strings.ReplaceAll(it.cur().v, "_", "")
		it.next()
		return p.enrich(n, start)
	case it.is(pdInteger):
		n := pdNew("ConstExprIntegerNode")
		n.text = strings.ReplaceAll(it.cur().v, "_", "")
		it.next()
		return p.enrich(n, start)
	case doctrine && it.is(pdDoctrineAnnotationString):
		it.next()
		return p.enrich(pdNew("DoctrineConstExprStringNode"), start)
	case it.is(pdSingleQuotedString, pdDoubleQuotedString):
		if doctrine {
			if it.is(pdSingleQuotedString) {
				panic(pdError{})
			}
			it.next()
			p.parseDoctrineString()
			return p.enrich(pdNew("DoctrineConstExprStringNode"), start)
		}
		n := pdNew("ConstExprStringNode")
		n.text = pdStringText(it.cur())
		it.next()
		return p.enrich(n, start)
	case it.is(pdIdentifier):
		identifier := it.cur().v
		it.next()
		switch strings.ToLower(identifier) {
		case "true", "false", "null":
			n := pdNew("ConstExprScalarNode")
			n.text = strings.ToLower(identifier)
			return p.enrich(n, start)
		case "array":
			it.consume(pdOpenParentheses)
			return p.parseConstArray(pdCloseParentheses, start, doctrine)
		}
		n := pdNew("ConstFetchNode")
		if it.tryConsume(pdDoubleColon) {
			var name strings.Builder
			last := pdEnd
			for {
				if last != pdIdentifier && it.cur().t == pdIdentifier {
					name.WriteString(it.cur().v)
					it.consume(pdIdentifier)
					last = pdIdentifier
					continue
				}
				if last != pdWildcard && it.tryConsume(pdWildcard) {
					name.WriteString("*")
					last = pdWildcard
					if it.skippedWS() != "" {
						break
					}
					continue
				}
				if last == pdEnd {
					it.consume(pdWildcard)
				}
				break
			}
			n.className, n.name = identifier, name.String()
		} else {
			n.name = identifier
		}
		return p.enrich(n, start)
	case it.tryConsume(pdOpenSquareBracket):
		return p.parseConstArray(pdCloseSquareBracket, start, doctrine)
	}
	panic(pdError{})
}

func (p *pdParser) parseConstArray(endToken pdTokType, start int, doctrine bool) *pdNode {
	it := p.it
	var items []*pdNode
	if !it.tryConsume(endToken) {
		for {
			itemStart := it.i
			expr := p.parseConstExpr(doctrine)
			var item *pdNode
			if it.tryConsume(pdDoubleArrow) {
				item = pdNew("ConstExprArrayItemNode", pdF(expr), pdF(p.parseConstExpr(doctrine)))
			} else {
				item = pdNew("ConstExprArrayItemNode", pdF(nil), pdF(expr))
			}
			items = append(items, p.enrich(item, itemStart))
			if !it.tryConsume(pdComma) || it.is(endToken) {
				break
			}
		}
		it.consume(endToken)
	}
	return p.enrich(pdNew("ConstExprArrayNode", pdL(items)), start)
}

// pdWalk visits n and its sub-nodes depth first, like NodeTraverser.
func pdWalk(n *pdNode, visit func(*pdNode)) {
	if n == nil {
		return
	}
	visit(n)
	for _, f := range n.fields {
		if f.isList {
			for _, c := range f.list {
				pdWalk(c, visit)
			}
		} else {
			pdWalk(f.node, visit)
		}
	}
}

// pdNodesByType is AnnotationHelper::getAnnotationNodesByType; looking for
// identifiers, it skips the keys of array and object shapes and of Doctrine
// arguments.
func pdNodesByType(n *pdNode, kind string) []*pdNode {
	var out []*pdNode
	ignored := map[*pdNode]bool{}
	pdWalk(n, func(c *pdNode) {
		if kind == "IdentifierTypeNode" {
			switch c.kind {
			case "ArrayShapeItemNode", "ObjectShapeItemNode", "DoctrineArgument":
				if k := c.fields[0].node; k != nil {
					ignored[k] = true
				}
			}
		}
		if c.kind == kind && !ignored[c] {
			out = append(out, c)
		}
	})
	return out
}

// pdPrintError is a LogicException of the printer.
type pdPrintError struct{}

func pdThrow() { panic(pdPrintError{}) }

// pdCatch turns a LogicException of the printer into ok being false.
func pdCatch(ok *bool) {
	if r := recover(); r != nil {
		if _, is := r.(pdPrintError); !is {
			panic(r)
		}
		*ok = false
	}
}

// pdClone is PhpDocParserHelper::cloneNode: a deep copy whose every node has
// the node it copies as its original.
func pdClone(n *pdNode) *pdNode {
	if n == nil {
		return nil
	}
	c := *n
	c.orig = n
	c.fields = make([]pdField, len(n.fields))
	for i, f := range n.fields {
		if !f.isList {
			c.fields[i] = pdField{node: pdClone(f.node)}
			continue
		}
		list := make([]*pdNode, len(f.list))
		for j, item := range f.list {
			list[j] = pdClone(item)
		}
		c.fields[i] = pdField{list: list, isList: true}
	}
	return &c
}

// pdReplaceIn swaps, below the clone n, the clone of target for fixed, like
// the visitor of AnnotationHelper::changeAnnotationNode.
func pdReplaceIn(n, target, fixed *pdNode) {
	swap := func(c *pdNode) *pdNode {
		if c != nil && c.orig == target {
			if want, typed := pdTypedSlots[n.kind]; typed && c == n.fields[0].node && fixed.kind != want {
				// a TypeError of the typed property in PHP
				pdThrow()
			}
			return fixed
		}
		if c != nil {
			pdReplaceIn(c, target, fixed)
		}
		return c
	}
	for i, f := range n.fields {
		if !f.isList {
			n.fields[i].node = swap(f.node)
			continue
		}
		for j, item := range f.list {
			f.list[j] = swap(item)
		}
	}
}

// pdTypedSlots are the nodes whose first sub-node is a property typed as one
// node class in PHP.
var pdTypedSlots = map[string]string{
	"DoctrineArgument":    "IdentifierTypeNode",
	"ExtendsTagValueNode": "GenericTypeNode",
}

func pdContains(n, target *pdNode) bool {
	found := false
	pdWalk(n, func(c *pdNode) {
		if c == target {
			found = true
		}
	})
	return found
}

// pdFixAnnotation is AnnotationHelper::fixAnnotation: the format preserving
// print of the doc comment root, parsed from toks, with target swapped for
// fixed. It is false where the printer throws.
func pdFixAnnotation(root *pdNode, toks []pdToken, newline string, target, fixed *pdNode) (out string, ok bool) {
	ok = true
	defer pdCatch(&ok)
	clone := pdClone(root)
	for _, tag := range clone.fields[0].list {
		if tag.kind == "PhpDocTagNode" && pdContains(tag.orig, target) {
			pdReplaceIn(tag, target, fixed)
			break
		}
	}
	return pdPrintFormatPreserving(clone, root, toks, newline)
}

// pdFix is pdFixAnnotation with target, an identifier or a constant fetch,
// swapped for a node of its class named replacement.
func pdFix(root *pdNode, toks []pdToken, target *pdNode, replacement string) (string, bool) {
	switch target.kind {
	case "IdentifierTypeNode", "ConstFetchNode":
	default:
		return "", false
	}
	fixed := pdNew(target.kind)
	fixed.name = replacement
	return pdFixAnnotation(root, toks, pdFirstNewline(toks), target, fixed)
}

// pdFirstNewline is the newline of the first end of line of a doc comment,
// what the parser's TokenIterator detects as its newline.
func pdFirstNewline(toks []pdToken) string {
	for _, t := range toks {
		if t.t != pdPhpdocEOL {
			continue
		}
		if strings.HasPrefix(t.v, "\r\n") {
			return "\r\n"
		}
		if strings.HasPrefix(t.v, "\n") {
			return "\n"
		}
	}
	return ""
}

// pdPrint is Printer::print, as AnnotationTypeHelper::print uses it; it is
// false for a node the port cannot print.
func pdPrint(n *pdNode) (out string, ok bool) {
	ok = true
	defer pdCatch(&ok)
	pr := &pdPrinter{lineStart: "\n *"}
	return pr.print(n), true
}

// pdPrintFormatPreserving is Printer::printFormatPreserving of the clone node
// of the doc comment orig, parsed from toks with newline detected.
func pdPrintFormatPreserving(node, orig *pdNode, toks []pdToken, newline string) (out string, ok bool) {
	ok = true
	defer pdCatch(&ok)
	pr := &pdPrinter{toks: toks, newline: newline}
	pr.lineStart = pr.detectLineStart()
	index := 0
	if res, done := pr.array(node.fields[0].list, orig.fields[0].list, &index, "PhpDocNode", 0); done {
		return res + pr.content(index, len(toks)), true
	}
	return pr.print(node), true
}

type pdPrinter struct {
	toks      []pdToken
	newline   string
	lineStart string
}

func (pr *pdPrinter) nl() string {
	if pr.newline == "" {
		return "\n"
	}
	return pr.newline
}

// content is TokenIterator::getContentBetween.
func (pr *pdPrinter) content(start, end int) string {
	if start < 0 || end > len(pr.toks) {
		pdThrow()
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		b.WriteString(pr.toks[i].v)
	}
	return b.String()
}

// hasParentheses is TokenIterator::hasParentheses.
func (pr *pdPrinter) hasParentheses(start, end int) bool {
	ws := func(t pdTokType) bool { return t == pdHorizontalWS || t == pdPhpdocEOL }
	before := false
	for i := start - 1; i >= 0; i-- {
		if pr.toks[i].t == pdOpenParentheses {
			before = true
			break
		}
		if !ws(pr.toks[i].t) {
			break
		}
	}
	if !before {
		return false
	}
	for i := end + 1; i < len(pr.toks); i++ {
		if pr.toks[i].t == pdCloseParentheses {
			return true
		}
		if !ws(pr.toks[i].t) {
			break
		}
	}
	return false
}

func (pr *pdPrinter) detectLineStart() string {
	for _, t := range pr.toks {
		if t.t != pdPhpdocEOL {
			continue
		}
		if p := strings.IndexByte(t.v, '*'); p >= 0 {
			return t.v[:p+1]
		}
	}
	return pr.nl() + " *"
}

// pdFieldNames names the sub-nodes the printer's maps refer to, by position.
var pdFieldNames = map[string][]string{
	"PhpDocNode":           {"children"},
	"UnionTypeNode":        {"types"},
	"IntersectionTypeNode": {"types"},
	"ArrayShapeNode":       {"items", "unsealedType"},
	"ObjectShapeNode":      {"items"},
	"CallableTypeNode":     {"identifier", "templateTypes", "parameters", "returnType"},
	"GenericTypeNode":      {"type", "genericTypes"},
	"ConstExprArrayNode":   {"items"},
	"MethodTagValueNode":   {"returnType", "templateTypes", "parameters"},
	"DoctrineArray":        {"items"},
	"DoctrineAnnotation":   {"arguments"},
	"ArrayTypeNode":        {"type"},
	"OffsetAccessTypeNode": {"type", "offset"},
	"NullableTypeNode":     {"type"},
	"ConditionalTypeNode":  {"subjectType", "targetType", "if", "else"},
}

func pdMapKey(kind string, field int) string {
	if names := pdFieldNames[kind]; field < len(names) {
		return kind + "->" + names[field]
	}
	return kind + "->"
}

var pdListInsertion = map[string]string{
	"PhpDocNode->children":            "\n * ",
	"UnionTypeNode->types":            "|",
	"IntersectionTypeNode->types":     "&",
	"ArrayShapeNode->items":           ", ",
	"ObjectShapeNode->items":          ", ",
	"CallableTypeNode->parameters":    ", ",
	"CallableTypeNode->templateTypes": ", ",
	"GenericTypeNode->genericTypes":   ", ",
	"ConstExprArrayNode->items":       ", ",
	"MethodTagValueNode->parameters":  ", ",
	"DoctrineArray->items":            ", ",
	"DoctrineAnnotation->arguments":   ", ",
}

var pdEmptyListInsertion = map[string]string{
	"CallableTypeNode->parameters":  "(",
	"ArrayShapeNode->items":         "{",
	"ObjectShapeNode->items":        "{",
	"DoctrineArray->items":          "{",
	"DoctrineAnnotation->arguments": "(",
}

var pdParentheses = map[string][]string{
	"CallableTypeNode->returnType":     {"CallableTypeNode", "UnionTypeNode", "IntersectionTypeNode"},
	"ArrayTypeNode->type":              {"CallableTypeNode", "UnionTypeNode", "IntersectionTypeNode", "ConstTypeNode", "NullableTypeNode"},
	"OffsetAccessTypeNode->type":       {"CallableTypeNode", "UnionTypeNode", "IntersectionTypeNode", "NullableTypeNode"},
	"NullableTypeNode->type":           {"UnionTypeNode", "IntersectionTypeNode", "NullableTypeNode"},
	"ConditionalTypeNode->subjectType": {"UnionTypeNode", "IntersectionTypeNode", "NullableTypeNode"},
}

var pdParenthesesList = map[string][]string{
	"IntersectionTypeNode->types": {"IntersectionTypeNode", "UnionTypeNode", "NullableTypeNode"},
	"UnionTypeNode->types":        {"IntersectionTypeNode", "UnionTypeNode", "NullableTypeNode"},
}

// node is Printer::printNodeFormatPreserving.
func (pr *pdPrinter) node(n *pdNode) string {
	o := n.orig
	if o == nil {
		return pr.replacement(n)
	}
	if n.kind != o.kind || o.start < 0 || o.end < 0 {
		pdThrow()
	}
	var b strings.Builder
	pos := o.start
	for i, f := range n.fields {
		of := o.fields[i]
		if f.isList {
			if pdSameList(f.list, of.list) {
				continue
			}
			res, ok := pr.array(f.list, of.list, &pos, n.kind, i)
			if !ok {
				return pr.replacement(n)
			}
			b.WriteString(res)
			continue
		}
		sub, osub := f.node, of.node
		if osub == nil {
			if sub == nil {
				continue
			}
			return pr.replacement(n)
		}
		if osub.start < 0 || osub.end < 0 {
			pdThrow()
		}
		if osub.end < osub.start {
			// the original sub-node has no tokens, e.g. a Doctrine annotation without arguments
			if sub != nil && (sub.orig == osub || pr.print(sub) == pr.print(osub)) {
				continue
			}
			return pr.replacement(n)
		}
		if sub == nil {
			return pr.replacement(n)
		}
		b.WriteString(pr.content(pos, osub.start))
		key := pdMapKey(n.kind, i)
		need := slices.Contains(pdParentheses[key], sub.kind)
		if sub.orig != nil {
			need = need && !slices.Contains(pdParentheses[key], sub.orig.kind)
		}
		add := need && !pr.hasParentheses(osub.start, osub.end)
		if add {
			b.WriteString("(")
		}
		b.WriteString(pr.node(sub))
		if add {
			b.WriteString(")")
		}
		pos = osub.end + 1
	}
	b.WriteString(pr.content(pos, o.end+1))
	return b.String()
}

// pdSameList is PHP's === of two lists of nodes.
func pdSameList(a, b []*pdNode) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// array is Printer::printArrayFormatPreserving; it is false where upstream
// returns null.
func (pr *pdPrinter) array(nodes, orig []*pdNode, index *int, parent string, field int) (string, bool) {
	diff := pdDiff(orig, nodes)
	key := pdMapKey(parent, field)
	insert, hasInsert := pdListInsertion[key]
	var result strings.Builder
	beforeFirstKeepOrReplace := true
	skipRemovedNode := false
	var delayed []*pdNode
	insertNewline := false
	multiline, before, after := pr.isMultiline(*index, orig)
	if hasInsert && insert == "\n * " {
		if b, a, ok := pr.detectLastChildIndent(orig); ok {
			before, after = b, a
		}
		insert = pr.nl() + before + "*" + after
	}
	lineBreak := func() string { return pr.nl() + before + "*" + after }
	parens := func(kind string) bool { return slices.Contains(pdParenthesesList[key], kind) }
	for i, d := range diff {
		switch d.typ {
		case pdKeep, pdReplace:
			beforeFirstKeepOrReplace = false
			item, o := d.new, d.old
			if o.start < 0 || o.end < 0 || o.start < *index {
				pdThrow()
			}
			if skipRemovedNode {
				// the separator after the removed first item belongs to it
				skipRemovedNode = false
			} else {
				result.WriteString(pr.content(*index, o.start))
			}
			for _, add := range delayed {
				if parens(add.kind) {
					result.WriteString("(")
				}
				pr.appendNode(&result, pr.node(add), after)
				if parens(add.kind) {
					result.WriteString(")")
				}
				result.WriteString(insert)
				if insertNewline {
					result.WriteString(lineBreak())
				}
			}
			delayed = nil
			add := parens(item.kind) && !parens(o.kind) && !pr.hasParentheses(o.start, o.end)
			if add {
				result.WriteString("(")
			}
			result.WriteString(pr.node(item))
			if add {
				result.WriteString(")")
			}
			*index = o.end + 1
		case pdAdd:
			if !hasInsert {
				return "", false
			}
			item := d.new
			if insert == ", " && multiline {
				insert = ","
				insertNewline = true
			}
			if beforeFirstKeepOrReplace {
				// will be inserted at the next "replace" or "keep" element
				delayed = append(delayed, item)
				continue
			}
			end := *index - 1
			result.WriteString(insert)
			if insertNewline {
				result.WriteString(lineBreak())
			}
			if parens(item.kind) {
				result.WriteString("(")
			}
			pr.appendNode(&result, pr.node(item), after)
			if parens(item.kind) {
				result.WriteString(")")
			}
			*index = end + 1
		case pdRemove:
			o := d.old
			if o.start < 0 || o.end < 0 {
				pdThrow()
			}
			if i == 0 {
				// removing from the start keeps the tokens before the node and drops those after it
				if key == "PhpDocNode->children" {
					// the line of the removed child is dropped, the next child keeps its own line
					for j := *index; j < o.start; j++ {
						if pr.toks[j].t == pdPhpdocEOL {
							break
						}
						result.WriteString(pr.toks[j].v)
					}
				} else {
					result.WriteString(pr.content(*index, o.start))
					skipRemovedNode = true
				}
			}
			*index = o.end + 1
		}
	}
	if len(delayed) > 0 {
		find, ok := pdEmptyListInsertion[key]
		if !ok {
			return "", false
		}
		for ; *index < len(pr.toks); *index++ {
			result.WriteString(pr.toks[*index].v)
			if pr.toks[*index].v == find {
				*index++
				break
			}
		}
		for j, add := range delayed {
			if j > 0 {
				result.WriteString(insert)
				if insertNewline {
					result.WriteString(lineBreak())
				}
			}
			pr.appendNode(&result, pr.node(add), after)
		}
	}
	return result.String(), true
}

type pdDiffType uint8

const (
	pdKeep pdDiffType = iota
	pdRemove
	pdAdd
	pdReplace
)

type pdDiffElem struct {
	typ      pdDiffType
	old, new *pdNode
}

// pdDiff is Differ::diffWithReplacements from old to new, where a new node
// equals the old one it is a clone of.
func pdDiff(old, new []*pdNode) []pdDiffElem {
	equal := func(a, b *pdNode) bool { return a != nil && b != nil && b.orig == a }
	n, m := len(old), len(new)
	v := map[int]int{1: 0}
	var trace []map[int]int
	x, y := 0, 0
	for d := 0; d <= n+m; d++ {
		trace = append(trace, maps.Clone(v))
		done := false
		for k := -d; k <= d; k += 2 {
			if k == -d || k != d && v[k-1] < v[k+1] {
				x = v[k+1]
			} else {
				x = v[k-1] + 1
			}
			y = x - k
			for x < n && y < m && equal(old[x], new[y]) {
				x++
				y++
			}
			v[k] = x
			if x >= n && y >= m {
				done = true
				break
			}
		}
		if done {
			break
		}
	}
	var result []pdDiffElem
	for d, v := range slices.Backward(trace) {
		k := x - y
		prevK := k - 1
		if k == -d || k != d && v[k-1] < v[k+1] {
			prevK = k + 1
		}
		prevX := v[prevK]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			result = append(result, pdDiffElem{pdKeep, old[x-1], new[y-1]})
			x--
			y--
		}
		if d == 0 {
			break
		}
		for x > prevX {
			result = append(result, pdDiffElem{pdRemove, old[x-1], nil})
			x--
		}
		for y > prevY {
			result = append(result, pdDiffElem{pdAdd, nil, new[y-1]})
			y--
		}
	}
	slices.Reverse(result)
	// coalesce equal-length sequences of remove+add into replace
	var out []pdDiffElem
	for i := 0; i < len(result); i++ {
		if result[i].typ != pdRemove {
			out = append(out, result[i])
			continue
		}
		j := i
		for j < len(result) && result[j].typ == pdRemove {
			j++
		}
		k := j
		for k < len(result) && result[k].typ == pdAdd {
			k++
		}
		if j-i == k-j {
			for l := range j - i {
				out = append(out, pdDiffElem{pdReplace, result[i+l].old, result[j+l].new})
			}
		} else {
			out = append(out, result[i:k]...)
		}
		i = k - 1
	}
	return out
}

// appendNode writes a printed node; a new empty line in a docblock is " *",
// without the space after the asterisk.
func (pr *pdPrinter) appendNode(result *strings.Builder, printed, after string) {
	lineStart := "*" + after
	s := result.String()
	if printed != "" || after == "" || !strings.HasSuffix(s, lineStart) {
		result.WriteString(printed)
		return
	}
	result.Reset()
	result.WriteString(s[:len(s)-len(after)])
}

var pdIndentRe = regexp.MustCompile(`\n([\t ]*)\*( *)`)

func (pr *pdPrinter) isMultiline(initial int, nodes []*pdNode) (bool, string, string) {
	multiline := len(nodes) > 1
	pos := initial
	var all strings.Builder
	for _, n := range nodes {
		if n == nil {
			continue
		}
		end := n.end + 1
		text := pr.content(pos, end)
		all.WriteString(text)
		if !strings.Contains(text, "\n") {
			// a newline is required between every item
			multiline = false
		}
		pos = end
	}
	matches := pdIndentRe.FindAllStringSubmatch(all.String(), -1)
	if len(matches) == 0 {
		return multiline, " ", "  "
	}
	before, after := "", ""
	for _, m := range matches {
		if len(m[1]) > len(before) {
			before = m[1]
		}
		if len(m[2]) > len(after) {
			after = m[2]
		}
	}
	if before == "" {
		before = " "
	}
	if after == "" {
		after = "  "
	}
	return multiline, before, after
}

var pdLastIndentRe = regexp.MustCompile(`\n([\t ]*)\*([\t ]*)$`)

// detectLastChildIndent is the indentation around the asterisk on the line
// where the last child of a doc comment starts.
func (pr *pdPrinter) detectLastChildIndent(nodes []*pdNode) (string, string, bool) {
	for _, node := range slices.Backward(nodes) {
		index := node.start - 1
		lineStart := ""
		for index >= 0 && index < len(pr.toks) && pr.toks[index].t == pdHorizontalWS {
			lineStart = pr.toks[index].v + lineStart
			index--
		}
		if index < 0 || index >= len(pr.toks) || pr.toks[index].t != pdPhpdocEOL {
			continue
		}
		lineStart = pr.toks[index].v + lineStart
		if m := pdLastIndentRe.FindStringSubmatch(lineStart); m != nil {
			return m[1], m[2], true
		}
	}
	return "", "", false
}

// replacement is Printer::printReplacement.
func (pr *pdPrinter) replacement(n *pdNode) string {
	if n.kind == "DoctrineTagValueNode" {
		pdThrow()
	}
	return pr.addLineStarts(pr.print(n))
}

// addLineStarts puts back the asterisk the parser removes at the start of each
// line of a text.
func (pr *pdPrinter) addLineStarts(s string) string {
	if !strings.Contains(s, "\n") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
			i++
		} else if s[i] != '\n' {
			b.WriteByte(s[i])
			continue
		}
		b.WriteString(pr.lineStart)
		if i+1 < len(s) && s[i+1] != '\r' && s[i+1] != '\n' {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// print is Printer::print for the nodes of types; the others throw.
func (pr *pdPrinter) print(n *pdNode) string {
	switch n.kind {
	case "CallableTypeParameterNode":
		s := pr.print(n.fields[0].node) + " "
		if n.has(pdFlagReference) {
			s += "&"
		}
		if n.has(pdFlagVariadic) {
			s += "..."
		}
		s = pdTrim(s + n.text)
		if n.has(pdFlagOptional) {
			s += "="
		}
		return s
	case "ArrayShapeUnsealedTypeNode":
		if key := n.fields[1].node; key != nil {
			return "<" + pr.printType(key) + ", " + pr.printType(n.fields[0].node) + ">"
		}
		return "<" + pr.printType(n.fields[0].node) + ">"
	case "ArrayShapeItemNode", "ObjectShapeItemNode":
		key := n.fields[0].node
		if key == nil {
			return pr.printType(n.fields[1].node)
		}
		optional := ""
		if n.has(pdFlagOptional) {
			optional = "?"
		}
		return pr.print(key) + optional + ": " + pr.printType(n.fields[1].node)
	case "TemplateTagValueNode":
		s := n.name
		if bound := n.fields[0].node; bound != nil {
			s += " of " + pr.printType(bound)
		}
		if lower := n.fields[2].node; lower != nil {
			s += " super " + pr.printType(lower)
		}
		if def := n.fields[1].node; def != nil {
			s += " = " + pr.printType(def)
		}
		return pdTrim(s + " ")
	}
	if strings.HasPrefix(n.kind, "ConstExpr") || n.kind == "ConstFetchNode" {
		return pdConstExpr(n)
	}
	return pr.printType(n)
}

func (pr *pdPrinter) printType(n *pdNode) string {
	switch n.kind {
	case "ArrayShapeNode":
		var items []string
		for _, item := range n.fields[0].list {
			items = append(items, pr.print(item))
		}
		if n.has(pdFlagUnsealed) {
			unsealed := ""
			if u := n.fields[1].node; u != nil {
				unsealed = pr.print(u)
			}
			items = append(items, "..."+unsealed)
		}
		return n.text + "{" + strings.Join(items, ", ") + "}"
	case "ArrayTypeNode":
		return pr.offsetAccessType(n.fields[0].node) + "[]"
	case "CallableTypeNode":
		ret := n.fields[3].node
		var returnType string
		switch ret.kind {
		case "CallableTypeNode", "UnionTypeNode", "IntersectionTypeNode":
			returnType = "(" + pr.printType(ret) + ")"
		default:
			returnType = pr.printType(ret)
		}
		template := ""
		if templates := n.fields[1].list; len(templates) > 0 {
			var parts []string
			for _, t := range templates {
				parts = append(parts, pr.print(t))
			}
			template = "<" + strings.Join(parts, ", ") + ">"
		}
		var params []string
		for _, p := range n.fields[2].list {
			params = append(params, pr.print(p))
		}
		return pr.printType(n.fields[0].node) + template + "(" + strings.Join(params, ", ") + "): " + returnType
	case "ConditionalTypeForParameterNode":
		return "(" + n.text + " " + pdIs(n) + " " + pr.printType(n.fields[0].node) + " ? " +
			pr.printType(n.fields[1].node) + " : " + pr.printType(n.fields[2].node) + ")"
	case "ConditionalTypeNode":
		subject := n.fields[0].node
		s := pr.printType(subject)
		switch subject.kind {
		case "UnionTypeNode", "IntersectionTypeNode", "NullableTypeNode":
			s = "(" + s + ")"
		}
		return "(" + s + " " + pdIs(n) + " " + pr.printType(n.fields[1].node) + " ? " +
			pr.printType(n.fields[2].node) + " : " + pr.printType(n.fields[3].node) + ")"
	case "ConstTypeNode":
		return pdConstExpr(n.fields[0].node)
	case "GenericTypeNode":
		var types []string
		for i, t := range n.fields[1].list {
			variance := "invariant"
			if i < len(n.variances) {
				variance = n.variances[i]
			}
			switch variance {
			case "invariant":
				types = append(types, pr.printType(t))
			case "bivariant":
				types = append(types, "*")
			default:
				types = append(types, variance+" "+pr.print(t))
			}
		}
		return pr.printType(n.fields[0].node) + "<" + strings.Join(types, ", ") + ">"
	case "IdentifierTypeNode":
		return n.name
	case "IntersectionTypeNode", "UnionTypeNode":
		var items []string
		for _, t := range n.fields[0].list {
			switch t.kind {
			case "IntersectionTypeNode", "UnionTypeNode", "NullableTypeNode":
				items = append(items, "("+pr.printType(t)+")")
			default:
				items = append(items, pr.printType(t))
			}
		}
		if n.kind == "IntersectionTypeNode" {
			return strings.Join(items, "&")
		}
		return strings.Join(items, "|")
	case "InvalidTypeNode":
		return "*Invalid type*"
	case "NullableTypeNode":
		inner := n.fields[0].node
		switch inner.kind {
		case "IntersectionTypeNode", "UnionTypeNode", "NullableTypeNode":
			return "?(" + pr.printType(inner) + ")"
		}
		return "?" + pr.printType(inner)
	case "ObjectShapeNode":
		var items []string
		for _, item := range n.fields[0].list {
			items = append(items, pr.print(item))
		}
		return "object{" + strings.Join(items, ", ") + "}"
	case "OffsetAccessTypeNode":
		return pr.offsetAccessType(n.fields[0].node) + "[" + pr.printType(n.fields[1].node) + "]"
	case "ThisTypeNode":
		return "$this"
	}
	pdThrow()
	return ""
}

func (pr *pdPrinter) offsetAccessType(n *pdNode) string {
	switch n.kind {
	case "CallableTypeNode", "UnionTypeNode", "IntersectionTypeNode", "NullableTypeNode":
		return "(" + pr.printType(n) + ")"
	}
	return pr.printType(n)
}

func pdIs(n *pdNode) string {
	if n.has(pdFlagNegated) {
		return "is not"
	}
	return "is"
}

// pdConstExpr is the __toString of a ConstExprNode.
func pdConstExpr(n *pdNode) string {
	switch n.kind {
	case "ConstExprIntegerNode", "ConstExprFloatNode", "ConstExprStringNode", "ConstExprScalarNode":
		return n.text
	case "ConstFetchNode":
		if n.className == "" {
			return n.name
		}
		return n.className + "::" + n.name
	case "ConstExprArrayNode":
		var items []string
		for _, item := range n.fields[0].list {
			items = append(items, pdConstExpr(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case "ConstExprArrayItemNode":
		if key := n.fields[0].node; key != nil {
			return pdConstExpr(key) + " => " + pdConstExpr(n.fields[1].node)
		}
		return pdConstExpr(n.fields[1].node)
	}
	pdThrow()
	return ""
}

// pdTrim is PHP's trim.
func pdTrim(s string) string { return strings.Trim(s, " \t\n\r\x00\x0B") }

// pdStringText is what the ConstExprStringNode of a quoted string token
// prints as: StringUnescaper::unescapeString, then ConstExprStringNode::__toString.
func pdStringText(t pdToken) string {
	raw := t.v[1 : len(t.v)-1]
	if t.t == pdSingleQuotedString {
		value := strings.ReplaceAll(strings.ReplaceAll(raw, `\\`, `\`), `\'`, `'`)
		return "'" + pdAddcslashes(value, `'\`) + "'"
	}
	value := pdUnescapeDouble(strings.ReplaceAll(raw, `\"`, `"`))
	return `"` + pdEscapeDouble(value) + `"`
}

func pdAddcslashes(s, chars string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(chars, s[i]) >= 0 {
			b.WriteByte('\\')
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

var pdEscapeRe = regexp.MustCompile(`\\([\\nrtfve]|[xX][0-9a-fA-F]{1,2}|[0-7]{1,3}|u\{([0-9a-fA-F]+)\})`)

func pdUnescapeDouble(s string) string {
	return pdEscapeRe.ReplaceAllStringFunc(s, func(m string) string {
		seq := m[1:]
		switch seq {
		case `\`:
			return `\`
		case "n":
			return "\n"
		case "r":
			return "\r"
		case "t":
			return "\t"
		case "f":
			return "\f"
		case "v":
			return "\v"
		case "e":
			return "\x1B"
		}
		switch seq[0] {
		case 'x', 'X':
			v, _ := strconv.ParseUint(seq[1:], 16, 8)
			return string([]byte{byte(v)})
		case 'u':
			v, _ := strconv.ParseUint(seq[2:len(seq)-1], 16, 64)
			return pdCodePointToUtf8(v)
		}
		v, _ := strconv.ParseUint(seq, 8, 16)
		return string([]byte{byte(v)})
	})
}

func pdCodePointToUtf8(n uint64) string {
	switch {
	case n <= 0x7F:
		return string([]byte{byte(n)})
	case n <= 0x7FF:
		return string([]byte{byte(n>>6 + 0xC0), byte(n&0x3F + 0x80)})
	case n <= 0xFFFF:
		return string([]byte{byte(n>>12 + 0xE0), byte(n>>6&0x3F + 0x80), byte(n&0x3F + 0x80)})
	case n <= 0x1FFFFF:
		return string([]byte{byte(n>>18 + 0xF0), byte(n>>12&0x3F + 0x80), byte(n>>6&0x3F + 0x80), byte(n&0x3F + 0x80)})
	}
	return "\xef\xbf\xbd"
}

// pdEscapeDouble is ConstExprStringNode::escapeDoubleQuotedString.
func pdEscapeDouble(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		switch c {
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\f':
			b.WriteString(`\f`)
		case '\v':
			b.WriteString(`\v`)
		case '$', '"', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			if c < 0x80 {
				if c <= 0x08 || c >= 0x0E && c <= 0x1F {
					fmt.Fprintf(&b, `\x%02x`, c)
				} else {
					b.WriteByte(c)
				}
				break
			}
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size <= 1 {
				fmt.Fprintf(&b, `\x%02x`, c)
				i++
				continue
			}
			b.WriteString(s[i : i+size])
			i += size
			continue
		}
		i++
	}
	return b.String()
}

// slvAnnotation is slevomat's Annotation: a tag of a doc comment.
type slvAnnotation struct {
	node *pdNode // PhpDocTagNode
	name string
}

func (a slvAnnotation) isInvalid() bool { return a.node.value().kind == "InvalidTagValueNode" }

// slvParsedDoc is slevomat's ParsedDocComment.
type slvParsedDoc struct {
	open, close int
	root        *pdNode
	toks        []pdToken
	newline     string
	annotations []slvAnnotation
}

// slvDocs holds slevomat's SniffLocalCache of parsed doc comments for one pass.
type slvDocs struct {
	f      *slvFile
	parsed map[int]*slvParsedDoc
	uses   *slvUseStatements
}

func newSlvDocs(f *slvFile) *slvDocs {
	return &slvDocs{f: f, parsed: map[int]*slvParsedDoc{}}
}

func (f *slvFile) isDocOpen(i int) bool { return f.toks[i].doc == slvDocOpen }

// docCloser is the comment_closer of a doc comment opener.
func (f *slvFile) docCloser(open int) int {
	for i := open + 1; i < len(f.toks); i++ {
		if f.toks[i].doc == slvDocClose {
			return i
		}
	}
	return len(f.toks) - 1
}

// parse is DocCommentHelper::parseDocComment, or nil where it fails.
func (d *slvDocs) parse(open int) *slvParsedDoc {
	if p, ok := d.parsed[open]; ok {
		return p
	}
	closeIdx := d.f.docCloser(open)
	root, toks, newline := pdParseWithNewline(strings.TrimSpace(d.f.content(open, closeIdx)))
	var p *slvParsedDoc
	if root != nil {
		p = &slvParsedDoc{open: open, close: closeIdx, root: root, toks: toks, newline: newline}
		for _, c := range root.fields[0].list {
			if c.kind == "PhpDocTagNode" {
				p.annotations = append(p.annotations, slvAnnotation{node: c, name: c.name})
			}
		}
	}
	d.parsed[open] = p
	return p
}

// annotations is AnnotationHelper::getAnnotations of a doc comment opener,
// optionally only those named name.
func (d *slvDocs) annotations(open int, name string) []slvAnnotation {
	p := d.parse(open)
	if p == nil {
		return nil
	}
	if name == "" {
		return p.annotations
	}
	var out []slvAnnotation
	for _, a := range p.annotations {
		if a.name == name {
			out = append(out, a)
		}
	}
	return out
}

// fix replaces target with replacement in its doc comment, as slevomat sniffs
// do with AnnotationHelper::fixAnnotation and FixerHelper::change; it is false
// where the printer throws, which aborts the whole file in ECS.
func (d *slvDocs) fix(open int, target *pdNode, replacement string) bool {
	switch target.kind {
	case "IdentifierTypeNode", "ConstFetchNode":
	default:
		return false
	}
	fixed := pdNew(target.kind)
	fixed.name = replacement
	return d.fixNode(open, target, fixed)
}

// fixNode is fix with target swapped for the node fixed, which may hold nodes
// of the parsed doc comment.
func (d *slvDocs) fixNode(open int, target, fixed *pdNode) bool {
	p := d.parse(open)
	out, ok := pdFixAnnotation(p.root, p.toks, p.newline, target, fixed)
	if !ok {
		return false
	}
	d.f.beginChangeset()
	d.f.change(p.open, p.close, out)
	d.f.endChangeset()
	return true
}

func (d *slvDocs) useStatements() *slvUseStatements {
	if d.uses == nil {
		d.uses = d.f.useStatements()
	}
	return d.uses
}

// isInline is DocCommentHelper::isInline.
func (d *slvDocs) isInline(open int) bool {
	f := d.f
	if n := f.nextNonWhitespace(f.docCloser(open)+1, len(f.toks)); n >= 0 && (f.isModifier(n) || f.isClassTypeKeyword(n) || f.isAttributeOpen(n)) {
		return false
	}
	p := d.parse(open)
	if p == nil {
		return false
	}
	for _, a := range p.annotations {
		if slvInlineVarTag.MatchString(a.name) {
			return true
		}
	}
	return false
}

var slvInlineVarTag = regexp.MustCompile(`(?i)^@(?:(?:phpstan|psalm)-)?var`)

func (f *slvFile) isModifier(i int) bool {
	if f.toks[i].kind != token.Keyword && f.toks[i].kind != token.Ident {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "final", "abstract", "var", "public", "protected", "private", "readonly", "static",
		"public(set)", "protected(set)", "private(set)":
		return true
	}
	return false
}

// docOwner is DocCommentHelper::findDocCommentOwnerPointer, or -1.
func (d *slvDocs) docOwner(open int) int {
	if d.isInline(open) {
		return -1
	}
	f := d.f
	for i := f.docCloser(open) + 1; i < len(f.toks); i++ {
		switch {
		case f.isDocOpen(i):
			return -1
		case f.isAttributeOpen(i):
			if c := f.toks[i].closer; c > i {
				i = c
			}
		case f.isKeyword(i, "function") || f.isClassTypeKeyword(i) || f.isKeyword(i, "const") ||
			f.toks[i].kind == token.Variable && f.toks[i].orig != "$":
			return i
		}
	}
	return -1
}

// classPointer is ClassHelper::getClassPointer, or -1.
func (f *slvFile) classPointer(i int) int {
	best := -1
	for c := range f.toks {
		if !f.isKeyword(c, "class") && !f.isKeyword(c, "interface") && !f.isKeyword(c, "trait") && !f.isKeyword(c, "enum") {
			continue
		}
		closeIdx := f.scopeCloser(c)
		if closeIdx <= c {
			continue
		}
		if open := f.toks[closeIdx].closer; open < i && closeIdx > i {
			best = c
		}
	}
	return best
}

// docOpenBefore is DocCommentHelper::findDocCommentOpenPointer of a
// non-docblock token, or -1.
func (f *slvFile) docOpenBefore(i int) int {
	for j := i - 1; j >= 0; j-- {
		if f.toks[j].doc == slvDocClose {
			for k := j; k >= 0; k-- {
				if f.isDocOpen(k) {
					return k
				}
			}
			return -1
		}
		if f.is(j, ";") || f.is(j, "{") || f.is(j, "}") {
			return -1
		}
	}
	return -1
}

var (
	slvTemplateTags = []string{"@template", "@psalm-template", "@phpstan-template",
		"@template-covariant", "@psalm-template-covariant", "@phpstan-template-covariant"}
	slvAliasTags = []string{"@psalm-type", "@phpstan-type", "@psalm-import-type", "@phpstan-import-type"}
)

// isTypeDefinedInAnnotation is TypeHintHelper::isTypeDefinedInAnnotation for a
// doc comment opener.
func (d *slvDocs) isTypeDefinedInAnnotation(open int, name string) bool {
	return d.templateNames(open)[name] || d.isAlias(open, name)
}

func (d *slvDocs) templateNamesOf(open int) map[string]bool {
	out := map[string]bool{}
	for _, tag := range slvTemplateTags {
		for _, a := range d.annotations(open, tag) {
			if !a.isInvalid() {
				out[a.node.value().name] = true
			}
		}
	}
	return out
}

func (d *slvDocs) templateNames(open int) map[string]bool {
	f := d.f
	names := map[string]bool{}
	if owner := d.docOwner(open); owner >= 0 {
		if f.isClassTypeKeyword(owner) {
			return d.templateNamesOf(open)
		}
		if f.isKeyword(owner, "function") {
			names = d.templateNamesOf(open)
		}
	}
	class := f.classPointer(open)
	if class < 0 {
		return names
	}
	classDoc := f.docOpenBefore(class)
	if classDoc < 0 {
		return names
	}
	for k := range d.templateNamesOf(classDoc) {
		names[k] = true
	}
	return names
}

func (d *slvDocs) isAlias(open int, name string) bool {
	f := d.f
	classDoc := -1
	if owner := d.docOwner(open); owner >= 0 && f.isClassTypeKeyword(owner) {
		classDoc = open
	} else if class := f.classPointer(open); class >= 0 {
		classDoc = f.docOpenBefore(class)
	}
	if classDoc < 0 {
		return false
	}
	for _, tag := range slvAliasTags {
		for _, a := range d.annotations(classDoc, tag) {
			v := a.node.value()
			switch v.kind {
			case "TypeAliasTagValueNode":
				if v.name == name {
					return true
				}
			case "TypeAliasImportTagValueNode":
				if v.importedAs == name || v.name == name {
					return true
				}
			}
		}
	}
	return false
}

// resolveConstantName is NamespaceHelper::resolveName for a constant.
func (d *slvDocs) resolveConstantName(name string, pointer int) string {
	if strings.HasPrefix(name, `\`) {
		return name
	}
	uses := d.useStatements().forPointer(pointer)
	canonical := strings.TrimLeft(name, `\`)
	if fqn, ok := uses["const "+canonical]; ok {
		return `\` + fqn
	}
	parts := strings.Split(canonical, `\`)
	if fqn, ok := uses["const "+parts[0]]; ok && len(parts) > 1 {
		return `\` + fqn + `\` + strings.Join(parts[1:], `\`)
	}
	if slvIsDefinedConstant(name) {
		return `\` + name
	}
	return d.namespaced(name, pointer)
}

func (d *slvDocs) namespaced(name string, pointer int) string {
	f := d.f
	for i := pointer - 1; i >= 0; i-- {
		if f.isKeyword(i, "namespace") {
			if n := f.nextEffective(i+1, len(f.toks)); n >= 0 {
				return `\` + f.content(n, n) + `\` + name
			}
			break
		}
	}
	return `\` + name
}

var slvDefinedConstants = sync.OnceValue(func() map[string]bool {
	m := map[string]bool{}
	for name := range strings.FieldsSeq(slvConstantNames) {
		m[name] = true
	}
	return m
})

// slvIsDefinedConstant is defined() in the PHP that runs ECS, for its core
// and bundled extensions' constants and those PHP_CodeSniffer defines.
func slvIsDefinedConstant(name string) bool {
	switch strings.ToLower(name) {
	case "true", "false", "null":
		return true
	}
	return slvDefinedConstants()[name]
}

const slvConstantNames = "" +
	`ABDAY_1 ABDAY_2 ABDAY_3 ABDAY_4 ABDAY_5 ABDAY_6 ABDAY_7 ABMON_1 ABMON_10 ABMON_11 ABMON_12 ABMON_2 ` +
	`ABMON_3 ABMON_4 ABMON_5 ABMON_6 ABMON_7 ABMON_8 ABMON_9 ALT_DIGITS AM_STR ARRAY_FILTER_USE_BOTH ` +
	`ARRAY_FILTER_USE_KEY ASSERT_ACTIVE ASSERT_BAIL ASSERT_CALLBACK ASSERT_EXCEPTION ASSERT_WARNING ` +
	`CASE_LOWER CASE_UPPER CHAR_MAX CODESET CONNECTION_ABORTED CONNECTION_NORMAL CONNECTION_TIMEOUT ` +
	`COUNT_NORMAL COUNT_RECURSIVE CREDITS_ALL CREDITS_DOCS CREDITS_FULLPAGE CREDITS_GENERAL ` +
	`CREDITS_GROUP CREDITS_MODULES CREDITS_QA CREDITS_SAPI CRNCYSTR CRYPT_BLOWFISH CRYPT_EXT_DES ` +
	`CRYPT_MD5 CRYPT_SALT_LENGTH CRYPT_SHA256 CRYPT_SHA512 CRYPT_STD_DES CURRENCY_SYMBOL DATE_ATOM ` +
	`DATE_COOKIE DATE_ISO8601 DATE_ISO8601_EXPANDED DATE_RFC1036 DATE_RFC1123 DATE_RFC2822 DATE_RFC3339 ` +
	`DATE_RFC3339_EXTENDED DATE_RFC7231 DATE_RFC822 DATE_RFC850 DATE_RSS DATE_W3C DAY_1 DAY_2 DAY_3 ` +
	`DAY_4 DAY_5 DAY_6 DAY_7 DEBUG_BACKTRACE_IGNORE_ARGS DEBUG_BACKTRACE_PROVIDE_OBJECT DECIMAL_POINT ` +
	`DEFAULT_INCLUDE_PATH DIRECTORY_SEPARATOR DNS_A DNS_A6 DNS_AAAA DNS_ALL DNS_ANY DNS_CAA DNS_CNAME ` +
	`DNS_HINFO DNS_MX DNS_NAPTR DNS_NS DNS_PTR DNS_SOA DNS_SRV DNS_TXT D_FMT D_T_FMT ENT_COMPAT ` +
	`ENT_DISALLOWED ENT_HTML401 ENT_HTML5 ENT_IGNORE ENT_NOQUOTES ENT_QUOTES ENT_SUBSTITUTE ENT_XHTML ` +
	`ENT_XML1 ERA ERA_D_FMT ERA_D_T_FMT ERA_T_FMT ERA_YEAR EXTR_IF_EXISTS EXTR_OVERWRITE EXTR_PREFIX_ALL ` +
	`EXTR_PREFIX_IF_EXISTS EXTR_PREFIX_INVALID EXTR_PREFIX_SAME EXTR_REFS EXTR_SKIP E_ALL ` +
	`E_COMPILE_ERROR E_COMPILE_WARNING E_CORE_ERROR E_CORE_WARNING E_DEPRECATED E_ERROR E_NOTICE E_PARSE ` +
	`E_RECOVERABLE_ERROR E_STRICT E_USER_DEPRECATED E_USER_ERROR E_USER_NOTICE E_USER_WARNING E_WARNING ` +
	`FALSE FILE_APPEND FILE_BINARY FILE_IGNORE_NEW_LINES FILE_NO_DEFAULT_CONTEXT FILE_SKIP_EMPTY_LINES ` +
	`FILE_TEXT FILE_USE_INCLUDE_PATH FILTER_CALLBACK FILTER_DEFAULT FILTER_FLAG_ALLOW_FRACTION ` +
	`FILTER_FLAG_ALLOW_HEX FILTER_FLAG_ALLOW_OCTAL FILTER_FLAG_ALLOW_SCIENTIFIC ` +
	`FILTER_FLAG_ALLOW_THOUSAND FILTER_FLAG_EMAIL_UNICODE FILTER_FLAG_EMPTY_STRING_NULL ` +
	`FILTER_FLAG_ENCODE_AMP FILTER_FLAG_ENCODE_HIGH FILTER_FLAG_ENCODE_LOW FILTER_FLAG_GLOBAL_RANGE ` +
	`FILTER_FLAG_HOSTNAME FILTER_FLAG_IPV4 FILTER_FLAG_IPV6 FILTER_FLAG_NONE ` +
	`FILTER_FLAG_NO_ENCODE_QUOTES FILTER_FLAG_NO_PRIV_RANGE FILTER_FLAG_NO_RES_RANGE ` +
	`FILTER_FLAG_PATH_REQUIRED FILTER_FLAG_QUERY_REQUIRED FILTER_FLAG_STRIP_BACKTICK ` +
	`FILTER_FLAG_STRIP_HIGH FILTER_FLAG_STRIP_LOW FILTER_FORCE_ARRAY FILTER_NULL_ON_FAILURE ` +
	`FILTER_REQUIRE_ARRAY FILTER_REQUIRE_SCALAR FILTER_SANITIZE_ADD_SLASHES FILTER_SANITIZE_EMAIL ` +
	`FILTER_SANITIZE_ENCODED FILTER_SANITIZE_FULL_SPECIAL_CHARS FILTER_SANITIZE_NUMBER_FLOAT ` +
	`FILTER_SANITIZE_NUMBER_INT FILTER_SANITIZE_SPECIAL_CHARS FILTER_SANITIZE_STRING ` +
	`FILTER_SANITIZE_STRIPPED FILTER_SANITIZE_URL FILTER_UNSAFE_RAW FILTER_VALIDATE_BOOL ` +
	`FILTER_VALIDATE_BOOLEAN FILTER_VALIDATE_DOMAIN FILTER_VALIDATE_EMAIL FILTER_VALIDATE_FLOAT ` +
	`FILTER_VALIDATE_INT FILTER_VALIDATE_IP FILTER_VALIDATE_MAC FILTER_VALIDATE_REGEXP ` +
	`FILTER_VALIDATE_URL FNM_CASEFOLD FNM_NOESCAPE FNM_PATHNAME FNM_PERIOD FRAC_DIGITS ` +
	`GLOB_AVAILABLE_FLAGS GLOB_BRACE GLOB_ERR GLOB_MARK GLOB_NOCHECK GLOB_NOESCAPE GLOB_NOSORT ` +
	`GLOB_ONLYDIR GROUPING HASH_HMAC HTML_ENTITIES HTML_SPECIALCHARS ICONV_IMPL ` +
	`ICONV_MIME_DECODE_CONTINUE_ON_ERROR ICONV_MIME_DECODE_STRICT ICONV_VERSION IMAGETYPE_AVIF ` +
	`IMAGETYPE_BMP IMAGETYPE_COUNT IMAGETYPE_GIF IMAGETYPE_ICO IMAGETYPE_IFF IMAGETYPE_JB2 IMAGETYPE_JP2 ` +
	`IMAGETYPE_JPC IMAGETYPE_JPEG IMAGETYPE_JPEG2000 IMAGETYPE_JPX IMAGETYPE_PNG IMAGETYPE_PSD ` +
	`IMAGETYPE_SWC IMAGETYPE_SWF IMAGETYPE_TIFF_II IMAGETYPE_TIFF_MM IMAGETYPE_UNKNOWN IMAGETYPE_WBMP ` +
	`IMAGETYPE_WEBP IMAGETYPE_XBM INF INFO_ALL INFO_CONFIGURATION INFO_CREDITS INFO_ENVIRONMENT ` +
	`INFO_GENERAL INFO_LICENSE INFO_MODULES INFO_VARIABLES INI_ALL INI_PERDIR INI_SCANNER_NORMAL ` +
	`INI_SCANNER_RAW INI_SCANNER_TYPED INI_SYSTEM INI_USER INPUT_COOKIE INPUT_ENV INPUT_GET INPUT_POST ` +
	`INPUT_SERVER INT_CURR_SYMBOL INT_FRAC_DIGITS JSON_BIGINT_AS_STRING JSON_ERROR_CTRL_CHAR ` +
	`JSON_ERROR_DEPTH JSON_ERROR_INF_OR_NAN JSON_ERROR_INVALID_PROPERTY_NAME JSON_ERROR_NONE ` +
	`JSON_ERROR_NON_BACKED_ENUM JSON_ERROR_RECURSION JSON_ERROR_STATE_MISMATCH JSON_ERROR_SYNTAX ` +
	`JSON_ERROR_UNSUPPORTED_TYPE JSON_ERROR_UTF16 JSON_ERROR_UTF8 JSON_FORCE_OBJECT JSON_HEX_AMP ` +
	`JSON_HEX_APOS JSON_HEX_QUOT JSON_HEX_TAG JSON_INVALID_UTF8_IGNORE JSON_INVALID_UTF8_SUBSTITUTE ` +
	`JSON_NUMERIC_CHECK JSON_OBJECT_AS_ARRAY JSON_PARTIAL_OUTPUT_ON_ERROR JSON_PRESERVE_ZERO_FRACTION ` +
	`JSON_PRETTY_PRINT JSON_THROW_ON_ERROR JSON_UNESCAPED_LINE_TERMINATORS JSON_UNESCAPED_SLASHES ` +
	`JSON_UNESCAPED_UNICODE LC_ALL LC_COLLATE LC_CTYPE LC_MESSAGES LC_MONETARY LC_NUMERIC LC_TIME ` +
	`LIBXML_BIGLINES LIBXML_COMPACT LIBXML_DOTTED_VERSION LIBXML_DTDATTR LIBXML_DTDLOAD LIBXML_DTDVALID ` +
	`LIBXML_ERR_ERROR LIBXML_ERR_FATAL LIBXML_ERR_NONE LIBXML_ERR_WARNING LIBXML_HTML_NODEFDTD ` +
	`LIBXML_HTML_NOIMPLIED LIBXML_LOADED_VERSION LIBXML_NOBLANKS LIBXML_NOCDATA LIBXML_NOEMPTYTAG ` +
	`LIBXML_NOENT LIBXML_NOERROR LIBXML_NONET LIBXML_NOWARNING LIBXML_NOXMLDECL LIBXML_NSCLEAN ` +
	`LIBXML_PARSEHUGE LIBXML_PEDANTIC LIBXML_RECOVER LIBXML_SCHEMA_CREATE LIBXML_VERSION LIBXML_XINCLUDE ` +
	`LOCK_EX LOCK_NB LOCK_SH LOCK_UN LOG_ALERT LOG_AUTH LOG_AUTHPRIV LOG_CONS LOG_CRIT LOG_CRON ` +
	`LOG_DAEMON LOG_DEBUG LOG_EMERG LOG_ERR LOG_INFO LOG_KERN LOG_LOCAL0 LOG_LOCAL1 LOG_LOCAL2 ` +
	`LOG_LOCAL3 LOG_LOCAL4 LOG_LOCAL5 LOG_LOCAL6 LOG_LOCAL7 LOG_LPR LOG_MAIL LOG_NDELAY LOG_NEWS ` +
	`LOG_NOTICE LOG_NOWAIT LOG_ODELAY LOG_PERROR LOG_PID LOG_SYSLOG LOG_USER LOG_UUCP LOG_WARNING ` +
	`MB_CASE_FOLD MB_CASE_FOLD_SIMPLE MB_CASE_LOWER MB_CASE_LOWER_SIMPLE MB_CASE_TITLE ` +
	`MB_CASE_TITLE_SIMPLE MB_CASE_UPPER MB_CASE_UPPER_SIMPLE MB_ONIGURUMA_VERSION MHASH_ADLER32 ` +
	`MHASH_CRC32 MHASH_CRC32B MHASH_CRC32C MHASH_FNV132 MHASH_FNV164 MHASH_FNV1A32 MHASH_FNV1A64 ` +
	`MHASH_GOST MHASH_HAVAL128 MHASH_HAVAL160 MHASH_HAVAL192 MHASH_HAVAL224 MHASH_HAVAL256 MHASH_JOAAT ` +
	`MHASH_MD2 MHASH_MD4 MHASH_MD5 MHASH_MURMUR3A MHASH_MURMUR3C MHASH_MURMUR3F MHASH_RIPEMD128 ` +
	`MHASH_RIPEMD160 MHASH_RIPEMD256 MHASH_RIPEMD320 MHASH_SHA1 MHASH_SHA224 MHASH_SHA256 MHASH_SHA384 ` +
	`MHASH_SHA512 MHASH_SNEFRU256 MHASH_TIGER MHASH_TIGER128 MHASH_TIGER160 MHASH_WHIRLPOOL MHASH_XXH128 ` +
	`MHASH_XXH3 MHASH_XXH32 MHASH_XXH64 MON_1 MON_10 MON_11 MON_12 MON_2 MON_3 MON_4 MON_5 MON_6 MON_7 ` +
	`MON_8 MON_9 MON_DECIMAL_POINT MON_GROUPING MON_THOUSANDS_SEP MT_RAND_MT19937 MT_RAND_PHP M_1_PI ` +
	`M_2_PI M_2_SQRTPI M_E M_EULER M_LN10 M_LN2 M_LNPI M_LOG10E M_LOG2E M_PI M_PI_2 M_PI_4 M_SQRT1_2 ` +
	`M_SQRT2 M_SQRT3 M_SQRTPI NAN NEGATIVE_SIGN NOEXPR NOSTR NULL N_CS_PRECEDES N_SEP_BY_SPACE ` +
	`N_SIGN_POSN PASSWORD_ARGON2I PASSWORD_ARGON2ID PASSWORD_ARGON2_DEFAULT_MEMORY_COST ` +
	`PASSWORD_ARGON2_DEFAULT_THREADS PASSWORD_ARGON2_DEFAULT_TIME_COST PASSWORD_ARGON2_PROVIDER ` +
	`PASSWORD_BCRYPT PASSWORD_BCRYPT_DEFAULT_COST PASSWORD_DEFAULT PATHINFO_ALL PATHINFO_BASENAME ` +
	`PATHINFO_DIRNAME PATHINFO_EXTENSION PATHINFO_FILENAME PATH_SEPARATOR PCRE_JIT_SUPPORT PCRE_VERSION ` +
	`PCRE_VERSION_MAJOR PCRE_VERSION_MINOR PEAR_EXTENSION_DIR PEAR_INSTALL_DIR PHP_BINARY PHP_BINDIR ` +
	`PHP_CLI_PROCESS_TITLE PHP_CODESNIFFER_CBF PHP_CODESNIFFER_VERBOSITY PHP_CONFIG_FILE_PATH ` +
	`PHP_CONFIG_FILE_SCAN_DIR PHP_DATADIR PHP_DEBUG PHP_EOL PHP_EXTENSION_DIR PHP_EXTRA_VERSION ` +
	`PHP_FD_SETSIZE PHP_FLOAT_DIG PHP_FLOAT_EPSILON PHP_FLOAT_MAX PHP_FLOAT_MIN PHP_INT_MAX PHP_INT_MIN ` +
	`PHP_INT_SIZE PHP_LIBDIR PHP_LOCALSTATEDIR PHP_MAJOR_VERSION PHP_MANDIR PHP_MAXPATHLEN ` +
	`PHP_MINOR_VERSION PHP_OS PHP_OS_FAMILY PHP_OUTPUT_HANDLER_CLEAN PHP_OUTPUT_HANDLER_CLEANABLE ` +
	`PHP_OUTPUT_HANDLER_CONT PHP_OUTPUT_HANDLER_DISABLED PHP_OUTPUT_HANDLER_END PHP_OUTPUT_HANDLER_FINAL ` +
	`PHP_OUTPUT_HANDLER_FLUSH PHP_OUTPUT_HANDLER_FLUSHABLE PHP_OUTPUT_HANDLER_PROCESSED ` +
	`PHP_OUTPUT_HANDLER_REMOVABLE PHP_OUTPUT_HANDLER_START PHP_OUTPUT_HANDLER_STARTED ` +
	`PHP_OUTPUT_HANDLER_STDFLAGS PHP_OUTPUT_HANDLER_WRITE PHP_PREFIX PHP_QUERY_RFC1738 PHP_QUERY_RFC3986 ` +
	`PHP_RELEASE_VERSION PHP_ROUND_HALF_DOWN PHP_ROUND_HALF_EVEN PHP_ROUND_HALF_ODD PHP_ROUND_HALF_UP ` +
	`PHP_SAPI PHP_SBINDIR PHP_SHLIB_SUFFIX PHP_SYSCONFDIR PHP_URL_FRAGMENT PHP_URL_HOST PHP_URL_PASS ` +
	`PHP_URL_PATH PHP_URL_PORT PHP_URL_QUERY PHP_URL_SCHEME PHP_URL_USER PHP_VERSION PHP_VERSION_ID ` +
	`PHP_ZTS PM_STR POSITIVE_SIGN PREG_BACKTRACK_LIMIT_ERROR PREG_BAD_UTF8_ERROR ` +
	`PREG_BAD_UTF8_OFFSET_ERROR PREG_GREP_INVERT PREG_INTERNAL_ERROR PREG_JIT_STACKLIMIT_ERROR ` +
	`PREG_NO_ERROR PREG_OFFSET_CAPTURE PREG_PATTERN_ORDER PREG_RECURSION_LIMIT_ERROR PREG_SET_ORDER ` +
	`PREG_SPLIT_DELIM_CAPTURE PREG_SPLIT_NO_EMPTY PREG_SPLIT_OFFSET_CAPTURE PREG_UNMATCHED_AS_NULL ` +
	`PSFS_ERR_FATAL PSFS_FEED_ME PSFS_FLAG_FLUSH_CLOSE PSFS_FLAG_FLUSH_INC PSFS_FLAG_NORMAL PSFS_PASS_ON ` +
	`P_CS_PRECEDES P_SEP_BY_SPACE P_SIGN_POSN RADIXCHAR SCANDIR_SORT_ASCENDING SCANDIR_SORT_DESCENDING ` +
	`SCANDIR_SORT_NONE SEEK_CUR SEEK_END SEEK_SET SORT_ASC SORT_DESC SORT_FLAG_CASE SORT_LOCALE_STRING ` +
	`SORT_NATURAL SORT_NUMERIC SORT_REGULAR SORT_STRING STDERR STDIN STDOUT STREAM_BUFFER_FULL ` +
	`STREAM_BUFFER_LINE STREAM_BUFFER_NONE STREAM_CAST_AS_STREAM STREAM_CAST_FOR_SELECT ` +
	`STREAM_CLIENT_ASYNC_CONNECT STREAM_CLIENT_CONNECT STREAM_CLIENT_PERSISTENT ` +
	`STREAM_CRYPTO_METHOD_ANY_CLIENT STREAM_CRYPTO_METHOD_ANY_SERVER STREAM_CRYPTO_METHOD_SSLv23_CLIENT ` +
	`STREAM_CRYPTO_METHOD_SSLv23_SERVER STREAM_CRYPTO_METHOD_SSLv2_CLIENT ` +
	`STREAM_CRYPTO_METHOD_SSLv2_SERVER STREAM_CRYPTO_METHOD_SSLv3_CLIENT ` +
	`STREAM_CRYPTO_METHOD_SSLv3_SERVER STREAM_CRYPTO_METHOD_TLS_CLIENT STREAM_CRYPTO_METHOD_TLS_SERVER ` +
	`STREAM_CRYPTO_METHOD_TLSv1_0_CLIENT STREAM_CRYPTO_METHOD_TLSv1_0_SERVER ` +
	`STREAM_CRYPTO_METHOD_TLSv1_1_CLIENT STREAM_CRYPTO_METHOD_TLSv1_1_SERVER ` +
	`STREAM_CRYPTO_METHOD_TLSv1_2_CLIENT STREAM_CRYPTO_METHOD_TLSv1_2_SERVER ` +
	`STREAM_CRYPTO_METHOD_TLSv1_3_CLIENT STREAM_CRYPTO_METHOD_TLSv1_3_SERVER STREAM_CRYPTO_PROTO_SSLv3 ` +
	`STREAM_CRYPTO_PROTO_TLSv1_0 STREAM_CRYPTO_PROTO_TLSv1_1 STREAM_CRYPTO_PROTO_TLSv1_2 ` +
	`STREAM_CRYPTO_PROTO_TLSv1_3 STREAM_FILTER_ALL STREAM_FILTER_READ STREAM_FILTER_WRITE ` +
	`STREAM_IGNORE_URL STREAM_IPPROTO_ICMP STREAM_IPPROTO_IP STREAM_IPPROTO_RAW STREAM_IPPROTO_TCP ` +
	`STREAM_IPPROTO_UDP STREAM_IS_URL STREAM_META_ACCESS STREAM_META_GROUP STREAM_META_GROUP_NAME ` +
	`STREAM_META_OWNER STREAM_META_OWNER_NAME STREAM_META_TOUCH STREAM_MKDIR_RECURSIVE STREAM_MUST_SEEK ` +
	`STREAM_NOTIFY_AUTH_REQUIRED STREAM_NOTIFY_AUTH_RESULT STREAM_NOTIFY_COMPLETED STREAM_NOTIFY_CONNECT ` +
	`STREAM_NOTIFY_FAILURE STREAM_NOTIFY_FILE_SIZE_IS STREAM_NOTIFY_MIME_TYPE_IS STREAM_NOTIFY_PROGRESS ` +
	`STREAM_NOTIFY_REDIRECTED STREAM_NOTIFY_RESOLVE STREAM_NOTIFY_SEVERITY_ERR ` +
	`STREAM_NOTIFY_SEVERITY_INFO STREAM_NOTIFY_SEVERITY_WARN STREAM_OOB STREAM_OPTION_BLOCKING ` +
	`STREAM_OPTION_READ_BUFFER STREAM_OPTION_READ_TIMEOUT STREAM_OPTION_WRITE_BUFFER STREAM_PEEK ` +
	`STREAM_PF_INET STREAM_PF_INET6 STREAM_PF_UNIX STREAM_REPORT_ERRORS STREAM_SERVER_BIND ` +
	`STREAM_SERVER_LISTEN STREAM_SHUT_RD STREAM_SHUT_RDWR STREAM_SHUT_WR STREAM_SOCK_DGRAM ` +
	`STREAM_SOCK_RAW STREAM_SOCK_RDM STREAM_SOCK_SEQPACKET STREAM_SOCK_STREAM STREAM_URL_STAT_LINK ` +
	`STREAM_URL_STAT_QUIET STREAM_USE_PATH STR_PAD_BOTH STR_PAD_LEFT STR_PAD_RIGHT SUNFUNCS_RET_DOUBLE ` +
	`SUNFUNCS_RET_STRING SUNFUNCS_RET_TIMESTAMP THOUSANDS_SEP THOUSEP TOKEN_PARSE TRUE T_ABSTRACT ` +
	`T_AMPERSAND_FOLLOWED_BY_VAR_OR_VARARG T_AMPERSAND_NOT_FOLLOWED_BY_VAR_OR_VARARG T_AND_EQUAL T_ARRAY ` +
	`T_ARRAY_CAST T_AS T_ATTRIBUTE T_BAD_CHARACTER T_BOOLEAN_AND T_BOOLEAN_OR T_BOOL_CAST T_BREAK ` +
	`T_CALLABLE T_CASE T_CATCH T_CLASS T_CLASS_C T_CLONE T_CLOSE_TAG T_COALESCE T_COALESCE_EQUAL ` +
	`T_COMMENT T_CONCAT_EQUAL T_CONST T_CONSTANT_ENCAPSED_STRING T_CONTINUE T_CURLY_OPEN T_DEC T_DECLARE ` +
	`T_DEFAULT T_DIR T_DIV_EQUAL T_DNUMBER T_DO T_DOC_COMMENT T_DOLLAR_OPEN_CURLY_BRACES T_DOUBLE_ARROW ` +
	`T_DOUBLE_CAST T_DOUBLE_COLON T_ECHO T_ELLIPSIS T_ELSE T_ELSEIF T_EMPTY T_ENCAPSED_AND_WHITESPACE ` +
	`T_ENDDECLARE T_ENDFOR T_ENDFOREACH T_ENDIF T_ENDSWITCH T_ENDWHILE T_END_HEREDOC T_ENUM T_EVAL ` +
	`T_EXIT T_EXTENDS T_FILE T_FINAL T_FINALLY T_FMT T_FMT_AMPM T_FN T_FOR T_FOREACH T_FUNCTION T_FUNC_C ` +
	`T_GLOBAL T_GOTO T_HALT_COMPILER T_IF T_IMPLEMENTS T_INC T_INCLUDE T_INCLUDE_ONCE T_INLINE_HTML ` +
	`T_INSTANCEOF T_INSTEADOF T_INTERFACE T_INT_CAST T_ISSET T_IS_EQUAL T_IS_GREATER_OR_EQUAL ` +
	`T_IS_IDENTICAL T_IS_NOT_EQUAL T_IS_NOT_IDENTICAL T_IS_SMALLER_OR_EQUAL T_LINE T_LIST T_LNUMBER ` +
	`T_LOGICAL_AND T_LOGICAL_OR T_LOGICAL_XOR T_MATCH T_METHOD_C T_MINUS_EQUAL T_MOD_EQUAL T_MUL_EQUAL ` +
	`T_NAMESPACE T_NAME_FULLY_QUALIFIED T_NAME_QUALIFIED T_NAME_RELATIVE T_NEW T_NS_C T_NS_SEPARATOR ` +
	`T_NULLSAFE_OBJECT_OPERATOR T_NUM_STRING T_OBJECT_CAST T_OBJECT_OPERATOR T_OPEN_TAG ` +
	`T_OPEN_TAG_WITH_ECHO T_OR_EQUAL T_PAAMAYIM_NEKUDOTAYIM T_PLUS_EQUAL T_POW T_POW_EQUAL T_PRINT ` +
	`T_PRIVATE T_PRIVATE_SET T_PROPERTY_C T_PROTECTED T_PROTECTED_SET T_PUBLIC T_PUBLIC_SET T_READONLY ` +
	`T_REQUIRE T_REQUIRE_ONCE T_RETURN T_SL T_SL_EQUAL T_SPACESHIP T_SR T_SR_EQUAL T_START_HEREDOC ` +
	`T_STATIC T_STRING T_STRING_CAST T_STRING_VARNAME T_SWITCH T_THROW T_TRAIT T_TRAIT_C T_TRY T_UNSET ` +
	`T_UNSET_CAST T_USE T_VAR T_VARIABLE T_WHILE T_WHITESPACE T_XOR_EQUAL T_YIELD T_YIELD_FROM ` +
	`UPLOAD_ERR_CANT_WRITE UPLOAD_ERR_EXTENSION UPLOAD_ERR_FORM_SIZE UPLOAD_ERR_INI_SIZE ` +
	`UPLOAD_ERR_NO_FILE UPLOAD_ERR_NO_TMP_DIR UPLOAD_ERR_OK UPLOAD_ERR_PARTIAL YESEXPR YESSTR ` +
	`ZEND_DEBUG_BUILD ZEND_THREAD_SAFE`
