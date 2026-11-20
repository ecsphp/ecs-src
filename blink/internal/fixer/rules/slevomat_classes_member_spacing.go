package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// clCCtx holds a pass's PHP_CodeSniffer scope map for the Classes ports.
type clCCtx struct {
	f      *slvFile
	sc     *phpcsScope
	levels []int
}

func newClCCtx(f *slvFile) *clCCtx {
	c := &clCCtx{f: f, sc: newPhpcsScope(f)}
	_, c.levels = c.sc.levelMap()
	return c
}

// hasScope tells a scope condition with a scope_opener.
func (c *clCCtx) hasScope(i int) bool {
	return c.sc.cond[i] == i && c.sc.opener[i] > i && c.sc.closer[i] > c.sc.opener[i]
}

// isFunction tells T_FUNCTION: a named function, not a closure (T_CLOSURE).
func (c *clCCtx) isFunction(i int) bool {
	return c.sc.code[i] == "function" && !c.f.isClosure(i)
}

// isClassType tells a CLASS_TYPE_WITH_ANONYMOUS_CLASS_TOKEN_CODES token with
// a body.
func (c *clCCtx) isClassType(i int) bool {
	switch c.sc.code[i] {
	case "class", "interface", "trait", "enum":
		return !c.f.is(c.f.prevEffective(i-1), "::") && c.hasScope(i)
	}
	return false
}

// isAnonClass tells T_ANON_CLASS: a "class" followed by "(", "{", "extends"
// or "implements".
func (c *clCCtx) isAnonClass(class int) bool {
	f := c.f
	n := f.nextEffective(class+1, len(f.toks))
	return f.is(n, "(") || f.is(n, "{") || f.isKeyword(n, "extends") || f.isKeyword(n, "implements")
}

// isModifier tells TokenHelper::MODIFIERS_TOKEN_CODES.
func (c *clCCtx) isModifier(i int) bool {
	f := c.f
	if i < 0 || !f.isModifier(i) {
		return false
	}
	return f.toks[i].kind == token.Keyword || strings.HasSuffix(f.toks[i].orig, "(set)")
}

// isMethod is FunctionHelper::isMethod.
func (c *clCCtx) isMethod(fn int) bool {
	conds := c.sc.conditions(fn)
	if len(conds) == 0 {
		return false
	}
	switch c.sc.code[conds[len(conds)-1]] {
	case "class", "interface", "trait", "enum":
		return true
	}
	return false
}

// classPointer is ClassHelper::getClassPointer, or -1.
func (c *clCCtx) classPointer(i int) int {
	best := -1
	for p := range c.f.toks {
		if !c.isClassType(p) {
			continue
		}
		if c.sc.opener[p] < i && c.sc.closer[p] > i {
			best = p
		}
	}
	return best
}

// findNext is TokenHelper::findNext over [from, end), or -1.
func (c *clCCtx) findNext(from, end int, match func(i int) bool) int {
	end = min(end, len(c.f.toks))
	for i := max(from, 0); i < end; i++ {
		if match(i) {
			return i
		}
	}
	return -1
}

// findPrevious is TokenHelper::findPrevious from i down to end (inclusive), or -1.
func (c *clCCtx) findPrevious(i, end int, match func(i int) bool) int {
	for ; i >= max(end, 0); i-- {
		if match(i) {
			return i
		}
	}
	return -1
}

func (c *clCCtx) isAttributeEnd(i int) bool {
	return i >= 0 && c.f.toks[i].kind == token.AttributeEnd && c.f.isAttributeOpen(c.f.toks[i].closer)
}

// isInlineComment tells TokenHelper::INLINE_COMMENT_TOKEN_CODES: T_COMMENT and
// the phpcs annotations.
func (c *clCCtx) isInlineComment(i int) bool {
	return i >= 0 && c.f.toks[i].kind == token.Comment
}

var clCLineComment = regexp.MustCompile(`^(?://|#)`)

// commentEnd is CommentHelper::getCommentEndPointer of an inline comment.
func (c *clCCtx) commentEnd(start int) int {
	f := c.f
	if clCLineComment.MatchString(f.toks[start].content) {
		return start
	}
	if !strings.HasPrefix(f.toks[start].content, "/*") {
		return -1
	}
	end := start
	for i := start + 1; i < len(f.toks) && c.isInlineComment(i); i++ {
		end = i
	}
	return end
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/MethodSpacingSniff.php
//
// SlevomatMethodSpacing sets the blank lines between two methods.
type SlevomatMethodSpacing struct {
	min, max   int
	configured bool
}

func (SlevomatMethodSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\MethodSpacingSniff`
}

func (SlevomatMethodSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatMethodSpacing{}.Name())
}

func (r SlevomatMethodSpacing) withDefaults() SlevomatMethodSpacing {
	if !r.configured {
		r.min, r.max, r.configured = 1, 1, true
	}
	return r
}

func (r SlevomatMethodSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r = r.withDefaults()
	r.min = thInt(config, "minLinesCount", r.min)
	r.max = thInt(config, "maxLinesCount", r.max)
	return r
}

func (r SlevomatMethodSpacing) Fix(s *tokens.Stream) bool {
	r = r.withDefaults()
	return slvRun(s, func(f *slvFile) {
		c := newClCCtx(f)
		for i := range f.toks {
			if c.isFunction(i) {
				r.process(c, i)
			}
		}
	})
}

func (r SlevomatMethodSpacing) process(c *clCCtx, method int) {
	f := c.f
	if !c.isMethod(method) {
		return
	}
	var methodEnd int
	if c.hasScope(method) {
		methodEnd = c.sc.closer[method]
	} else {
		methodEnd = c.findNext(method+1, len(f.toks), func(i int) bool { return f.is(i, ";") })
	}
	class := c.classPointer(method)
	if methodEnd < 0 || class < 0 {
		return
	}
	next := c.findNext(methodEnd+1, c.sc.closer[class], c.isFunction)
	if next < 0 {
		return
	}

	attrStart := -1
	docStart := f.docOpenBefore(next)
	if docStart >= 0 && f.toks[f.docCloser(docStart)].line+1 != f.toks[next].line {
		docStart = -1
	} else {
		attrStart = c.findPrevious(next-1, methodEnd, func(i int) bool { return f.isAttributeOpen(i) })
		if attrStart >= 0 {
			for {
				before := c.findPrevious(attrStart-1, methodEnd, func(i int) bool { return f.toks[i].kind != token.Whitespace })
				if c.isAttributeEnd(before) {
					attrStart = f.toks[before].closer
					continue
				}
				break
			}
		}
	}

	var firstLine int
	if f.toks[next].line == f.toks[methodEnd].line {
		firstLine = f.nextEffective(methodEnd+1, len(f.toks))
	} else {
		from := next
		if docStart >= 0 {
			from = docStart
		} else if attrStart >= 0 {
			from = attrStart
		}
		firstLine = f.firstOnLine(from)
	}
	if f.nextNonWhitespace(methodEnd+1, firstLine) >= 0 {
		return
	}

	hasLines := f.toks[firstLine].line != f.toks[methodEnd].line
	between := f.toks[firstLine].line - f.toks[methodEnd].line - 1
	if hasLines && between >= r.min && between <= r.max {
		return
	}

	f.beginChangeset()
	switch {
	case !hasLines:
		f.add(methodEnd, f.eol+strings.Repeat(f.eol, r.min)+f.indentation(f.firstNonWhitespaceOnLine(method)))
		f.removeBetweenIncluding(methodEnd+1, firstLine-1)
	case between > r.max:
		f.add(methodEnd, strings.Repeat(f.eol, r.max+1))
		f.removeBetweenIncluding(methodEnd+1, f.firstOnLine(firstLine)-1)
	default:
		f.add(methodEnd, strings.Repeat(f.eol, r.min-between))
	}
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/ClassMemberSpacingSniff.php
//
// SlevomatClassMemberSpacing sets the blank lines between class members of
// different kinds.
type SlevomatClassMemberSpacing struct {
	lines      int
	configured bool
}

func (SlevomatClassMemberSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\ClassMemberSpacingSniff`
}

func (SlevomatClassMemberSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatClassMemberSpacing{}.Name())
}

func (r SlevomatClassMemberSpacing) withDefaults() SlevomatClassMemberSpacing {
	if !r.configured {
		r.lines, r.configured = 1, true
	}
	return r
}

func (r SlevomatClassMemberSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r = r.withDefaults()
	r.lines = thInt(config, "linesCountBetweenMembers", r.lines)
	return r
}

func (r SlevomatClassMemberSpacing) Fix(s *tokens.Stream) bool {
	r = r.withDefaults()
	return slvRun(s, func(f *slvFile) {
		c := newClCCtx(f)
		for i := range f.toks {
			if c.isClassType(i) {
				r.process(c, i)
			}
		}
	})
}

// memberCode is the token code ClassMemberSpacing compares members by.
func (c *clCCtx) memberCode(i int) string {
	if c.f.toks[i].kind == token.Variable {
		return "variable"
	}
	return c.sc.code[i]
}

func (r SlevomatClassMemberSpacing) process(c *clCCtx, class int) {
	f := c.f
	member := -1
	for {
		prev := member
		from := prev
		if from < 0 {
			from = c.sc.opener[class]
		}
		member = c.nextMember(class, from)
		if member < 0 {
			return
		}
		if prev < 0 || c.memberCode(prev) == c.memberCode(member) {
			continue
		}

		prevEnd := c.memberEnd(prev)
		if prevEnd < 0 {
			return
		}
		commentWithNewLine := false
		comment := f.nextNonWhitespace(prevEnd+1, len(f.toks))
		if c.isInlineComment(comment) &&
			(f.toks[prevEnd].line == f.toks[comment].line || f.toks[prevEnd].line+1 == f.toks[comment].line) {
			prevEnd = c.commentEnd(comment)
			if prevEnd < 0 {
				return
			}
			if strings.HasSuffix(f.toks[comment].content, f.eol) {
				commentWithNewLine = true
			}
		}

		start := c.memberStart(member, prevEnd)
		if start < 0 {
			return
		}
		actual := f.toks[start].line - f.toks[prevEnd].line - 1
		if actual == r.lines {
			continue
		}
		firstOnLine := f.firstOnLine(start)
		if f.nextNonWhitespace(prevEnd+1, firstOnLine) >= 0 {
			continue
		}
		n := r.lines + 1
		if commentWithNewLine {
			n = r.lines
		}
		f.beginChangeset()
		f.add(prevEnd, strings.Repeat(f.eol, max(n, 0)))
		f.removeBetweenIncluding(prevEnd+1, firstOnLine-1)
		f.endChangeset()
	}
}

// nextMember is ClassMemberSpacingSniff::findNextMember, or -1.
func (c *clCCtx) nextMember(class, prev int) int {
	f := c.f
	closer := c.sc.closer[class]
	member := prev
	for {
		member = c.findNext(member+1, closer, func(i int) bool {
			switch c.sc.code[i] {
			case "use", "const", "enum_case":
				return true
			}
			return c.isFunction(i) || c.isModifier(i)
		})
		if member < 0 {
			return -1
		}
		switch {
		case c.sc.code[member] == "use":
			if !f.forbiddenIsTraitUse(member) {
				continue
			}
		case c.isModifier(member):
			if f.isKeyword(f.prevEffective(member-1), "as") {
				continue
			}
			property := c.findNext(member+1, len(f.toks), func(i int) bool {
				return f.toks[i].kind == token.Variable && f.toks[i].orig != "$" || c.isFunction(i) || c.sc.code[i] == "const"
			})
			if property < 0 || f.toks[property].kind != token.Variable || !f.isProperty(property) {
				continue
			}
			member = property
		}
		if slvlcSameScope(c.sc, member, prev) {
			return member
		}
	}
}

// memberStart is ClassMemberSpacingSniff::getMemberStartPointer.
func (c *clCCtx) memberStart(member, prevEnd int) int {
	f := c.f
	first := c.memberFirstCode(member)
	for {
		if first <= prevEnd {
			return f.nextNonWhitespace(first+1, len(f.toks))
		}
		before := f.prevNonWhitespace(first - 1)
		if before < 0 {
			return first
		}
		if c.isAttributeEnd(before) {
			first = f.toks[before].closer
			continue
		}
		if k := f.toks[before].kind; k == token.Comment || k == token.DocComment {
			beforeComment := f.prevEffective(before - 1)
			if beforeComment < 0 || f.toks[beforeComment].line != f.toks[before].line {
				first = f.nsuCommentStart(before)
				continue
			}
		}
		return first
	}
}

// memberFirstCode is ClassMemberSpacingSniff::getMemberFirstCodePointer.
func (c *clCCtx) memberFirstCode(member int) int {
	f := c.f
	if c.sc.code[member] == "use" {
		return member
	}
	first, prevFirst := member, member
	for {
		first = c.findPrevious(first-1, 0, func(i int) bool {
			return c.isModifier(i) || f.is(i, ";") || f.is(i, "}")
		})
		if first < 0 || f.is(first, ";") || f.is(first, "}") {
			return prevFirst
		}
		prevFirst = first
	}
}

// memberEnd is ClassMemberSpacingSniff::getMemberEndPointer, or -1.
func (c *clCCtx) memberEnd(member int) int {
	f := c.f
	if c.sc.code[member] == "use" || f.toks[member].kind == token.Variable {
		p := c.findNext(member+1, len(f.toks), func(i int) bool { return f.is(i, ";") || f.is(i, "{") })
		if p < 0 {
			return -1
		}
		if f.is(p, "{") {
			return f.toks[p].closer
		}
		return p
	}
	if c.isFunction(member) && c.hasScope(member) {
		return c.sc.closer[member]
	}
	return c.findNext(member+1, len(f.toks), func(i int) bool { return f.is(i, ";") })
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/ReadonlyClassSniff.php
//
// SlevomatReadonlyClass marks a class readonly when all its properties are,
// and drops the readonly modifier of the properties of a readonly class.
type SlevomatReadonlyClass struct {
	disable, allowNonFinal, ignoreTraits bool
}

func (SlevomatReadonlyClass) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\ReadonlyClassSniff`
}

func (SlevomatReadonlyClass) SourceURL() string {
	return fixer.SourceURLFor(SlevomatReadonlyClass{}.Name())
}

func (r SlevomatReadonlyClass) WithConfig(config map[string]any) fixer.Fixer {
	r.disable = !thBool(config, "enable", true)
	r.allowNonFinal = thBool(config, "allowNonFinalClasses", false)
	r.ignoreTraits = thBool(config, "ignoreTraits", false)
	return r
}

type clCProperty struct{ property, readonly int }

func (r SlevomatReadonlyClass) Fix(s *tokens.Stream) bool {
	if r.disable {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		c := newClCCtx(f)
		for i := range f.toks {
			if c.isClassType(i) && f.isKeyword(i, "class") && !c.isAnonClass(i) {
				r.process(c, i)
			}
		}
	})
}

func (r SlevomatReadonlyClass) process(c *clCCtx, class int) {
	f := c.f
	constructor := c.constructor(class)
	var promoted []clCProperty
	if constructor >= 0 {
		promoted = c.promotedProperties(constructor)
	}
	body := c.bodyProperties(class, constructor)
	if len(promoted) == 0 && len(body) == 0 {
		return
	}
	if c.hasClassModifier(class, "readonly") {
		for _, p := range append(promoted, body...) {
			if p.readonly < 0 {
				continue
			}
			f.beginChangeset()
			c.removeReadonly(p.readonly)
			f.endChangeset()
		}
		return
	}
	for _, p := range promoted {
		if p.readonly < 0 {
			return
		}
	}
	for _, p := range body {
		if p.readonly < 0 {
			return
		}
	}
	if c.hasClassModifier(class, "abstract") {
		return
	}
	if !r.allowNonFinal && !f.isKeyword(f.prevEffective(class-1), "final") {
		return
	}
	if c.findNext(class+1, c.sc.opener[class], func(i int) bool { return f.isKeyword(i, "extends") }) >= 0 {
		return
	}
	if !r.ignoreTraits && c.hasTraitUse(class) {
		return
	}
	if c.hasAttribute(class, `\AllowDynamicProperties`) {
		return
	}
	f.beginChangeset()
	for _, p := range append(promoted, body...) {
		c.removeReadonly(p.readonly)
	}
	f.replaceToken(class, "readonly "+f.tokenContent(class))
	f.endChangeset()
}

func (c *clCCtx) constructor(class int) int {
	level := c.levels[class]
	for i := c.sc.opener[class] + 1; i < c.sc.closer[class]; i++ {
		if !c.isFunction(i) || c.levels[i] != level+1 {
			continue
		}
		if strings.ToLower(c.functionName(i)) == "__construct" {
			return i
		}
	}
	return -1
}

// functionName is FunctionHelper::getName: the first T_STRING before the
// parameters.
func (c *clCCtx) functionName(fn int) string {
	f := c.f
	open := f.functionParenOpen(fn)
	if open < 0 {
		return ""
	}
	for j := fn + 1; j < open; j++ {
		if c.sc.code[j] == "string" {
			return f.toks[j].content
		}
	}
	return ""
}

func (c *clCCtx) hasClassModifier(class int, modifier string) bool {
	f := c.f
	for p := f.prevEffective(class - 1); p >= 0 && (f.isKeyword(p, "final") || f.isKeyword(p, "abstract") || f.isKeyword(p, "readonly")); p = f.prevEffective(p - 1) {
		if f.isKeyword(p, modifier) {
			return true
		}
	}
	return false
}

func (c *clCCtx) promotedProperties(constructor int) []clCProperty {
	f := c.f
	open := f.functionParenOpen(constructor)
	if open < 0 {
		return nil
	}
	var out []clCProperty
	for i := open + 1; i < f.toks[open].closer; i++ {
		if f.toks[i].kind != token.Variable || f.toks[i].orig == "$" {
			continue
		}
		before := c.findPrevious(i-1, 0, func(j int) bool { return f.is(j, ",") || f.is(j, "(") || c.isAttributeEnd(j) })
		modifier := f.nextEffective(before+1, len(f.toks))
		if modifier < 0 || !c.isModifier(modifier) {
			continue
		}
		out = append(out, clCProperty{i, c.findNext(modifier, i, func(j int) bool { return f.isKeyword(j, "readonly") })})
	}
	return out
}

func (c *clCCtx) bodyProperties(class, constructor int) []clCProperty {
	f := c.f
	paramsOpen, paramsClose := -1, -1
	if constructor >= 0 {
		if paramsOpen = f.functionParenOpen(constructor); paramsOpen >= 0 {
			paramsClose = f.toks[paramsOpen].closer
		}
	}
	var out []clCProperty
	for i := c.sc.opener[class] + 1; i < c.sc.closer[class]; i++ {
		if f.toks[i].kind != token.Variable || f.toks[i].orig == "$" {
			continue
		}
		if paramsOpen >= 0 && i > paramsOpen && i < paramsClose {
			continue
		}
		if !f.isProperty(i) {
			continue
		}
		if conds := c.sc.conditions(i); len(conds) == 0 || conds[len(conds)-1] != class {
			continue
		}
		start := c.propertyStart(i)
		out = append(out, clCProperty{i, c.findNext(start, i, func(j int) bool { return f.isKeyword(j, "readonly") })})
	}
	return out
}

// propertyStart is PropertyHelper::getStartPointer.
func (c *clCCtx) propertyStart(property int) int {
	f := c.f
	end := c.findPrevious(property-1, 0, func(i int) bool {
		return f.is(i, ";") || f.is(i, "}") || f.is(i, "{") || f.is(i, "(") || f.is(i, ",")
	})
	start := property - 1
	for start > end && start >= 0 && f.isEmpty(start) {
		start--
	}
	if start <= end {
		start = -1
	}
	for {
		possible := c.findPrevious(start-1, end+1, f.isModifier)
		if possible < 0 {
			return start
		}
		start = possible
	}
}

// hasTraitUse is ClassHelper::getTraitUsePointers being non-empty.
func (c *clCCtx) hasTraitUse(class int) bool {
	level := c.levels[class] + 1
	return c.findNext(c.sc.opener[class]+1, c.sc.closer[class], func(i int) bool {
		return c.sc.code[i] == "use" && c.levels[i] == level
	}) >= 0
}

// hasAttribute is AttributeHelper::hasAttribute, which only reads the first of
// the attribute groups before pointer.
func (c *clCCtx) hasAttribute(pointer int, name string) bool {
	f := c.f
	opener := -1
	from := pointer - 1
	for {
		found := c.findPrevious(from, 0, func(j int) bool {
			return c.isAttributeEnd(j) || f.is(j, ";") || f.is(j, "{") || f.is(j, "}")
		})
		if found < 0 || !c.isAttributeEnd(found) {
			break
		}
		opener = f.toks[found].closer
		from = opener
	}
	if opener < 0 {
		return false
	}
	for _, a := range f.attributes(opener) {
		if f.useStatements().resolveClassName(f, a.name, opener) == name {
			return true
		}
	}
	return false
}

func (c *clCCtx) removeReadonly(readonly int) {
	f := c.f
	f.replaceToken(readonly, "")
	if readonly+1 < len(f.toks) && f.toks[readonly+1].kind == token.Whitespace {
		f.replaceToken(readonly+1, "")
	}
}
