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

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/StaticClosureSniff.php
//
// SlevomatStaticClosure makes a closure or arrow function static when it uses
// neither $this nor parent.
type SlevomatStaticClosure struct{}

func (SlevomatStaticClosure) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\StaticClosureSniff`
}

func (SlevomatStaticClosure) SourceURL() string {
	return fixer.SourceURLFor(SlevomatStaticClosure{}.Name())
}

func (SlevomatStaticClosure) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		strs := slvcStringKinds(f)
		isClosureOrFn := func(i int) bool { return f.closureIsClosure(i) || f.closureIsFn(i) }
		f.eachToken(isClosureOrFn, func(i int) int {
			prev := f.prevEffective(i - 1)
			if f.isKeyword(prev, "static") {
				return 0
			}
			if f.is(prev, "(") {
				p := f.prevEffective(prev - 1)
				if p >= 0 && f.toks[p].kind == token.Ident && f.toks[p].orig == "bind" {
					return 0
				}
			}
			opener, closer := f.closureScope(i)
			if opener < 0 {
				return 0
			}
			for j := opener + 1; j <= closer && j < len(f.toks); j++ {
				if f.toks[j].kind == token.Variable && f.toks[j].orig == "$this" {
					return 0
				}
			}
			for j := opener + 1; j < closer; j++ {
				if strs[j] == slvcDoubleQuoted && slvcUsedInString("$this", f.toks[j].content) {
					return 0
				}
			}
			for j := opener + 1; j < closer; j++ {
				if f.toks[j].kind == token.Ident && strings.EqualFold(f.toks[j].orig, "parent") && !f.closureIsMemberName(j) {
					return 0
				}
			}
			f.beginChangeset()
			f.addBefore(i, "static ")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/UnusedInheritedVariablePassedToClosureSniff.php
//
// SlevomatUnusedInheritedVariablePassedToClosure removes the variables of a
// closure's "use" that its body never uses, and the whole "use" when none is left.
type SlevomatUnusedInheritedVariablePassedToClosure struct{}

func (SlevomatUnusedInheritedVariablePassedToClosure) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\UnusedInheritedVariablePassedToClosureSniff`
}

func (SlevomatUnusedInheritedVariablePassedToClosure) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUnusedInheritedVariablePassedToClosure{}.Name())
}

func (SlevomatUnusedInheritedVariablePassedToClosure) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		scopes := slvcFunctionScopes(f)
		strs := slvcStringKinds(f)
		isUse := func(i int) bool {
			if !f.isKeyword(i, "use") {
				return false
			}
			o := f.nextEffective(i+1, len(f.toks))
			return f.is(o, "(") && f.toks[o].closer >= 0
		}
		f.eachToken(isUse, func(use int) int {
			open := f.nextEffective(use+1, len(f.toks))
			closeIdx := f.toks[open].closer
			closure := -1
			for j := use - 1; j >= 0; j-- {
				if f.closureIsClosure(j) {
					closure = j
					break
				}
			}
			if closure < 0 {
				return 0
			}
			for v := open + 1; v < closeIdx; v++ {
				if f.toks[v].kind != token.Variable {
					continue
				}
				if f.closureIsUsedInScope(closure, v, scopes, strs) {
					continue
				}
				f.closureRemoveInherited(use, open, closeIdx, v)
			}
			return 0
		})
	})
}

func (f *slvFile) closureRemoveInherited(use, open, closeIdx, v int) {
	start := v
	for !f.is(start-1, "(") {
		start--
		if f.is(start, ",") {
			break
		}
	}
	end := v
	for {
		next := end + 1
		if f.is(next, ")") || f.is(next, ",") && f.is(start, ",") ||
			f.toks[next].kind == token.Variable || f.is(next, "&") {
			break
		}
		end++
	}
	f.beginChangeset()
	for k := start; k <= end; k++ {
		f.replaceToken(k, "")
	}
	empty := true
	for k := open + 1; k < closeIdx; k++ {
		if f.tokenContent(k) != "" {
			empty = false
			break
		}
	}
	if empty {
		for k := use; k <= closeIdx; k++ {
			f.replaceToken(k, "")
		}
	}
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Functions/RequireArrowFunctionSniff.php
//
// SlevomatRequireArrowFunction turns a closure whose body is a single return
// into an arrow function, unless it inherits a variable by reference or, with
// allowNested false, holds another closure or arrow function.
type SlevomatRequireArrowFunction struct {
	disallowNested bool
	disabled       bool
}

func (SlevomatRequireArrowFunction) Name() string {
	return `SlevomatCodingStandard\Sniffs\Functions\RequireArrowFunctionSniff`
}

func (SlevomatRequireArrowFunction) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireArrowFunction{}.Name())
}

func (r SlevomatRequireArrowFunction) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["allowNested"].(bool); ok {
		r.disallowNested = !v
	}
	if v, ok := config["enable"].(bool); ok {
		r.disabled = !v
	}
	return r
}

func (r SlevomatRequireArrowFunction) Fix(s *tokens.Stream) bool {
	if r.disabled {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		scopes := slvcFunctionScopes(f)
		f.eachToken(f.closureIsClosure, func(i int) int {
			opener, closer := f.closureScope(i)
			if opener < 0 {
				return 0
			}
			ret := f.nextEffective(opener+1, len(f.toks))
			if !f.isKeyword(ret, "return") {
				return 0
			}
			paramsClose := f.closureParamsClose(i)
			if use := f.nextEffective(paramsClose+1, len(f.toks)); f.isKeyword(use, "use") {
				if o := f.nextEffective(use+1, len(f.toks)); f.is(o, "(") && f.toks[o].closer >= 0 {
					for k := o + 1; k < f.toks[o].closer; k++ {
						if f.is(k, "&") {
							return 0
						}
					}
				}
			}
			if r.disallowNested {
				for k := opener + 1; k < closer; k++ {
					if f.closureIsClosure(k) || f.isKeyword(k, "fn") {
						return 0
					}
				}
			}

			afterReturn := f.nextNonWhitespace(ret+1, len(f.toks))
			semicolon := -1
			for k := ret + 1; k < len(f.toks)-1; k++ {
				if f.is(k, ";") && scopes[k] == scopes[ret] {
					semicolon = k
					break
				}
			}
			use := -1
			for k := paramsClose + 1; k < opener; k++ {
				if f.isKeyword(k, "use") {
					use = k
					break
				}
			}
			beforeOpener := opener - 1
			for beforeOpener >= 0 && f.toks[beforeOpener].kind == token.Whitespace {
				beforeOpener--
			}
			afterUse := -1
			if use >= 0 {
				if o := f.nextEffective(use+1, len(f.toks)); f.is(o, "(") && f.toks[o].closer >= 0 {
					afterUse = f.nextNonWhitespace(f.toks[o].closer+1, len(f.toks))
				}
			}

			f.beginChangeset()
			f.replace(i, "fn")
			if afterUse >= 0 {
				f.closureRemove(paramsClose+1, afterUse-1)
			}
			f.closureRemove(beforeOpener+1, afterReturn-1)
			f.add(beforeOpener, " => ")
			f.closureRemove(semicolon, closer)
			f.endChangeset()
			return 0
		})
	})
}

// closureIsMemberName tells a name PHP_CodeSniffer keeps T_STRING (a member
// or function name) from a keyword like parent.
func (f *slvFile) closureIsMemberName(i int) bool {
	p := f.prevEffective(i - 1)
	return f.is(p, "->") || f.is(p, "?->") || f.is(p, "::") || f.isKeyword(p, "function")
}

func (f *slvFile) closureRemove(start, end int) {
	for k := start; k <= end; k++ {
		f.replaceToken(k, "")
	}
}

// closureIsClosure tells T_CLOSURE: a "function" keyword opening an anonymous function.
func (f *slvFile) closureIsClosure(i int) bool {
	return f.isKeyword(i, "function") && f.isClosure(i)
}

// closureIsFn tells T_FN.
func (f *slvFile) closureIsFn(i int) bool {
	return f.isKeyword(i, "fn") && f.arrowEnd(i) >= 0
}

// closureParamsClose returns the ")" closing the parameters of a closure or arrow function.
func (f *slvFile) closureParamsClose(fn int) int {
	o := f.nextEffective(fn+1, len(f.toks))
	if f.is(o, "&") {
		o = f.nextEffective(o+1, len(f.toks))
	}
	if !f.is(o, "(") {
		return -1
	}
	return f.toks[o].closer
}

// closureScope returns the scope_opener and scope_closer of a closure ("{",
// "}") or an arrow function ("=>", its expression end), or -1, -1.
func (f *slvFile) closureScope(fn int) (int, int) {
	if f.isKeyword(fn, "fn") {
		end := f.arrowEnd(fn)
		if end < 0 {
			return -1, -1
		}
		for j := f.closureParamsClose(fn) + 1; j < end; j++ {
			if f.is(j, "=>") {
				return j, end
			}
		}
		return -1, -1
	}
	closer := f.scopeCloser(fn)
	if closer < 0 {
		return -1, -1
	}
	return f.toks[closer].closer, closer
}

// slvcFunctionScopes maps each token to the "{" of the innermost function or
// closure body holding it (-1 outside any), which is what
// ScopeHelper::isInSameScope compares; the braces belong to the outer scope.
func slvcFunctionScopes(f *slvFile) []int {
	scopes := make([]int, len(f.toks))
	type entry struct {
		closer int
		fn     bool
	}
	var stack []entry
	current := func() int {
		for _, e := range slices.Backward(stack) {
			if e.fn {
				return f.toks[e.closer].closer
			}
		}
		return -1
	}
	for i := range f.toks {
		for len(stack) > 0 && stack[len(stack)-1].closer <= i {
			stack = stack[:len(stack)-1]
		}
		scopes[i] = current()
		if f.is(i, "{") && f.toks[i].closer > i {
			owner := -1
			if f.closesScope(f.toks[i].closer) {
				owner = f.braceOwner(i)
			}
			stack = append(stack, entry{closer: f.toks[i].closer, fn: owner >= 0 && f.isKeyword(owner, "function")})
		}
	}
	return scopes
}

const (
	slvcOtherString uint8 = iota
	slvcDoubleQuoted
	slvcHeredoc
)

// slvcStringKinds marks the string tokens PHP_CodeSniffer types
// T_DOUBLE_QUOTED_STRING (an interpolated "...") and T_HEREDOC (a heredoc body line).
func slvcStringKinds(f *slvFile) []uint8 {
	kinds := make([]uint8, len(f.toks))
	for i := 0; i < len(f.toks); {
		if f.toks[i].kind != token.String {
			i++
			continue
		}
		end := i
		var b strings.Builder
		for end < len(f.toks) && f.toks[end].kind == token.String && (end == i || f.toks[end].heredoc == f.toks[i].heredoc) {
			b.WriteString(f.toks[end].orig)
			end++
			if f.toks[i].heredoc && !strings.Contains(f.toks[end-1].orig, "\n") {
				break
			}
		}
		text := b.String()
		_, rest := token.SplitBinaryPrefix(text)
		switch {
		case f.toks[i].heredoc:
			if !strings.HasPrefix(strings.TrimLeft(rest[3:], " \t"), "'") {
				for k := i + 1; k < end-1; k++ {
					kinds[k] = slvcHeredoc
				}
			}
		case strings.HasPrefix(rest, `"`):
			for _, t := range lexer.Lex("<?php " + text) {
				if t.Kind == token.String && t.Part != token.Whole {
					for k := i; k < end; k++ {
						kinds[k] = slvcDoubleQuoted
					}
					break
				}
			}
		}
		i = end
	}
	return kinds
}

// slvcUsedInString is VariableHelper::isUsedInScopeInString.
func slvcUsedInString(name, content string) bool {
	re := regexp.MustCompile(`(\\)?(` + regexp.QuoteMeta(name) + `)\b`)
	if m := re.FindStringSubmatch(content); m != nil {
		if m[1] == "" || len(m[1])%2 == 1 {
			return true
		}
	}
	return regexp.MustCompile(`\$\{` + regexp.QuoteMeta(name[1:]) + `(<=\}|\b)`).MatchString(content)
}

// closureIsParameter is ParameterHelper::isParameter: the innermost parentheses
// holding v are a function's parameter list.
func (f *slvFile) closureIsParameter(v int) bool {
	o := f.enclosing(v)
	for o >= 0 && !f.is(o, "(") {
		o = f.enclosing(o)
	}
	if o < 0 {
		return false
	}
	p := f.prevEffective(o - 1)
	if f.is(p, "&") {
		p = f.prevEffective(p - 1)
	}
	if f.isKeyword(p, "function") || f.isKeyword(p, "fn") {
		return true
	}
	if p >= 0 && f.toks[p].kind == token.Ident {
		q := f.prevEffective(p - 1)
		if f.is(q, "&") {
			q = f.prevEffective(q - 1)
		}
		return f.isKeyword(q, "function")
	}
	return false
}

// closureIsUsedInScope is VariableHelper::isUsedInScope for a closure or
// arrow function owner.
func (f *slvFile) closureIsUsedInScope(owner, v int, scopes []int, strs []uint8) bool {
	opener, closer := f.closureScope(owner)
	if opener < 0 {
		return false
	}
	first, last := opener+1, closer-1
	if f.isKeyword(owner, "fn") {
		first, last = opener, closer
	}
	name := f.toks[v].content
	for i := first; i <= last; i++ {
		if scopes[i] != scopes[first] {
			continue
		}
		t := f.toks[i]
		switch {
		case t.kind == token.Variable:
			if t.content != name {
				continue
			}
			if f.is(i-1, "::") {
				if f.is(f.nextEffective(i+1, len(f.toks)), "(") {
					return true
				}
				continue
			}
			if !f.closureIsParameter(i) {
				return true
			}
		case t.kind == token.Ident:
			fn := strings.ToLower(strings.TrimLeft(t.content, `\`))
			if fn != "get_defined_vars" && fn != "compact" {
				continue
			}
			o := f.nextEffective(i+1, len(f.toks))
			if !f.is(o, "(") {
				continue
			}
			if fn == "get_defined_vars" {
				return true
			}
			re := regexp.MustCompile(`^(['"])` + name[1:] + `(['"])$`)
			for k := o + 1; k < f.toks[o].closer; k++ {
				if m := re.FindStringSubmatch(f.toks[k].content); m != nil && m[1] == m[2] {
					return true
				}
			}
		case strs[i] != slvcOtherString:
			if slvcUsedInString(name, t.content) {
				return true
			}
		}
	}
	return false
}
