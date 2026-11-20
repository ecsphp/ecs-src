package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Numbers/DisallowNumericLiteralSeparatorSniff.php
//
// SlevomatDisallowNumericLiteralSeparator removes the "_" separators of
// numeric literals.
type SlevomatDisallowNumericLiteralSeparator struct{}

func (SlevomatDisallowNumericLiteralSeparator) Name() string {
	return `SlevomatCodingStandard\Sniffs\Numbers\DisallowNumericLiteralSeparatorSniff`
}

func (SlevomatDisallowNumericLiteralSeparator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowNumericLiteralSeparator{}.Name())
}

func (SlevomatDisallowNumericLiteralSeparator) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i, t := range f.toks {
			if t.kind != token.Number || !strings.Contains(t.content, "_") {
				continue
			}
			f.beginChangeset()
			f.replace(i, strings.ReplaceAll(t.content, "_", ""))
			f.endChangeset()
		}
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Whitespaces/DuplicateSpacesSniff.php
//
// SlevomatDuplicateSpaces collapses runs of spaces inside a line into one.
type SlevomatDuplicateSpaces struct {
	ignoreBeforeAssignment, ignoreInAnnotation, ignoreInComment, ignoreInParameters, ignoreInMatch bool
}

func (SlevomatDuplicateSpaces) Name() string {
	return `SlevomatCodingStandard\Sniffs\Whitespaces\DuplicateSpacesSniff`
}

func (SlevomatDuplicateSpaces) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDuplicateSpaces{}.Name())
}

func (r SlevomatDuplicateSpaces) WithConfig(config map[string]any) fixer.Fixer {
	for key, field := range map[string]*bool{
		"ignoreSpacesBeforeAssignment": &r.ignoreBeforeAssignment,
		"ignoreSpacesInAnnotation":     &r.ignoreInAnnotation,
		"ignoreSpacesInComment":        &r.ignoreInComment,
		"ignoreSpacesInParameters":     &r.ignoreInParameters,
		"ignoreSpacesInMatch":          &r.ignoreInMatch,
	} {
		if v, ok := config[key].(bool); ok {
			*field = v
		}
	}
	return r
}

var (
	slvDuplicateSpaces = regexp.MustCompile(` {2,}`)
	// Tokens::ASSIGNMENT_TOKENS, T_DOUBLE_ARROW included
	slvAssignmentTokens = phpcsSet("=", "&=", "|=", ".=", "/=", "-=", "**=", "%=", "*=", "+=", "^=", "=>", "<<=", ">>=", "??=")
)

func (r SlevomatDuplicateSpaces) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		var sc *phpcsScope
		if r.ignoreBeforeAssignment || r.ignoreInParameters || r.ignoreInMatch {
			sc = newPhpcsScope(f)
		}
		for i, t := range f.toks {
			code := t.kind == token.Whitespace
			if !code && (t.kind != token.DocComment || t.doc != slvDocWhitespace && t.doc != slvDocString) {
				continue
			}
			if t.column == 1 || t.content == f.eol || t.doc == slvDocString && slvIsPhpcsAnnotation(t.content) {
				continue
			}
			if code && r.skipCode(f, sc, i) || !code && r.skipDoc(f, i) {
				continue
			}
			if !slvDuplicateSpaces.MatchString(t.content) {
				continue
			}
			f.beginChangeset()
			f.replace(i, slvDuplicateSpaces.ReplaceAllString(t.content, " "))
			f.endChangeset()
		}
	})
}

func (r SlevomatDuplicateSpaces) skipCode(f *slvFile, sc *phpcsScope, i int) bool {
	after := f.nextNonWhitespace(i+1, len(f.toks))
	if after < 0 {
		return false
	}
	if r.ignoreBeforeAssignment && slvAssignmentTokens[sc.code[after]] {
		return true
	}
	if r.ignoreInParameters {
		v := after
		if f.is(v, "&") || f.is(v, "...") {
			v = f.nextNonWhitespace(v+1, len(f.toks))
		}
		if f.isVariable(v) && slvIsParameter(f, sc, v) {
			return true
		}
	}
	return r.ignoreInMatch && sc.code[after] == "match_arrow"
}

func (r SlevomatDuplicateSpaces) skipDoc(f *slvFile, i int) bool {
	if r.ignoreInComment {
		return true
	}
	if i > 0 && f.toks[i-1].doc == slvDocStar && i+1 < len(f.toks) && f.toks[i+1].doc == slvDocString &&
		!slvIsPhpcsAnnotation(f.toks[i+1].content) {
		return true
	}
	if r.ignoreInAnnotation {
		for j := i - 1; j >= 0; j-- {
			if d := f.toks[j].doc; d == slvDocOpen || d == slvDocTag {
				return d == slvDocTag && i+1 < len(f.toks) && f.toks[i+1].doc != slvDocClose
			}
		}
	}
	return false
}

// slvIsPhpcsAnnotation tells a docblock string PHP_CodeSniffer's tokenizer
// turns into a T_PHPCS_* token ("@phpcs:disable Foo").
func slvIsPhpcsAnnotation(content string) bool {
	text := strings.ToLower(strings.TrimLeft(content, " \t/*#"))
	text = strings.TrimPrefix(text, "@")
	for _, prefix := range []string{"phpcs:set", "phpcs:ignorefile", "phpcs:disable", "phpcs:enable", "phpcs:ignore"} {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

// slvIsParameter is ParameterHelper::isParameter: the innermost parentheses
// holding v belong to a function, closure or arrow function.
func slvIsParameter(f *slvFile, sc *phpcsScope, v int) bool {
	open := sc.nested[v]
	if open < 0 {
		return false
	}
	owner := sc.pOwner[open]
	if owner < 0 {
		p := f.prevEffective(open - 1)
		if f.is(p, "&") {
			p = f.prevEffective(p - 1)
		}
		return p >= 0 && sc.code[p] == "fn" && sc.opener[p] > open
	}
	return sc.code[owner] == "function" || sc.code[owner] == "fn"
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Variables/UselessVariableSniff.php
//
// SlevomatUselessVariable returns the value right away instead of assigning
// it to a variable that is returned by the next statement.
type SlevomatUselessVariable struct{}

func (SlevomatUselessVariable) Name() string {
	return `SlevomatCodingStandard\Sniffs\Variables\UselessVariableSniff`
}

func (SlevomatUselessVariable) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessVariable{}.Name())
}

var slvCompoundAssignments = map[string]string{
	"+=": "+", "-=": "-", "*=": "*", "/=": "/", "**=": "**", "%=": "%",
	"&=": "&", "|=": "|", "^=": "^", "<<=": "<<", ">>=": ">>", ".=": ".",
}

func (SlevomatUselessVariable) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		u := newSlvUselessVariable(f)
		for i := range f.toks {
			if f.isKeyword(i, "return") {
				u.process(i)
			}
		}
	})
}

type slvUselessVariable struct {
	f      *slvFile
	sc     *phpcsScope
	conds  [][]phpcsCond
	levels []int
	pOpen  map[int]int // parenthesis_opener of a parenthesis owner
}

func newSlvUselessVariable(f *slvFile) *slvUselessVariable {
	sc := newPhpcsScope(f)
	conds, levels := sc.levelMap()
	u := &slvUselessVariable{f: f, sc: sc, conds: conds, levels: levels, pOpen: map[int]int{}}
	for open, owner := range sc.pOwner {
		if owner >= 0 {
			u.pOpen[owner] = open
		}
	}
	for i, code := range sc.code {
		if code == "fn" && sc.opener[i] > i {
			if open := f.nextEffective(i+1, len(f.toks)); f.is(open, "(") {
				u.pOpen[i] = open
			} else if f.is(open, "&") {
				u.pOpen[i] = f.nextEffective(open+1, len(f.toks))
			}
		}
	}
	return u
}

func (u *slvUselessVariable) isFunction(i int) bool {
	c := u.sc.code[i]
	return c == "function" || c == "fn"
}

// scope is ScopeHelper::isInSameScope's scope of a token.
func (u *slvUselessVariable) scope(i int) int {
	for _, c := range slices.Backward(u.conds[i]) {
		if !c.caseIndex && u.isFunction(c.key) {
			return u.levels[c.key] + 1
		}
	}
	return 0
}

func (u *slvUselessVariable) sameScope(a, b int) bool { return u.scope(a) == u.scope(b) }

func (u *slvUselessVariable) process(ret int) {
	f := u.f
	n := len(f.toks)
	v := f.nextEffective(ret+1, n)
	if !f.isVariable(v) {
		return
	}
	retSemi := f.nextEffective(v+1, n)
	if !f.is(retSemi, ";") {
		return
	}
	name := f.toks[v].content
	if fn := u.functionOf(v); fn >= 0 {
		if f.is(f.nextEffective(fn+1, n), "&") || u.isStaticVariable(fn, v, name) || u.isParameter(fn, name) {
			return
		}
	}
	prev := u.previousVariable(ret, name)
	if prev < 0 || !u.isAssignment(prev) || u.inControlStructure(prev) || u.inFunctionCall(prev) ||
		u.hasVarAnnotation(prev) || u.hasAnotherAssignmentBefore(prev, name) || !u.nearby(prev, ret) {
		return
	}
	before := f.prevEffective(prev - 1)
	if !f.is(before, ";") && !f.is(before, "{") && !f.is(before, "}") && f.nextEffective(retSemi+1, n) >= 0 {
		return
	}
	assignment := f.nextEffective(prev+1, n)
	semi := u.semicolon(prev)
	f.beginChangeset()
	if f.is(assignment, "=") {
		f.change(prev, assignment, "return")
	} else {
		f.addBefore(prev, "return ")
		f.replace(assignment, slvCompoundAssignments[f.toks[assignment].orig])
	}
	f.removeBetweenIncluding(semi+1, retSemi)
	f.endChangeset()
}

func (u *slvUselessVariable) functionOf(i int) int {
	for _, c := range slices.Backward(u.conds[i]) {
		if !c.caseIndex && u.isFunction(c.key) {
			return c.key
		}
	}
	return -1
}

func (u *slvUselessVariable) isStaticVariable(fn, v int, name string) bool {
	f := u.f
	for i := u.sc.opener[fn] + 1; i < v; i++ {
		if f.isVariable(i) && f.toks[i].content == name && f.isKeyword(f.prevEffective(i-1), "static") {
			return true
		}
	}
	return false
}

func (u *slvUselessVariable) isParameter(fn int, name string) bool {
	open, ok := u.pOpen[fn]
	if !ok {
		return false
	}
	f := u.f
	for i := open + 1; i < f.toks[open].closer; i++ {
		if f.isVariable(i) && f.toks[i].content == name {
			return true
		}
	}
	return false
}

// previousVariable is findPreviousVariablePointer.
func (u *slvUselessVariable) previousVariable(pointer int, name string) int {
	f := u.f
	for i := pointer - 1; i >= 0; i-- {
		if u.isFunction(i) {
			// a function without a body has no scope_opener, read as 0
			opener := max(u.sc.opener[i], 0)
			if u.sameScope(opener+1, pointer) {
				return -1
			}
		}
		if !f.isVariable(i) || f.toks[i].content != name || f.is(f.prevEffective(i-1), "::") {
			continue
		}
		if u.sameScope(i, pointer) {
			return i
		}
	}
	return -1
}

func (u *slvUselessVariable) isAssignment(v int) bool {
	a := u.f.nextEffective(v+1, len(u.f.toks))
	if a < 0 || u.f.toks[a].kind != token.Punct {
		return false
	}
	_, compound := slvCompoundAssignments[u.f.toks[a].orig]
	return compound || u.f.is(a, "=")
}

func (u *slvUselessVariable) inControlStructure(v int) bool {
	f := u.f
	for i := v - 1; i >= 0; i-- {
		switch u.sc.code[i] {
		case "while", "for", "foreach", "switch", "if", "elseif":
			open, ok := u.pOpen[i]
			return ok && open < v && v < f.toks[open].closer
		}
	}
	return false
}

func (u *slvUselessVariable) inFunctionCall(v int) bool {
	f := u.f
	for i := v - 1; i >= 0; i-- {
		if !u.isTString(i) {
			continue
		}
		open := f.nextEffective(i+1, len(f.toks))
		return f.is(open, "(") && open < v && v < f.toks[open].closer
	}
	return false
}

// isTString tells a T_STRING: an unqualified name PHP_CodeSniffer does not
// give a token of its own.
func (u *slvUselessVariable) isTString(i int) bool {
	t := u.f.toks[i]
	return t.kind == token.Ident && !strings.Contains(t.orig, `\`) && u.f.lineCallIsName(i) && u.sc.code[i] == "string"
}

func (u *slvUselessVariable) hasVarAnnotation(v int) bool {
	f := u.f
	closeTag := f.prevNonWhitespace(v - 1)
	if closeTag < 0 || f.toks[closeTag].doc != slvDocClose {
		return false
	}
	open := closeTag
	for open > 0 && f.toks[open].doc != slvDocOpen {
		open--
	}
	var b strings.Builder
	for i := open; i <= closeTag; i++ {
		b.WriteString(f.toks[i].content)
	}
	re := regexp.MustCompile(`@(?:(?:phpstan|psalm)-)?var\s+.+\s+` + regexp.QuoteMeta(f.toks[v].content) + `(?:\s|$)`)
	return re.MatchString(b.String())
}

func (u *slvUselessVariable) hasAnotherAssignmentBefore(v int, name string) bool {
	prev := u.previousVariable(v, name)
	return prev >= 0 && u.isAssignment(prev) && u.variablesNearby(prev, v)
}

// nearby is areBothPointersNearby.
func (u *slvUselessVariable) nearby(first, second int) bool {
	semi := u.semicolon(first)
	return semi >= 0 && u.f.nextEffective(semi+1, len(u.f.toks)) == second
}

// variablesNearby is areBothVariablesNearby.
func (u *slvUselessVariable) variablesNearby(first, second int) bool {
	if u.nearby(first, second) {
		return true
	}
	f := u.f
	c := u.conds[first]
	if len(c) == 0 || c[len(c)-1].caseIndex {
		return false
	}
	cond := c[len(c)-1].key
	closer := u.sc.closer[cond]
	if closer < 0 {
		return false
	}
	if u.sc.code[cond] == "do" {
		for closer++; closer < len(f.toks) && !f.is(closer, ";"); closer++ {
		}
	}
	return f.nextEffective(closer+1, len(f.toks)) == second
}

// semicolon is findSemicolon: the next ";" in the same scope.
func (u *slvUselessVariable) semicolon(v int) int {
	f := u.f
	for i := v + 1; i < len(f.toks)-1; i++ {
		if f.is(i, ";") && u.sameScope(v, i) {
			return i
		}
	}
	return -1
}
