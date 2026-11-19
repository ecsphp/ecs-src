package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/RequireExplicitAssertionSniff.php
//
// SlevomatRequireExplicitAssertion turns an inline "@var" doc comment into an
// assert() call (enableIntegerRanges, enableAdvancedStringTypes).
type SlevomatRequireExplicitAssertion struct {
	integerRanges       bool
	advancedStringTypes bool
}

func (SlevomatRequireExplicitAssertion) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\RequireExplicitAssertionSniff`
}

func (SlevomatRequireExplicitAssertion) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireExplicitAssertion{}.Name())
}

func (r SlevomatRequireExplicitAssertion) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["enableIntegerRanges"].(bool); ok {
		r.integerRanges = v
	}
	if v, ok := config["enableAdvancedStringTypes"].(bool); ok {
		r.advancedStringTypes = v
	}
	return r
}

func (r SlevomatRequireExplicitAssertion) Fix(s *tokens.Stream) bool {
	src := s.Render()
	crashed := false
	changed := slvRun(s, func(f *slvFile) {
		var scopes []int
		f.eachToken(f.assertionIsDocOpen, func(open int) int {
			if crashed {
				return len(f.toks)
			}
			closeIdx := f.assertionDocClose(open)
			code := f.assertionFirstOnNextLine(closeIdx)
			if code < 0 || !f.assertionIsCode(code) {
				code = f.assertionFirstOnPreviousLine(open)
				if code < 0 || !f.assertionIsCode(code) {
					return 0
				}
			}
			annotations := f.assertionVarAnnotations(open, closeIdx)
			for _, a := range slices.Backward(annotations) {
				if a.invalid || a.variable == "" || !r.isValid(a.typ) {
					continue
				}
				assertion := r.createAssert(a.variable, a.typ)
				if assertion == "" {
					continue
				}
				if scopes == nil {
					scopes = slvcFunctionScopes(f)
				}
				add, indentation := f.assertionTarget(code, a.variable, open, scopes)
				if add == assertionNoSemicolon {
					// upstream passes null to ScopeHelper::isInSameScope and
					// the TypeError leaves the whole file unfixed
					crashed = true
					return len(f.toks)
				}
				if add < 0 {
					continue
				}
				f.beginChangeset()
				f.removeBetweenIncluding(a.start, a.end)
				useful := false
				for i := open + 1; i < closeIdx; i++ {
					v := strings.Trim(f.tokenContent(i), " \t\n\r\x00\x0B")
					if v != "" && v != "*" {
						useful = true
						break
					}
				}
				before := -1
				for i := open - 1; i >= 0; i-- {
					if f.toks[i].kind == token.Whitespace && f.toks[i].content == f.eol {
						before = i
						break
					}
				}
				after := -1
				for i := closeIdx + 1; i < len(f.toks); i++ {
					if f.toks[i].kind == token.Whitespace && f.toks[i].content == f.eol {
						after = i
						break
					}
				}
				if !useful {
					f.removeBetweenIncluding(before+1, after)
				}
				if add < closeIdx && after >= 0 && after+1 < len(f.toks) {
					f.addBefore(after+1, indentation+assertion+f.eol)
				} else {
					f.add(add, f.eol+indentation+assertion)
				}
				f.endChangeset()
			}
			return 0
		})
	})
	if crashed {
		if changed {
			s.ReplaceRange(0, s.Len()-1, lexer.Lex(src))
		}
		return false
	}
	return changed
}

const assertionNoSemicolon = -2

// assertionTarget returns the token the assertion goes after and its
// indentation, or -1 when the code does not assign the variable.
func (f *slvFile) assertionTarget(code int, variable string, open int, scopes []int) (int, string) {
	n := len(f.toks)
	switch {
	case f.isVariable(code):
		if !f.is(f.nextEffective(code+1, n), "=") || f.toks[code].content != variable {
			return -1, ""
		}
		return f.assertionNextSemicolon(code, code+1, scopes), f.indentation(open)
	case f.isKeyword(code, "list"):
		paren := f.nextEffective(code+1, n)
		if !f.is(paren, "(") || f.toks[paren].closer < 0 || f.assertionFindVariable(variable, paren+1, f.toks[paren].closer) < 0 {
			return -1, ""
		}
		return f.assertionNextSemicolon(code, code+1, scopes), f.indentation(open)
	case f.isShortArrayOpen(code):
		closeIdx := f.toks[code].closer
		if !f.is(f.nextEffective(closeIdx+1, n), "=") || f.assertionFindVariable(variable, code+1, closeIdx) < 0 {
			return -1, ""
		}
		return f.assertionNextSemicolon(code, closeIdx+1, scopes), f.indentation(open)
	}
	paren := f.nextEffective(code+1, n)
	if !f.is(paren, "(") || f.toks[paren].closer < 0 {
		return -1, ""
	}
	closeIdx := f.toks[paren].closer
	if f.isKeyword(code, "while") {
		v := f.assertionFindVariable(variable, paren+1, closeIdx)
		if v < 0 || !f.is(f.nextEffective(v+1, n), "=") {
			return -1, ""
		}
	} else {
		as := -1
		for i := paren + 1; i < closeIdx; i++ {
			if f.isKeyword(i, "as") {
				as = i
				break
			}
		}
		if as < 0 || f.assertionFindVariable(variable, as+1, closeIdx) < 0 {
			return -1, ""
		}
	}
	opener := f.nextEffective(closeIdx+1, n)
	if !f.is(opener, "{") && !f.is(opener, ":") {
		return -1, ""
	}
	return opener, f.indentation(code) + "    "
}

func (f *slvFile) assertionFindVariable(name string, start, end int) int {
	for i := start; i < end; i++ {
		if f.isVariable(i) && f.toks[i].content == name {
			return i
		}
	}
	return -1
}

// assertionNextSemicolon is getNextSemicolonInSameScope, or -1.
func (f *slvFile) assertionNextSemicolon(scope, from int, scopes []int) int {
	for i := from; i < len(f.toks); i++ {
		if f.is(i, ";") && scopes[i] == scopes[scope] {
			return i
		}
	}
	return assertionNoSemicolon
}

func (f *slvFile) assertionIsCode(i int) bool {
	return f.isVariable(i) || f.isKeyword(i, "foreach") || f.isKeyword(i, "while") ||
		f.isKeyword(i, "list") || f.isShortArrayOpen(i)
}

func (f *slvFile) assertionIsDocOpen(i int) bool {
	return f.toks[i].kind == token.DocComment && strings.HasPrefix(f.toks[i].orig, "/**") &&
		(i == 0 || f.toks[i-1].kind != token.DocComment)
}

func (f *slvFile) assertionDocClose(open int) int {
	closeIdx := open
	for closeIdx+1 < len(f.toks) && f.toks[closeIdx+1].kind == token.DocComment {
		closeIdx++
	}
	return closeIdx
}

// assertionIsWhitespace tells T_WHITESPACE and T_DOC_COMMENT_WHITESPACE.
func (f *slvFile) assertionIsWhitespace(i int) bool {
	t := f.toks[i]
	return t.kind == token.Whitespace || t.kind == token.DocComment && t.orig != "" && strings.TrimSpace(t.orig) == ""
}

func (f *slvFile) assertionIsNewline(i int) bool {
	return f.assertionIsWhitespace(i) && f.toks[i].content == f.eol
}

// assertionFirstOnNextLine is TokenHelper::findFirstNonWhitespaceOnNextLine.
func (f *slvFile) assertionFirstOnNextLine(i int) int {
	nl := i
	for nl < len(f.toks) && !f.assertionIsNewline(nl) {
		nl++
	}
	if nl >= len(f.toks) {
		return -1
	}
	next := nl + 1
	for next < len(f.toks) && f.assertionIsWhitespace(next) {
		next++
	}
	if next < len(f.toks) && f.toks[i].line == f.toks[next].line-1 {
		return next
	}
	return -1
}

// assertionFirstOnPreviousLine is TokenHelper::findFirstNonWhitespaceOnPreviousLine.
func (f *slvFile) assertionFirstOnPreviousLine(i int) int {
	nl := i
	for nl >= 0 && !f.assertionIsNewline(nl) {
		nl--
	}
	if nl < 0 {
		return -1
	}
	before := nl - 1
	for before >= 0 && !f.assertionIsNewline(before) {
		before--
	}
	next := before + 1
	if before < 0 {
		next = 1
	}
	for next < len(f.toks) && f.assertionIsWhitespace(next) {
		next++
	}
	if next < len(f.toks) && f.toks[i].line == f.toks[next].line+1 {
		return next
	}
	return -1
}

// slveaAnnotation is a parsed "@var" annotation and its tokens.
type slveaAnnotation struct {
	start, end int
	invalid    bool
	variable   string
	typ        *slveaType
}

const (
	slveaIdentifier = iota
	slveaThis
	slveaIntRange
	slveaUnion
	slveaIntersection
	slveaOther
)

// slveaType is the part of phpdoc-parser's type tree the sniff reads.
type slveaType struct {
	kind  int
	name  string      // identifier name
	args  [2]string   // int<min, max> bounds: "min", "max" or an integer
	types []slveaType // union or intersection members
}

// assertionVarAnnotations is AnnotationHelper::getAnnotations(..., '@var')
// for the doc comment open..closeIdx.
func (f *slvFile) assertionVarAnnotations(open, closeIdx int) []slveaAnnotation {
	var tags []int
	for i := open + 1; i < closeIdx; i++ {
		if strings.HasPrefix(f.toks[i].orig, "@") {
			tags = append(tags, i)
		}
	}
	var out []slveaAnnotation
	for n, tag := range tags {
		if f.toks[tag].orig != "@var" {
			continue
		}
		limit := closeIdx
		if n+1 < len(tags) {
			limit = tags[n+1]
		}
		end := tag
		for i := tag + 1; i < limit; i++ {
			if f.assertionIsDocString(i) {
				end = i
			}
		}
		a := slveaAnnotation{start: tag, end: end}
		value := ""
		if tag+2 < limit && f.assertionIsWhitespace(tag+1) && f.assertionIsDocString(tag+2) {
			value = f.toks[tag+2].orig
		}
		if value == "" {
			a.invalid = true
		} else {
			a.typ, a.variable, a.invalid = slveaParseVarValue(value)
		}
		out = append(out, a)
	}
	return out
}

func (f *slvFile) assertionIsDocString(i int) bool {
	v := f.toks[i].orig
	return f.toks[i].kind == token.DocComment && strings.TrimSpace(v) != "" && v != "*" &&
		!strings.HasPrefix(v, "@") && !strings.HasPrefix(v, "/**") && !strings.HasSuffix(v, "*/")
}

var (
	slveaIdentRe    = regexp.MustCompile(`^(?:\\?[a-zA-Z_\x80-\xff][0-9a-zA-Z_\x80-\xff-]*)+`)
	slveaVariableRe = regexp.MustCompile(`^\$[a-zA-Z_\x80-\xff][0-9a-zA-Z_\x80-\xff]*`)
	slveaIntegerRe  = regexp.MustCompile(`^-?(?:0[bB][01]+(?:_[01]+)*|0[oO][0-7]+(?:_[0-7]+)*|0[xX][0-9a-fA-F]+(?:_[0-9a-fA-F]+)*|[0-9]+(?:_[0-9]+)*)`)
)

// slveaParseVarValue parses "Type $variable description" of an "@var" tag.
func slveaParseVarValue(v string) (*slveaType, string, bool) {
	p := &slveaParser{s: v}
	typ := p.parseType()
	if typ == nil {
		return nil, "", true
	}
	p.skipSpaces()
	variable := ""
	if strings.HasPrefix(p.s[p.i:], "$this") && !slveaIsIdentChar(p.peekAt(5)) {
		variable = "$this"
		p.i += 5
	} else if m := slveaVariableRe.FindString(p.s[p.i:]); m != "" {
		variable = m
		p.i += len(m)
	}
	return typ, variable, false
}

type slveaParser struct {
	s string
	i int
}

func (p *slveaParser) peekAt(n int) byte {
	if p.i+n < len(p.s) {
		return p.s[p.i+n]
	}
	return 0
}

func (p *slveaParser) skipSpaces() {
	for p.i < len(p.s) && (p.s[p.i] == ' ' || p.s[p.i] == '\t') {
		p.i++
	}
}

// parseType parses an atomic type or a union or intersection of them, or
// returns nil on a syntax error. Types the sniff never asserts parse as other.
func (p *slveaParser) parseType() *slveaType {
	p.skipSpaces()
	if p.i < len(p.s) && p.s[p.i] == '?' {
		p.i++
		if p.parseAtomic() == nil {
			return nil
		}
		return &slveaType{kind: slveaOther}
	}
	first := p.parseAtomic()
	if first == nil {
		return nil
	}
	save := p.i
	p.skipSpaces()
	if p.i >= len(p.s) || (p.s[p.i] != '|' && p.s[p.i] != '&') {
		p.i = save
		return first
	}
	op := p.s[p.i]
	kind := slveaUnion
	if op == '&' {
		kind = slveaIntersection
	}
	t := &slveaType{kind: kind, types: []slveaType{*first}}
	for {
		save = p.i
		p.skipSpaces()
		if p.i >= len(p.s) || p.s[p.i] != op {
			p.i = save
			break
		}
		p.i++
		p.skipSpaces()
		next := p.parseAtomic()
		if next == nil {
			return nil
		}
		t.types = append(t.types, *next)
	}
	save = p.i
	p.skipSpaces()
	if p.i < len(p.s) && (p.s[p.i] == '|' || p.s[p.i] == '&') {
		return nil
	}
	p.i = save
	return t
}

func (p *slveaParser) parseAtomic() *slveaType {
	rest := p.s[p.i:]
	var t *slveaType
	switch {
	case strings.HasPrefix(rest, "$this") && !slveaIsIdentChar(p.peekAt(5)):
		p.i += 5
		t = &slveaType{kind: slveaThis, name: "$this"}
	case slveaIdentRe.MatchString(rest):
		name := slveaIdentRe.FindString(rest)
		p.i += len(name)
		t = &slveaType{kind: slveaIdentifier, name: name}
		if p.i < len(p.s) && p.s[p.i] == '<' {
			return p.parseGeneric(name)
		}
	case rest != "" && rest[0] == '(':
		// phpdoc-parser returns the type inside parentheses as is
		p.i++
		inner := p.parseType()
		p.skipSpaces()
		if inner == nil || p.i >= len(p.s) || p.s[p.i] != ')' {
			return nil
		}
		p.i++
		if p.i < len(p.s) && p.s[p.i] == '[' {
			return p.skipOther()
		}
		return inner
	case rest != "" && strings.ContainsRune("[{'\"", rune(rest[0])), slveaIntegerRe.MatchString(rest):
		return p.skipOther()
	default:
		return nil
	}
	if p.i < len(p.s) && strings.ContainsRune("[({:<", rune(p.s[p.i])) {
		return p.skipOther()
	}
	return t
}

// parseGeneric parses int<min, max>; other generics are "other".
func (p *slveaParser) parseGeneric(name string) *slveaType {
	start := p.i
	depth := 0
	end := -1
	for j := p.i; j < len(p.s); j++ {
		switch p.s[j] {
		case '<':
			depth++
		case '>':
			depth--
			if depth == 0 {
				end = j
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return nil
	}
	p.i = end + 1
	if p.i < len(p.s) && strings.ContainsRune("[({:<", rune(p.s[p.i])) {
		return p.skipOther()
	}
	if name != "int" {
		return &slveaType{kind: slveaOther}
	}
	parts := strings.Split(p.s[start+1:end], ",")
	if len(parts) != 2 {
		return &slveaType{kind: slveaOther}
	}
	t := &slveaType{kind: slveaIntRange, name: name}
	for k, part := range parts {
		part = strings.TrimSpace(part)
		if part != "min" && part != "max" && slveaIntegerRe.FindString(part) != part {
			return &slveaType{kind: slveaOther}
		}
		t.args[k] = part
	}
	return t
}

// skipOther consumes a type the sniff never asserts, up to the next space
// outside brackets.
func (p *slveaParser) skipOther() *slveaType {
	depth := 0
	for p.i < len(p.s) {
		c := p.s[p.i]
		switch {
		case strings.ContainsRune("([{<", rune(c)):
			depth++
		case strings.ContainsRune(")]}>", rune(c)):
			depth--
		case depth == 0 && (slveaIsSpace(c) || c == '|' || c == '&'):
			return &slveaType{kind: slveaOther}
		}
		p.i++
	}
	return &slveaType{kind: slveaOther}
}

func slveaIsSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func slveaIsIdentChar(c byte) bool {
	return c == '_' || c == '-' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}

func (r SlevomatRequireExplicitAssertion) isValid(t *slveaType) bool {
	if t.kind == slveaUnion || t.kind == slveaIntersection {
		for k := range t.types {
			if !r.isValidAtomic(&t.types[k]) {
				return false
			}
		}
		return true
	}
	return r.isValidAtomic(t)
}

func (r SlevomatRequireExplicitAssertion) isValidAtomic(t *slveaType) bool {
	switch t.kind {
	case slveaThis, slveaIdentifier:
		return true
	case slveaIntRange:
		return r.integerRanges
	}
	return false
}

func (r SlevomatRequireExplicitAssertion) createAssert(variable string, t *slveaType) string {
	if t.kind != slveaUnion && t.kind != slveaIntersection {
		conditions := r.createConditions(variable, t)
		if len(conditions) == 0 {
			return ""
		}
		return `\assert(` + strings.Join(conditions, " || ") + ");"
	}
	var conditions []string
	for k := range t.types {
		inner := r.createConditions(variable, &t.types[k])
		if len(inner) == 0 {
			return ""
		}
		conditions = append(conditions, inner...)
	}
	operator := "||"
	if t.kind == slveaIntersection {
		operator = "&&"
	}
	var formatted []string
	seen := map[string]bool{}
	for _, c := range conditions {
		if seen[c] {
			continue
		}
		seen[c] = true
		if operator == "||" && strings.Contains(c, "&&") {
			c = "(" + c + ")"
		}
		formatted = append(formatted, c)
	}
	return `\assert(` + strings.Join(formatted, " "+operator+" ") + ");"
}

var (
	slveaSimpleTypes = map[string]bool{
		"int": true, "integer": true, "false": true, "float": true, "string": true, "bool": true,
		"boolean": true, "callable": true, "self": true, "array": true, "iterable": true, "void": true, "never": true,
	}
	slveaUnofficialTypes = map[string]bool{
		"null": true, "mixed": true, "scalar": true, "numeric": true, "true": true, "object": true,
		"resource": true, "static": true, "$this": true, "array-key": true, "list": true,
		"non-empty-array": true, "non-empty-list": true, "empty": true, "positive-int": true,
		"non-positive-int": true, "negative-int": true, "non-negative-int": true, "literal-int": true,
		"int-mask": true, "min": true, "max": true, "callable-array": true, "callable-string": true,
	}
	slveaClassStringRe = regexp.MustCompile(`^(?:class|trait|enum)-string$`)
)

func (r SlevomatRequireExplicitAssertion) createConditions(v string, t *slveaType) []string {
	if t.kind == slveaIntRange {
		conditions := []string{`\is_int(` + v + ")"}
		if t.args[0] != "min" && t.args[0] != "max" {
			conditions = append(conditions, v+" >= "+t.args[0])
		}
		if t.args[1] != "min" && t.args[1] != "max" {
			conditions = append(conditions, v+" <= "+t.args[1])
		}
		return []string{strings.Join(conditions, " && ")}
	}
	if t.kind == slveaThis {
		return []string{v + " instanceof $this"}
	}
	name := t.name
	switch {
	case name == "self":
		return []string{v + " instanceof " + name}
	case name == "static":
		return []string{v + " instanceof static"}
	case name == "true" || name == "false" || name == "null":
		return []string{v + " === " + name}
	case name == "mixed" || name == "void" || name == "never" || name == "never-return" ||
		name == "never-returns" || name == "no-return":
		return nil
	case slveaSimpleTypes[name]:
		short := name
		switch name {
		case "integer":
			short = "int"
		case "boolean":
			short = "bool"
		}
		return []string{`\is_` + short + "(" + v + ")"}
	case name == "resource" || name == "object":
		return []string{`\is_` + name + "(" + v + ")"}
	case name == "numeric":
		return []string{`\is_numeric(` + v + ")"}
	case name == "scalar":
		return []string{`\is_int(` + v + ")", `\is_float(` + v + ")", `\is_bool(` + v + ")", `\is_string(` + v + ")"}
	}
	if r.integerRanges {
		switch name {
		case "positive-int", "non-negative-int":
			return []string{`\is_int(` + v + ") && " + v + " > 0"}
		case "negative-int", "non-positive-int":
			return []string{`\is_int(` + v + ") && " + v + " < 0"}
		case "literal-int":
			return []string{`\is_int(` + v + ")"}
		}
	}
	if r.advancedStringTypes && strings.HasSuffix(name, "-string") && !slveaClassStringRe.MatchString(name) {
		conditions := []string{`\is_string(` + v + ")"}
		lower := strings.ToLower(name)
		switch {
		case name == "callable-string":
			conditions = append(conditions, `\is_callable(`+v+")")
		case name == "numeric-string":
			conditions = append(conditions, `\is_numeric(`+v+")")
		case strings.HasPrefix(lower, "non-empty-"):
			conditions = append(conditions, v+" !== ''")
		case strings.HasPrefix(lower, "non-falsy-"):
			conditions = append(conditions, "(bool) "+v+" === true")
		}
		return []string{strings.Join(conditions, " && ")}
	}
	if slveaUnofficialTypes[name] || strings.HasSuffix(strings.ToLower(name), "-string") {
		return nil
	}
	return []string{v + " instanceof " + name}
}
