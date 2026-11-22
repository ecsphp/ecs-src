package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// clFMemberSpacing holds the options of AbstractPropertyConstantAndEnumCaseSpacing;
// the multiline counts are -1 when unset.
type clFMemberSpacing struct {
	MinLinesCountBeforeWithComment, MaxLinesCountBeforeWithComment       int
	MinLinesCountBeforeWithoutComment, MaxLinesCountBeforeWithoutComment int
	MinLinesCountBeforeMultiline, MaxLinesCountBeforeMultiline           int
}

var clFMemberSpacingDefaults = clFMemberSpacing{
	MinLinesCountBeforeWithComment:    1,
	MaxLinesCountBeforeWithComment:    1,
	MinLinesCountBeforeWithoutComment: 0,
	MaxLinesCountBeforeWithoutComment: 1,
	MinLinesCountBeforeMultiline:      -1,
	MaxLinesCountBeforeMultiline:      -1,
}

func (o clFMemberSpacing) withConfig(config map[string]any) clFMemberSpacing {
	o.MinLinesCountBeforeWithComment = configInt(config, "minLinesCountBeforeWithComment", o.MinLinesCountBeforeWithComment)
	o.MaxLinesCountBeforeWithComment = configInt(config, "maxLinesCountBeforeWithComment", o.MaxLinesCountBeforeWithComment)
	o.MinLinesCountBeforeWithoutComment = configInt(config, "minLinesCountBeforeWithoutComment", o.MinLinesCountBeforeWithoutComment)
	o.MaxLinesCountBeforeWithoutComment = configInt(config, "maxLinesCountBeforeWithoutComment", o.MaxLinesCountBeforeWithoutComment)
	o.MinLinesCountBeforeMultiline = configInt(config, "minLinesCountBeforeMultiline", o.MinLinesCountBeforeMultiline)
	o.MaxLinesCountBeforeMultiline = configInt(config, "maxLinesCountBeforeMultiline", o.MaxLinesCountBeforeMultiline)
	return o
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/PropertySpacingSniff.php
//
// SlevomatPropertySpacing sets the blank lines after a property.
type SlevomatPropertySpacing struct{ clFMemberSpacing }

func (SlevomatPropertySpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\PropertySpacingSniff`
}

func (SlevomatPropertySpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatPropertySpacing{}.Name())
}

func (r SlevomatPropertySpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.clFMemberSpacing = r.withConfig(config)
	return r
}

func (r SlevomatPropertySpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		m := newClFMembers(f, r.clFMemberSpacing, clFProperty)
		f.eachToken(m.isModifier, func(i int) int {
			if f.isKeyword(f.prevEffective(i-1), "as") {
				return i
			}
			if next := f.nextEffective(i+1, len(f.toks)); next >= 0 && m.isModifier(next) {
				return next
			}
			property := m.findNext(func(j int) bool {
				return m.isVariable(j) || m.isFunction(j) || m.isConst(j) || m.isClass(j)
			}, i+1, len(f.toks))
			if property < 0 {
				return i
			}
			if !m.isVariable(property) || !m.isProperty(property) {
				return property
			}
			return m.process(property)
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/ConstantSpacingSniff.php
//
// SlevomatConstantSpacing sets the blank lines after a class constant.
type SlevomatConstantSpacing struct{ clFMemberSpacing }

func (SlevomatConstantSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\ConstantSpacingSniff`
}

func (SlevomatConstantSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatConstantSpacing{}.Name())
}

func (r SlevomatConstantSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.clFMemberSpacing = r.withConfig(config)
	return r
}

func (r SlevomatConstantSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		m := newClFMembers(f, r.clFMemberSpacing, clFConstant)
		f.eachToken(m.isConst, func(i int) int {
			conds := m.conds[i]
			if len(conds) == 0 || !m.isOOScope(conds[len(conds)-1].key) {
				return i
			}
			return m.process(i)
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/EnumCaseSpacingSniff.php
//
// SlevomatEnumCaseSpacing sets the blank lines after an enum case.
type SlevomatEnumCaseSpacing struct{ clFMemberSpacing }

func (SlevomatEnumCaseSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\EnumCaseSpacingSniff`
}

func (SlevomatEnumCaseSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatEnumCaseSpacing{}.Name())
}

func (r SlevomatEnumCaseSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.clFMemberSpacing = r.withConfig(config)
	return r
}

func (r SlevomatEnumCaseSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		m := newClFMembers(f, r.clFMemberSpacing, clFEnumCase)
		f.eachToken(m.isEnumCase, m.process)
	})
}

type clFMemberKind uint8

const (
	clFProperty clFMemberKind = iota
	clFConstant
	clFEnumCase
)

// clFMembers is AbstractPropertyConstantAndEnumCaseSpacing over one pass of a file.
type clFMembers struct {
	f     *slvFile
	sc    *phpcsScope
	conds [][]phpcsCond
	opts  clFMemberSpacing
	kind  clFMemberKind
}

func newClFMembers(f *slvFile, opts clFMemberSpacing, kind clFMemberKind) *clFMembers {
	sc := newPhpcsScope(f)
	conds, _ := sc.levelMap()
	return &clFMembers{f: f, sc: sc, conds: conds, opts: opts, kind: kind}
}

func (m *clFMembers) findNext(match func(int) bool, start, end int) int {
	end = min(end, len(m.f.toks))
	for i := max(start, 0); i < end; i++ {
		if match(i) {
			return i
		}
	}
	return -1
}

func (m *clFMembers) findPrevious(match func(int) bool, start int) int {
	for i := min(start, len(m.f.toks)-1); i >= 0; i-- {
		if match(i) {
			return i
		}
	}
	return -1
}

// isModifier is TokenHelper::PROPERTY_MODIFIERS_TOKEN_CODES.
func (m *clFMembers) isModifier(i int) bool {
	f := m.f
	if i < 0 || f.toks[i].kind != token.Keyword {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "final", "abstract", "var", "public", "protected", "private", "readonly", "static":
		return true
	}
	return f.isSetScope(i)
}

func (m *clFMembers) isVariable(i int) bool { return i >= 0 && m.f.thIsVariable(i) }

// isFunction is T_FUNCTION: a named function, not a closure.
func (m *clFMembers) isFunction(i int) bool {
	return m.f.isKeyword(i, "function") && !m.f.isClosure(i)
}

func (m *clFMembers) isConst(i int) bool { return i >= 0 && m.sc.code[i] == "const" }

func (m *clFMembers) isEnumCase(i int) bool { return i >= 0 && m.sc.code[i] == "enum_case" }

func (m *clFMembers) isUse(i int) bool { return m.f.isKeyword(i, "use") }

// isClass is T_CLASS: not an anonymous class.
func (m *clFMembers) isClass(i int) bool {
	return m.f.isKeyword(i, "class") && !m.f.isAnonClass(i)
}

// isComment is T_COMMENT, without the phpcs annotations PHP_CodeSniffer
// tokenizes on their own.
func (m *clFMembers) isComment(i int) bool {
	t := m.f.toks[i]
	return t.kind == token.Comment && !slvIsPhpcsAnnotation(t.content)
}

func (m *clFMembers) isOOScope(i int) bool {
	switch m.sc.code[i] {
	case "class", "interface", "trait", "enum":
		return true
	}
	return false
}

// endPointer is getEndPointer: the ";" of the member, or the closer of its hooks.
func (m *clFMembers) endPointer(i int) int {
	f := m.f
	end := m.findNext(func(j int) bool { return f.is(j, ";") || f.is(j, "{") }, i+1, len(f.toks))
	if end >= 0 && f.is(end, "{") {
		return f.toks[end].closer
	}
	return end
}

func (m *clFMembers) isNextMemberValid(i int) bool {
	switch m.kind {
	case clFProperty:
		next := m.findNext(func(j int) bool { return m.isFunction(j) || m.isVariable(j) }, i+1, len(m.f.toks))
		return next >= 0 && m.isVariable(next)
	case clFConstant:
		if m.isConst(i) {
			return true
		}
	case clFEnumCase:
		if m.isEnumCase(i) {
			return true
		}
	}
	next := m.findNext(func(j int) bool {
		return m.isFunction(j) || m.isEnumCase(j) || m.isConst(j) || m.isVariable(j) || m.isUse(j)
	}, i+1, len(m.f.toks))
	if next < 0 {
		return false
	}
	if m.kind == clFConstant {
		return m.isConst(next)
	}
	return m.isEnumCase(next)
}

// process is AbstractPropertyConstantAndEnumCaseSpacing::process.
func (m *clFMembers) process(pointer int) int {
	f := m.f
	o := m.opts
	n := len(f.toks)

	end := m.endPointer(pointer)
	if end < 0 {
		return n
	}
	firstOnLine := f.firstOnNextLine(end)
	if firstOnLine < 0 {
		return n
	}

	nextFunction := m.findNext(func(j int) bool {
		return m.isFunction(j) || m.isEnumCase(j) || m.isConst(j) || m.isVariable(j) || m.isUse(j)
	}, firstOnLine+1, n)
	if nextFunction < 0 {
		return firstOnLine
	}
	if m.isFunction(nextFunction) || !slices.Equal(m.conds[nextFunction], m.conds[pointer]) {
		return nextFunction
	}

	// ClassHelper::getClassPointer: the innermost OO scope
	closer := n
	for _, c := range slices.Backward(m.conds[pointer]) {
		if m.isOOScope(c.key) {
			if m.sc.closer[c.key] >= 0 {
				closer = m.sc.closer[c.key]
			}
			break
		}
	}
	next := m.findNext(func(j int) bool {
		return m.isComment(j) || f.isDocOpen(j) || f.toks[j].kind == token.AttributeStart ||
			m.isEnumCase(j) || m.isConst(j) || m.isUse(j) || m.isModifier(j)
	}, firstOnLine+1, closer)
	if next < 0 {
		// upstream passes null on to an int parameter and throws
		return n
	}
	if !m.isNextMemberValid(next) {
		return next
	}

	linesBetween := f.toks[next].line - f.toks[end].line - 1
	minLines, maxLines := o.MinLinesCountBeforeWithoutComment, o.MaxLinesCountBeforeWithoutComment
	if m.isComment(next) || f.isDocOpen(next) || f.toks[next].kind == token.AttributeStart {
		minLines, maxLines = o.MinLinesCountBeforeWithComment, o.MaxLinesCountBeforeWithComment
	}
	multiline := m.kind != clFEnumCase && f.toks[pointer].line != f.toks[end].line
	if o.MinLinesCountBeforeMultiline >= 0 && multiline {
		minLines = max(minLines, o.MinLinesCountBeforeMultiline)
		maxLines = max(minLines, maxLines)
	}
	if o.MaxLinesCountBeforeMultiline >= 0 && multiline {
		maxLines = max(minLines, o.MaxLinesCountBeforeMultiline)
	}

	if linesBetween >= minLines && linesBetween <= maxLines {
		return firstOnLine
	}

	if linesBetween > maxLines {
		lastOnLine := f.lastOnLine(end)
		firstOnNextLine := f.firstOnLine(next)
		f.beginChangeset()
		if maxLines > 0 {
			f.add(lastOnLine, strings.Repeat(f.eol, maxLines))
		}
		f.removeBetweenIncluding(lastOnLine+1, firstOnNextLine-1)
		f.endChangeset()
	} else {
		f.beginChangeset()
		for range minLines - linesBetween {
			f.addContentBefore(firstOnLine, f.eol)
		}
		f.endChangeset()
	}
	return firstOnLine
}

// isTypeHintToken is TokenHelper::TYPE_HINT_TOKEN_CODES plus T_NULLABLE.
func (m *clFMembers) isTypeHintToken(i int) bool {
	f := m.f
	t := f.toks[i]
	switch t.kind {
	case token.Ident:
		return true
	case token.Keyword:
		return strings.EqualFold(t.orig, "callable")
	case token.Punct:
		switch t.orig {
		case "?", "|":
			return true
		case "&":
			next := f.nextEffective(i+1, len(f.toks))
			return !m.isVariable(next) && !f.is(next, "...")
		case "(":
			return f.slvmIsTypeParen(i)
		case ")":
			return t.closer >= 0 && f.slvmIsTypeParen(t.closer)
		}
	}
	return false
}

// isProperty is PropertyHelper::isProperty without promoted parameters.
func (m *clFMembers) isProperty(v int) bool {
	f := m.f
	sc := m.sc
	prev := v - 1
	for prev >= 0 && (f.isEmpty(prev) || m.isTypeHintToken(prev)) {
		prev--
	}
	if prev < 0 {
		return false
	}
	if f.isKeyword(prev, "final") || f.isKeyword(prev, "abstract") {
		return true
	}
	if f.isKeyword(prev, "static") {
		prev = f.prevEffective(prev - 1)
	}
	if prev >= 0 && (f.isScopeModifier(prev) || f.isKeyword(prev, "readonly")) {
		if sc.nested[prev] < 0 {
			return true
		}
		constructor := m.findPrevious(m.isFunction, prev-1)
		if constructor < 0 {
			return true
		}
		return sc.pCloser[constructor] < prev
	}

	conds := m.conds[v]
	if len(conds) == 0 || !m.isOOScope(conds[len(conds)-1].key) {
		return false
	}

	function := m.findPrevious(func(j int) bool {
		return f.isKeyword(j, "function") || f.isKeyword(j, "fn") && sc.code[j] == "fn" || f.is(j, ";") || f.is(j, "{") || f.is(j, "}")
	}, v-1)
	if function >= 0 && (f.isKeyword(function, "function") || f.isKeyword(function, "fn")) {
		return false
	}

	paren := m.findPrevious(func(j int) bool { return f.is(j, "(") && !f.slvmIsTypeParen(j) }, v-1)
	if paren >= 0 && f.toks[paren].closer > v {
		if p := f.prevEffective(paren - 1); p >= 0 && (f.toks[p].content == "get" || f.toks[p].content == "set") {
			return false
		}
	}

	curly := m.findPrevious(func(j int) bool { return f.is(j, "{") }, v-1)
	if curly >= 0 && f.toks[curly].closer > v && sc.cond[curly] < 0 {
		return false
	}
	return true
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/ParentCallSpacingSniff.php
//
// SlevomatParentCallSpacing sets the blank lines around a parent call.
type SlevomatParentCallSpacing struct {
	LinesCountBefore, LinesCountBeforeFirst, LinesCountAfter, LinesCountAfterLast int
}

func (SlevomatParentCallSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\ParentCallSpacingSniff`
}

func (SlevomatParentCallSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatParentCallSpacing{}.Name())
}

func (r SlevomatParentCallSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.LinesCountBefore = configInt(config, "linesCountBefore", r.LinesCountBefore)
	r.LinesCountBeforeFirst = configInt(config, "linesCountBeforeFirst", r.LinesCountBeforeFirst)
	r.LinesCountAfter = configInt(config, "linesCountAfter", r.LinesCountAfter)
	r.LinesCountAfterLast = configInt(config, "linesCountAfterLast", r.LinesCountAfterLast)
	return r
}

var clFParentIgnoredBefore = phpcsSet(
	"=", "&=", "|=", ".=", "/=", "-=", "**=", "%=", "*=", "+=", "^=", "=>", "<<=", ">>=", "??=",
	"==", "!=", "<>", "===", "!==", "<=", ">=",
	"&&", "||", "and", "or", "xor",
	"return", "yield", ".", "inline_else", "??", "match_arrow",
)

func (r SlevomatParentCallSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		c := newSlvcss(f)
		c.before, c.after = r.LinesCountBefore, r.LinesCountAfter
		c.beforeFirst = func(int) int { return r.LinesCountBeforeFirst }
		c.afterLast = func(int, int) int { return r.LinesCountAfterLast }
		f.eachToken(func(i int) bool { return clFIsParent(c, i) }, func(i int) int {
			if c.sc.nested[i] >= 0 {
				return 0
			}
			previous := f.prevEffective(i - 1)
			if clFIsCast(f, previous) || f.is(previous, "@") {
				previous = f.prevEffective(previous - 1)
			}
			if previous >= 0 {
				code := c.code(previous)
				if clFParentIgnoredBefore[code] || c.isColon(previous) ||
					strings.HasPrefix(code, "yield") || code == "?" && c.sc.isInlineThen(previous) {
					return 0
				}
			}
			for j := i - 1; j >= 0; j-- {
				if f.isShortArrayOpen(j) {
					if f.toks[j].closer > i {
						return 0
					}
					break
				}
			}
			start := i
			if p := f.prevEffective(i - 1); f.is(p, "@") {
				start = p
			}
			clFCheckLinesBefore(c, i, start)
			c.checkLinesAfter(i)
			return 0
		})
	})
}

// clFIsCast is Tokens::CAST_TOKENS.
func clFIsCast(f *slvFile, i int) bool {
	if i < 0 || f.toks[i].kind != token.Punct {
		return false
	}
	v := f.toks[i].orig
	return len(v) > 2 && v[0] == '(' && v[len(v)-1] == ')'
}

// clFIsParent is T_PARENT: "parent" outside the contexts PHP_CodeSniffer keeps
// as T_STRING.
func clFIsParent(c *slvcss, i int) bool {
	f := c.f
	t := f.toks[i]
	if t.kind != token.Ident || !strings.EqualFold(t.orig, "parent") {
		return false
	}
	prev := i - 1
	for prev >= 0 && f.isEmpty(prev) {
		prev--
	}
	if prev >= 0 {
		switch c.code(prev) {
		case "->", "?->", "::", "function", "class", "interface", "trait", "enum", "enum_case",
			"extends", "implements", "const", "use", "namespace", "goto":
			return false
		case "new":
			return true
		case "&":
			return !f.isKeyword(f.prevEffective(prev-1), "function")
		}
		if f.toks[prev].kind == token.AttributeStart {
			return false
		}
	}
	// the name of a typed constant, `const string PARENT = ...`
	next := f.nextEffective(i+1, len(f.toks))
	return !f.is(next, "(") && !f.is(next, "=")
}

// clFCheckLinesBefore is AbstractControlStructureSpacing::checkLinesBefore
// with a control structure starting at start, before i.
func clFCheckLinesBefore(c *slvcss, i, start int) {
	f := c.f
	nonWhitespaceBefore := f.prevNonWhitespace(start - 1)
	if nonWhitespaceBefore < 0 {
		return
	}
	pointerBefore := nonWhitespaceBefore
	pointerToCheckFirst := pointerBefore
	if c.isComment(nonWhitespaceBefore) {
		effectiveBefore := f.prevEffective(pointerBefore - 1)
		if effectiveBefore < 0 {
			return
		}
		if c.line(effectiveBefore) == c.line(nonWhitespaceBefore) {
			pointerToCheckFirst = effectiveBefore
		} else if c.line(nonWhitespaceBefore)+1 == c.line(i) {
			if f.isDocClose(nonWhitespaceBefore) {
				start = f.docBlockOpener(nonWhitespaceBefore)
			} else {
				start = c.multilineCommentStart(nonWhitespaceBefore)
			}
			pointerBefore = f.prevNonWhitespace(start - 1)
			pointerToCheckFirst = pointerBefore
		}
	}

	isFirst := c.code(pointerToCheckFirst) == "{" || c.isColon(pointerToCheckFirst)
	whitespaceBefore := ""
	isOpenTag := f.toks[pointerBefore].kind == token.OpenTag
	if isOpenTag {
		whitespaceBefore += f.toks[pointerBefore].content[min(len("<?php"), len(f.toks[pointerBefore].content)):]
	}
	hasCommentWithLineEndBefore := c.isInlineComment(pointerBefore) && strings.HasSuffix(f.toks[pointerBefore].content, f.eol)
	if hasCommentWithLineEndBefore {
		whitespaceBefore += f.eol
	}
	if pointerBefore+1 != i {
		whitespaceBefore += f.content(pointerBefore+1, start-1)
	}

	required := c.before
	if isFirst {
		required = c.beforeFirst(i)
	}
	if required == strings.Count(whitespaceBefore, f.eol)-1 {
		return
	}

	endOfLineBefore := c.findPrevious(func(j int) bool {
		return f.toks[j].kind == token.Whitespace && f.toks[j].content == f.eol
	}, start-1)

	f.beginChangeset()
	if isOpenTag {
		f.replace(pointerBefore, "<?php")
	}
	if endOfLineBefore >= 0 {
		f.removeBetweenIncluding(pointerBefore+1, endOfLineBefore)
	}
	if c.line(pointerBefore) == c.line(start) {
		f.removeBetweenIncluding(pointerBefore+1, start-1)
	}
	linesToAdd := required
	if hasCommentWithLineEndBefore {
		linesToAdd--
	}
	for range linesToAdd + 1 {
		f.addContent(pointerBefore, f.eol)
	}
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/BackedEnumTypeSpacingSniff.php
//
// SlevomatBackedEnumTypeSpacing sets the spaces around the colon of a backed enum.
type SlevomatBackedEnumTypeSpacing struct {
	SpacesCountBeforeColon, SpacesCountBeforeType int
}

func (SlevomatBackedEnumTypeSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\BackedEnumTypeSpacingSniff`
}

func (SlevomatBackedEnumTypeSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatBackedEnumTypeSpacing{}.Name())
}

func (r SlevomatBackedEnumTypeSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.SpacesCountBeforeColon = configInt(config, "spacesCountBeforeColon", r.SpacesCountBeforeColon)
	r.SpacesCountBeforeType = configInt(config, "spacesCountBeforeType", r.SpacesCountBeforeType)
	return r
}

func (r SlevomatBackedEnumTypeSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		f.eachToken(func(i int) bool { return sc.code[i] == "enum" }, func(i int) int {
			opener := sc.opener[i]
			if opener < 0 {
				return 0
			}
			colon := -1
			for j := i + 1; j < opener; j++ {
				if sc.code[j] == ":" {
					colon = j
					break
				}
			}
			if colon < 0 {
				return 0
			}
			name := f.prevEffective(colon - 1)
			if len(f.content(name+1, colon-1)) != r.SpacesCountBeforeColon {
				f.beginChangeset()
				f.removeBetweenIncluding(name+1, colon-1)
				f.addBefore(colon, strings.Repeat(" ", r.SpacesCountBeforeColon))
				f.endChangeset()
			}
			typ := f.nextEffective(colon+1, len(f.toks))
			if typ >= 0 && len(f.content(colon+1, typ-1)) != r.SpacesCountBeforeType {
				f.beginChangeset()
				f.removeBetweenIncluding(colon+1, typ-1)
				f.addBefore(typ, strings.Repeat(" ", r.SpacesCountBeforeType))
				f.endChangeset()
			}
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/ClassKeywordOrderSniff.php
//
// SlevomatClassKeywordOrder puts abstract or final before readonly.
type SlevomatClassKeywordOrder struct{}

func (SlevomatClassKeywordOrder) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\ClassKeywordOrderSniff`
}

func (SlevomatClassKeywordOrder) SourceURL() string {
	return fixer.SourceURLFor(SlevomatClassKeywordOrder{}.Name())
}

func (SlevomatClassKeywordOrder) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		isClass := func(i int) bool { return f.isKeyword(i, "class") && !f.isAnonClass(i) }
		f.eachToken(isClass, func(i int) int {
			var found []int
			for p := f.prevEffective(i - 1); p >= 0 && clFClassModifier(f, p) != ""; p = f.prevEffective(p - 1) {
				found = append(found, p)
			}
			if len(found) == 0 {
				return 0
			}
			slices.Reverse(found)
			actual := make([]string, len(found))
			for k, p := range found {
				actual[k] = clFClassModifier(f, p)
			}
			expected := slices.Clone(actual)
			slices.SortStableFunc(expected, func(a, b string) int {
				return clFClassModifierPriority(a) - clFClassModifierPriority(b)
			})
			if slices.Equal(actual, expected) {
				return 0
			}
			f.beginChangeset()
			for _, p := range found {
				f.replaceToken(p, "")
				if p+1 < len(f.toks) && f.toks[p+1].kind == token.Whitespace {
					f.replaceToken(p+1, "")
				}
			}
			f.addContentBefore(found[0], strings.Join(expected, " ")+" ")
			f.endChangeset()
			return 0
		})
	})
}

// clFClassModifier is the lowercase class modifier at i, "" for none.
func clFClassModifier(f *slvFile, i int) string {
	for _, v := range []string{"abstract", "final", "readonly"} {
		if f.isKeyword(i, v) {
			return v
		}
	}
	return ""
}

func clFClassModifierPriority(v string) int {
	if v == "readonly" {
		return 1
	}
	return 0
}
