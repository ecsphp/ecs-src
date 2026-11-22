package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/TraitUseDeclarationSniff.php
// SlevomatTraitUseDeclaration splits "use A, B;" into one use per trait.
type SlevomatTraitUseDeclaration struct{}

func (SlevomatTraitUseDeclaration) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\TraitUseDeclarationSniff`
}

func (SlevomatTraitUseDeclaration) SourceURL() string {
	return fixer.SourceURLFor(SlevomatTraitUseDeclaration{}.Name())
}

func (SlevomatTraitUseDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.clDIsTraitUser, func(cls int) int {
			for _, use := range f.clDTraitUsePointers(cls) {
				f.clDCheckDeclaration(use)
			}
			return 0
		})
	})
}

func (f *slvFile) clDCheckDeclaration(use int) {
	if f.clDNextLocal(use+1, func(i int) bool { return f.is(i, ",") }) < 0 {
		return
	}
	end := f.clDNext(use+1, func(i int) bool { return f.is(i, "{") || f.is(i, ";") })
	if end < 0 || f.is(end, "{") {
		return
	}
	var indentation strings.Builder
	for cur := use - 1; cur >= 0 && f.toks[cur].kind == token.Whitespace && f.toks[cur].content != f.eol; cur-- {
		indentation.WriteString(f.toks[cur].content)
	}
	f.beginChangeset()
	for i := use + 1; i < end; i++ {
		if !f.is(i, ",") {
			continue
		}
		after := f.nextEffective(i+1, len(f.toks))
		f.change(i, after-1, ";"+f.eol+indentation.String()+"use ")
	}
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/TraitUseOrderSniff.php
// SlevomatTraitUseOrder sorts the trait uses of a class alphabetically.
type SlevomatTraitUseOrder struct {
	caseSensitive bool
}

func (SlevomatTraitUseOrder) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\TraitUseOrderSniff`
}

func (SlevomatTraitUseOrder) SourceURL() string {
	return fixer.SourceURLFor(SlevomatTraitUseOrder{}.Name())
}

func (r SlevomatTraitUseOrder) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["caseSensitive"].(bool); ok {
		r.caseSensitive = v
	}
	return r
}

type clDUse struct {
	name       string
	start, end int
}

func (r SlevomatTraitUseOrder) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.clDIsTraitUser, func(cls int) int {
			r.process(f, cls)
			return 0
		})
	})
}

func (r SlevomatTraitUseOrder) compare(a, b string) int {
	return SlevomatCatchExceptionsOrder(r).compare(a, b)
}

func (r SlevomatTraitUseOrder) process(f *slvFile, cls int) {
	pointers := f.clDTraitUsePointers(cls)
	if len(pointers) == 0 {
		return
	}
	for _, use := range pointers {
		r.processCommaSeparated(f, use)
	}
	if len(pointers) <= 1 {
		return
	}
	uses := make([]clDUse, 0, len(pointers))
	for _, use := range pointers {
		nameStart := f.clDNext(use+1, f.isName)
		end := f.clDUseEnd(use)
		if nameStart < 0 || end < 0 {
			return
		}
		start := use
		prev := f.prevNonWhitespace(use - 1)
		switch {
		case prev >= 0 && f.toks[prev].doc == slvDocClose:
			start = f.clDDocOpener(prev)
		case prev >= 0 && f.toks[prev].kind == token.Comment:
			start = prev
			for {
				p := f.prevNonWhitespace(start - 1)
				if p < 0 || f.toks[p].kind != token.Comment {
					break
				}
				start = p
			}
		}
		uses = append(uses, clDUse{name: f.toks[nameStart].content, start: start, end: end})
	}
	sorted := slices.Clone(uses)
	slices.SortStableFunc(sorted, func(a, b clDUse) int { return r.compare(a.name, b.name) })
	isSorted := true
	for i := range uses {
		if uses[i].name != sorted[i].name {
			isSorted = false
			break
		}
	}
	if isSorted {
		return
	}
	contents := make([]string, len(sorted))
	for i, u := range sorted {
		contents[i] = f.content(u.start, u.end)
	}
	f.beginChangeset()
	for i, u := range uses {
		f.change(u.start, u.end, contents[i])
	}
	f.endChangeset()
}

func (r SlevomatTraitUseOrder) processCommaSeparated(f *slvFile, use int) {
	end := f.clDNextLocal(use+1, func(i int) bool { return f.is(i, ";") || f.is(i, "{") })
	if end < 0 {
		return
	}
	var namePointers []int
	for i := use + 1; i < end; i++ {
		if f.isName(i) {
			namePointers = append(namePointers, i)
		}
	}
	if len(namePointers) <= 1 {
		return
	}
	names := make([]string, len(namePointers))
	for i, p := range namePointers {
		names[i] = f.toks[p].content
	}
	sorted := slices.Clone(names)
	slices.SortStableFunc(sorted, r.compare)
	if slices.Equal(names, sorted) {
		return
	}
	f.beginChangeset()
	for i, p := range namePointers {
		f.replaceToken(p, sorted[i])
	}
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/TraitUseSpacingSniff.php
// SlevomatTraitUseSpacing enforces the blank lines before, between and after
// the trait uses of a class.
type SlevomatTraitUseSpacing struct {
	before              int
	beforeWhenFirst     int
	beforeWhenFirstNull bool
	between             int
	afterLast           int
	afterLastNull       bool
	afterLastWhenLast   int
	configured          bool
}

func (SlevomatTraitUseSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\TraitUseSpacingSniff`
}

func (SlevomatTraitUseSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatTraitUseSpacing{}.Name())
}

func (r SlevomatTraitUseSpacing) withDefaults() SlevomatTraitUseSpacing {
	if !r.configured {
		r = SlevomatTraitUseSpacing{
			before: 1, beforeWhenFirstNull: true, between: 0, afterLast: 1, afterLastWhenLast: 1, configured: true,
		}
	}
	return r
}

func (r SlevomatTraitUseSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r = r.withDefaults()
	r.before = configInt(config, "linesCountBeforeFirstUse", r.before)
	r.beforeWhenFirst, r.beforeWhenFirstNull = clDNullableInt(config, "linesCountBeforeFirstUseWhenFirstInClass", r.beforeWhenFirst, r.beforeWhenFirstNull)
	r.between = configInt(config, "linesCountBetweenUses", r.between)
	r.afterLast, r.afterLastNull = clDNullableInt(config, "linesCountAfterLastUse", r.afterLast, r.afterLastNull)
	r.afterLastWhenLast = configInt(config, "linesCountAfterLastUseWhenLastInClass", r.afterLastWhenLast)
	return r
}

// clDNullableInt reads a ?int sniff setting: a null value clears it.
func clDNullableInt(config map[string]any, key string, def int, defNull bool) (int, bool) {
	v, ok := config[key]
	if !ok {
		return def, defNull
	}
	if v == nil {
		return 0, true
	}
	return configInt(config, key, def), false
}

func (r SlevomatTraitUseSpacing) Fix(s *tokens.Stream) bool {
	r = r.withDefaults()
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.clDIsTraitUser, func(cls int) int {
			uses := f.clDTraitUsePointers(cls)
			if len(uses) == 0 {
				return 0
			}
			r.checkBeforeFirst(f, uses[0])
			r.checkAfterLast(f, uses[len(uses)-1])
			r.checkBetween(f, uses)
			return 0
		})
	})
}

func (r SlevomatTraitUseSpacing) checkBeforeFirst(f *slvFile, first int) {
	useStart := first
	before := f.prevNonWhitespace(first - 1)
	if before < 0 {
		return
	}
	if f.nsuIsComment(before) {
		before = f.prevEffective(before - 1)
		useStart = f.clDNext(before+1, f.nsuIsComment)
	}
	isStart := f.is(before, "{")
	ws := ""
	if before+1 != first {
		ws += f.content(before+1, useStart-1)
	}
	required := r.before
	if isStart && !r.beforeWhenFirstNull {
		required = r.beforeWhenFirst
	}
	if strings.Count(ws, f.eol)-1 == required {
		return
	}
	indent := f.clDPrevContent(first, before, f.eol)
	f.beginChangeset()
	if indent >= 0 {
		f.removeBetweenIncluding(before+1, indent)
	}
	for range required + 1 {
		f.clDAddNewline(before)
	}
	f.endChangeset()
}

func (r SlevomatTraitUseSpacing) checkAfterLast(f *slvFile, last int) {
	end := f.clDUseEnd(last)
	if end < 0 {
		return
	}
	after := f.nextEffective(end+1, len(f.toks))
	isEnd := f.is(after, "}")
	wsEnd := f.nextNonWhitespace(end+1, len(f.toks)) - 1
	if wsEnd < 0 {
		return
	}
	if end != wsEnd && f.toks[wsEnd].content != f.eol {
		if p := f.clDPrevContent(wsEnd-1, end, f.eol); p >= 0 {
			wsEnd = p
		} else {
			wsEnd = end
		}
	}
	ws := f.content(end+1, wsEnd)
	required := r.afterLast
	if isEnd {
		required = r.afterLastWhenLast
	} else if r.afterLastNull {
		return
	}
	if strings.Count(ws, f.eol)-1 == required {
		return
	}
	f.beginChangeset()
	f.removeBetweenIncluding(end+1, wsEnd)
	for range required + 1 {
		f.clDAddNewline(end)
	}
	f.endChangeset()
}

func (r SlevomatTraitUseSpacing) checkBetween(f *slvFile, uses []int) {
	if len(uses) == 1 {
		return
	}
	for n := 1; n < len(uses); n++ {
		prevUse, use := uses[n-1], uses[n]
		prevEnd := f.clDUseEnd(prevUse)
		if prevEnd < 0 {
			continue
		}
		useStart := use
		before := f.prevNonWhitespace(use - 1)
		if f.nsuIsComment(before) {
			useStart = f.clDNext(f.prevEffective(before-1)+1, f.nsuIsComment)
		}
		if f.toks[useStart].line-f.toks[prevEnd].line-1 == r.between {
			continue
		}
		if prevEnd != f.prevEffective(use-1) {
			continue
		}
		indent := f.clDPrevContent(use, prevEnd, f.eol)
		f.beginChangeset()
		if indent >= 0 {
			f.removeBetweenIncluding(prevEnd+1, indent)
		}
		for range r.between + 1 {
			f.clDAddNewline(prevEnd)
		}
		f.endChangeset()
	}
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Classes/EmptyLinesAroundClassBracesSniff.php
// SlevomatEmptyLinesAroundClassBraces enforces the blank lines after a class's
// opening brace and before its closing brace.
type SlevomatEmptyLinesAroundClassBraces struct {
	afterOpening, beforeClosing int
	configured                  bool
}

func (SlevomatEmptyLinesAroundClassBraces) Name() string {
	return `SlevomatCodingStandard\Sniffs\Classes\EmptyLinesAroundClassBracesSniff`
}

func (SlevomatEmptyLinesAroundClassBraces) SourceURL() string {
	return fixer.SourceURLFor(SlevomatEmptyLinesAroundClassBraces{}.Name())
}

func (r SlevomatEmptyLinesAroundClassBraces) withDefaults() SlevomatEmptyLinesAroundClassBraces {
	if !r.configured {
		r.afterOpening, r.beforeClosing, r.configured = 1, 1, true
	}
	return r
}

func (r SlevomatEmptyLinesAroundClassBraces) WithConfig(config map[string]any) fixer.Fixer {
	r = r.withDefaults()
	r.afterOpening = configInt(config, "linesCountAfterOpeningBrace", r.afterOpening)
	r.beforeClosing = configInt(config, "linesCountBeforeClosingBrace", r.beforeClosing)
	return r
}

func (r SlevomatEmptyLinesAroundClassBraces) Fix(s *tokens.Stream) bool {
	r = r.withDefaults()
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.clDIsTraitUser(i) || f.isKeyword(i, "interface") }, func(cls int) int {
			closer := f.scopeCloser(cls)
			if closer < 0 || f.toks[closer].closer < 0 {
				return 0
			}
			r.processOpening(f, f.toks[closer].closer)
			r.processClosing(f, closer)
			return 0
		})
	})
}

func (r SlevomatEmptyLinesAroundClassBraces) processOpening(f *slvFile, opener int) {
	next := f.nextNonWhitespace(opener+1, len(f.toks))
	if next < 0 {
		return
	}
	lines := f.toks[next].line - f.toks[opener].line - 1
	if lines == r.afterOpening {
		return
	}
	f.beginChangeset()
	if lines < r.afterOpening {
		for range r.afterOpening - lines {
			f.clDAddNewline(opener)
		}
	} else {
		for i := opener + r.afterOpening + 2; i < next; i++ {
			if f.tokenContent(i) != f.eol {
				break
			}
			f.replace(i, "")
		}
	}
	f.endChangeset()
}

func (r SlevomatEmptyLinesAroundClassBraces) processClosing(f *slvFile, closer int) {
	prev := f.prevNonWhitespace(closer - 1)
	if prev < 0 {
		return
	}
	lines := f.toks[closer].line - f.toks[prev].line - 1
	if lines == r.beforeClosing {
		return
	}
	f.beginChangeset()
	if lines < r.beforeClosing {
		for range r.beforeClosing - lines {
			f.replaceToken(closer, f.eol+f.tokenContent(closer))
		}
	} else {
		f.removeBetweenIncluding(prev+r.beforeClosing+2, closer-1)
	}
	f.endChangeset()
}

// clDIsTraitUser tells the T_CLASS, T_ANON_CLASS, T_TRAIT and T_ENUM tokens.
func (f *slvFile) clDIsTraitUser(i int) bool {
	if !f.isKeyword(i, "class") && !f.isKeyword(i, "trait") && !f.isKeyword(i, "enum") {
		return false
	}
	return !f.is(f.prevEffective(i-1), "::")
}

// clDTraitUsePointers is ClassHelper::getTraitUsePointers.
func (f *slvFile) clDTraitUsePointers(cls int) []int {
	closer := f.scopeCloser(cls)
	if closer < 0 || f.toks[closer].closer < 0 {
		return nil
	}
	opener := f.toks[closer].closer
	var out []int
	for i := opener + 1; i < closer; i++ {
		if f.isKeyword(i, "use") && f.enclosingScope(i) == opener {
			out = append(out, i)
		}
	}
	return out
}

// clDUseEnd is the ";" ending a trait use, or the "}" of its block.
func (f *slvFile) clDUseEnd(use int) int {
	end := f.clDNextLocal(use+1, func(i int) bool { return f.is(i, ";") || f.is(i, "{") })
	if end >= 0 && f.is(end, "{") {
		end = f.toks[end].closer
	}
	return end
}

// clDNext is TokenHelper::findNext.
func (f *slvFile) clDNext(start int, match func(i int) bool) int {
	for i := max(start, 0); i < len(f.toks); i++ {
		if match(i) {
			return i
		}
	}
	return -1
}

// clDNextLocal is TokenHelper::findNextLocal: the search stops after a ";".
func (f *slvFile) clDNextLocal(start int, match func(i int) bool) int {
	for i := max(start, 0); i < len(f.toks); i++ {
		if match(i) {
			return i
		}
		if f.is(i, ";") {
			return -1
		}
	}
	return -1
}

// clDPrevContent is TokenHelper::findPreviousContent for T_WHITESPACE: it
// searches from start back to end, both inclusive.
func (f *slvFile) clDPrevContent(start, end int, content string) int {
	for i := start; i >= max(end, 0); i-- {
		if f.toks[i].kind == token.Whitespace && f.toks[i].content == content {
			return i
		}
	}
	return -1
}

// clDAddNewline is Fixer::addNewline.
func (f *slvFile) clDAddNewline(i int) {
	f.replaceToken(i, f.tokenContent(i)+f.eol)
}

// clDDocOpener is a docblock close tag's comment_opener.
func (f *slvFile) clDDocOpener(closeTag int) int {
	for j := closeTag; j >= 0; j-- {
		if f.toks[j].doc == slvDocOpen {
			return j
		}
	}
	return closeTag
}
