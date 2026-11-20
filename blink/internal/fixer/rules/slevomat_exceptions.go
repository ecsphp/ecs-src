package rules

import (
	"slices"
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// excCatchParens returns the parenthesis_opener and parenthesis_closer of a
// "catch", or -1, -1.
func (f *slvFile) excCatchParens(catch int) (int, int) {
	open := f.nextEffective(catch+1, len(f.toks))
	if !f.is(open, "(") || f.toks[open].closer < 0 {
		return -1, -1
	}
	return open, f.toks[open].closer
}

// excCatchScope returns the scope_opener and scope_closer of a "catch" or
// "finally", or -1, -1.
func (f *slvFile) excCatchScope(keyword int) (int, int) {
	from := keyword + 1
	if f.isKeyword(keyword, "catch") {
		_, closeIdx := f.excCatchParens(keyword)
		if closeIdx < 0 {
			return -1, -1
		}
		from = closeIdx + 1
	}
	open := f.nextEffective(from, len(f.toks))
	if !f.is(open, "{") || f.toks[open].closer < 0 {
		return -1, -1
	}
	return open, f.toks[open].closer
}

// excCaughtNames lists the names of the types a catch catches, as
// CatchHelper::findCaughtTypesInCatch walks them.
func (f *slvFile) excCaughtNames(open, closeIdx int) [][2]int {
	var out [][2]int
	end := open
	for {
		start := -1
		for j := end + 1; j < closeIdx; j++ {
			if f.is(j, "|") || f.isName(j) {
				start = j
				break
			}
		}
		if start < 0 {
			return out
		}
		if f.is(start, "|") {
			start = f.nextEffective(start+1, closeIdx)
			if start < 0 {
				return out
			}
		}
		end = start
		for end+1 < len(f.toks) && f.isName(end+1) {
			end++
		}
		out = append(out, [2]int{start, end})
	}
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Exceptions/CatchExceptionsOrderSniff.php
//
// SlevomatCatchExceptionsOrder sorts the types a catch catches alphabetically.
type SlevomatCatchExceptionsOrder struct {
	caseSensitive bool
}

func (SlevomatCatchExceptionsOrder) Name() string {
	return `SlevomatCodingStandard\Sniffs\Exceptions\CatchExceptionsOrderSniff`
}

func (SlevomatCatchExceptionsOrder) SourceURL() string {
	return fixer.SourceURLFor(SlevomatCatchExceptionsOrder{}.Name())
}

func (r SlevomatCatchExceptionsOrder) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["caseSensitive"].(bool); ok {
		r.caseSensitive = v
	}
	return r
}

// compare is NamespaceHelper::compareStatements.
func (r SlevomatCatchExceptionsOrder) compare(a, b string) int {
	ap := strings.Split(strings.TrimLeft(a, `\`), `\`)
	bp := strings.Split(strings.TrimLeft(b, `\`), `\`)
	for n := range min(len(ap), len(bp)) {
		var c int
		if r.caseSensitive {
			c = strings.Compare(ap[n], bp[n])
		} else {
			c = nsuStrcasecmp(ap[n], bp[n])
		}
		if c != 0 {
			return c
		}
	}
	return len(ap) - len(bp)
}

func (r SlevomatCatchExceptionsOrder) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if !f.isKeyword(i, "catch") {
				continue
			}
			open, closeIdx := f.excCatchParens(i)
			if open < 0 {
				continue
			}
			names := f.excCaughtNames(open, closeIdx)
			if len(names) <= 1 {
				continue
			}
			original := make([]string, len(names))
			for n, p := range names {
				original[n] = f.content(p[0], p[1])
			}
			sorted := slices.Clone(original)
			sort.SliceStable(sorted, func(a, b int) bool { return r.compare(sorted[a], sorted[b]) < 0 })
			if slices.Equal(sorted, original) {
				continue
			}
			f.beginChangeset()
			for n, p := range names {
				f.change(p[0], p[1], sorted[n])
			}
			f.endChangeset()
		}
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Exceptions/ReferenceThrowableOnlySniff.php
//
// SlevomatReferenceThrowableOnly references \Throwable instead of \Exception,
// except after extends, new and instanceof.
type SlevomatReferenceThrowableOnly struct{}

func (SlevomatReferenceThrowableOnly) Name() string {
	return `SlevomatCodingStandard\Sniffs\Exceptions\ReferenceThrowableOnlySniff`
}

func (SlevomatReferenceThrowableOnly) SourceURL() string {
	return fixer.SourceURLFor(SlevomatReferenceThrowableOnly{}.Name())
}

const excReferenceThrowableOnlyCode = "SlevomatCodingStandard.Exceptions.ReferenceThrowableOnly.ReferencedGeneralException"

func (SlevomatReferenceThrowableOnly) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		openTag := f.slvFirstOpenTag()
		if openTag < 0 {
			return
		}
		uses := f.useStatements()
		names := newSlvNames(f)
		docs := newSlvDocs(f)
		isBitwiseOr := func(i int) bool { return f.is(i, "|") && !names.typeOps[i] }
		for _, ref := range names.referencedNames(openTag) {
			if uses.resolveName(f, ref.name, slvRefClass, ref.start) != `\Exception` {
				continue
			}
			prev := f.prevEffective(ref.start - 1)
			if f.isKeyword(prev, "extends") || f.isKeyword(prev, "new") || f.isKeyword(prev, "instanceof") {
				continue
			}
			if isBitwiseOr(prev) {
				for prev--; prev >= 0 && (f.isEmpty(prev) || f.isName(prev) || isBitwiseOr(prev)); prev-- {
				}
			}
			if f.is(prev, "(") {
				opener := f.prevEffective(prev - 1)
				if f.isKeyword(opener, "catch") {
					if f.excThrowableInNextCatches(uses, opener) {
						continue
					}
				} else if f.excIsFunctionParens(prev) && f.toks[prev].closer > ref.start &&
					f.excIsSuppressed(docs, opener, excReferenceThrowableOnlyCode) {
					continue
				}
			}
			f.beginChangeset()
			f.change(ref.start, ref.end, `\Throwable`)
			f.endChangeset()
		}
	})
}

// excIsFunctionParens tells parentheses whose parenthesis_owner is T_FUNCTION,
// the parameters of a named function (a closure's owner is T_CLOSURE).
func (f *slvFile) excIsFunctionParens(open int) bool {
	p := f.prevEffective(open - 1)
	if p < 0 || f.toks[p].kind != token.Ident && f.toks[p].kind != token.Keyword {
		return false
	}
	q := f.prevEffective(p - 1)
	if f.is(q, "&") {
		q = f.prevEffective(q - 1)
	}
	return f.isKeyword(q, "function")
}

// excThrowableInNextCatches tells whether a catch after the one at catch
// catches \Throwable.
func (f *slvFile) excThrowableInNextCatches(uses *slvUseStatements, catch int) bool {
	_, closer := f.excCatchScope(catch)
	for closer >= 0 {
		next := f.nextEffective(closer+1, len(f.toks))
		if !f.isKeyword(next, "catch") {
			return false
		}
		open, closeIdx := f.excCatchParens(next)
		if open < 0 {
			return false
		}
		for _, p := range f.excCaughtNames(open, closeIdx) {
			if uses.resolveName(f, f.content(p[0], p[1]), slvRefClass, open) == `\Throwable` {
				return true
			}
		}
		_, closer = f.excCatchScope(next)
	}
	return false
}

// excIsSuppressed is SuppressHelper::isSniffSuppressed.
func (f *slvFile) excIsSuppressed(docs *slvDocs, pointer int, code string) bool {
	open := f.docOpenBefore(pointer)
	if open < 0 {
		return false
	}
	doc := docs.parse(open)
	for _, a := range docs.annotations(open, "@phpcsSuppress") {
		value := a.node.value()
		if value.kind != "GenericTagValueNode" {
			continue
		}
		name, _, _ := strings.Cut(uuDocText(doc, value), " ")
		if code == name || strings.HasPrefix(code, name+".") {
			return true
		}
	}
	return false
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Exceptions/RequireNonCapturingCatchSniff.php
//
// SlevomatRequireNonCapturingCatch drops the variable of a catch that never
// uses it.
type SlevomatRequireNonCapturingCatch struct {
	disabled bool
}

func (SlevomatRequireNonCapturingCatch) Name() string {
	return `SlevomatCodingStandard\Sniffs\Exceptions\RequireNonCapturingCatchSniff`
}

func (SlevomatRequireNonCapturingCatch) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireNonCapturingCatch{}.Name())
}

func (r SlevomatRequireNonCapturingCatch) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["enable"].(bool); ok {
		r.disabled = !v
	}
	return r
}

func (r SlevomatRequireNonCapturingCatch) Fix(s *tokens.Stream) bool {
	if r.disabled {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		scopes := slvcFunctionScopes(f)
		strs := slvcStringKinds(f)
		used := func(start, end int, name string) bool {
			first := start + 1
			for i := first; i <= end && i < len(f.toks); i++ {
				if f.isVariable(i) {
					if f.toks[i].content != name || f.closureIsParameter(i) || scopes[first] != scopes[i] {
						continue
					}
					catch := -1
					for j := i - 1; j >= first; j-- {
						if f.isKeyword(j, "catch") {
							catch = j
							break
						}
					}
					if catch < 0 {
						return true
					}
					if _, closeIdx := f.excCatchParens(catch); closeIdx < i {
						return true
					}
				} else if strs[i] != slvcOtherString && slvcUsedInString(name, f.toks[i].content) {
					return true
				}
			}
			return false
		}
		for i := range f.toks {
			if !f.isKeyword(i, "catch") {
				continue
			}
			open, closeIdx := f.excCatchParens(i)
			bodyOpen, bodyClose := f.excCatchScope(i)
			if open < 0 || bodyOpen < 0 {
				continue
			}
			variable := -1
			for j := open; j < closeIdx; j++ {
				if f.isVariable(j) {
					variable = j
					break
				}
			}
			if variable < 0 {
				continue
			}
			name := f.toks[variable].content
			if used(bodyOpen, bodyClose, name) {
				continue
			}
			tryEnd, owner := bodyClose, i
			for {
				next := f.nextEffective(tryEnd+1, len(f.toks))
				if !f.isKeyword(next, "catch") && !f.isKeyword(next, "finally") {
					break
				}
				_, c := f.excCatchScope(next)
				if c < 0 {
					break
				}
				tryEnd, owner = c, next
			}
			if f.isKeyword(owner, "finally") {
				if o, c := f.excCatchScope(owner); used(o, c, name) {
					continue
				}
			}
			scopeEnd := len(f.toks) - 1
			if fn := scopes[tryEnd]; fn >= 0 {
				scopeEnd = f.toks[fn].closer
			}
			if used(tryEnd, scopeEnd, name) {
				continue
			}
			before := f.prevEffective(variable - 1)
			end := closeIdx
			for j := variable + 1; j < closeIdx; j++ {
				if f.toks[j].kind == token.Whitespace && f.toks[j].content == f.eol {
					end = j
					break
				}
			}
			f.beginChangeset()
			if before+1 <= end-1 {
				f.removeBetweenIncluding(before+1, end-1)
			}
			f.endChangeset()
		}
	})
}
