package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// slvRunTernaries runs process for every T_INLINE_THEN of a file holding a "?".
func slvRunTernaries(s *tokens.Stream, process func(f *slvFile, t *slvTernary, then int)) bool {
	if !slvHasPunct(s, "?") {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		t := newSlvTernary(f)
		f.eachToken(t.isThen, func(then int) int {
			process(f, t, then)
			return 0
		})
	})
}

func slvHasPunct(s *tokens.Stream, v string) bool {
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.Punct && t.Value == v {
			return true
		}
	}
	return false
}

func (f *slvFile) slvIsBoolLiteral(i int) bool {
	if i < 0 || i >= len(f.toks) || f.toks[i].kind != token.Ident {
		return false
	}
	v := strings.ToLower(f.toks[i].orig)
	return v == "true" || v == "false"
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/DisallowShortTernaryOperatorSniff.php
//
// SlevomatDisallowShortTernaryOperator expands "$a ?: $b" to "$a ? $a : $b"
// when the condition is a variable.
type SlevomatDisallowShortTernaryOperator struct {
	notFixable bool
}

func (SlevomatDisallowShortTernaryOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\DisallowShortTernaryOperatorSniff`
}

func (SlevomatDisallowShortTernaryOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowShortTernaryOperator{}.Name())
}

func (r SlevomatDisallowShortTernaryOperator) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["fixable"].(bool); ok {
		r.notFixable = !v
	}
	return r
}

func (r SlevomatDisallowShortTernaryOperator) Fix(s *tokens.Stream) bool {
	if r.notFixable {
		return false
	}
	return slvRunTernaries(s, func(f *slvFile, t *slvTernary, then int) {
		if !t.isElse(f.nextEffective(then+1, len(f.toks))) {
			return
		}
		prev := f.prevEffective(then - 1)
		if prev < 0 || f.toks[prev].kind != token.Variable {
			return
		}
		f.beginChangeset()
		f.add(then, " "+f.toks[prev].content+" ")
		f.endChangeset()
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireShortTernaryOperatorSniff.php
//
// SlevomatRequireShortTernaryOperator turns "$a ? $a : $b" into "$a ?: $b".
type SlevomatRequireShortTernaryOperator struct{}

func (SlevomatRequireShortTernaryOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireShortTernaryOperatorSniff`
}

func (SlevomatRequireShortTernaryOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireShortTernaryOperator{}.Name())
}

func (SlevomatRequireShortTernaryOperator) Fix(s *tokens.Stream) bool {
	return slvRunTernaries(s, func(f *slvFile, t *slvTernary, then int) {
		if t.isElse(f.nextEffective(then+1, len(f.toks))) {
			return
		}
		start := t.start(then)
		elsePtr := t.elseOf(then)
		if start < 0 || elsePtr < 0 {
			return
		}
		elseEnd := t.end(then, elsePtr)
		thenContent := strings.Trim(f.content(then+1, elsePtr-1), phpTrimChars)
		elseContent := strings.Trim(f.content(elsePtr+1, elseEnd), phpTrimChars)
		condition := f.content(start, f.prevEffective(then-1))
		negated := f.is(start, "!")
		if negated {
			if elseContent != strings.TrimLeft(condition, "!") {
				return
			}
		} else if thenContent != condition {
			return
		}
		f.beginChangeset()
		if negated {
			f.replace(start, "")
			f.change(then, elseEnd, "?: "+thenContent)
		} else {
			f.removeBetweenIncluding(then+1, elsePtr-1)
		}
		f.endChangeset()
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/UselessTernaryOperatorSniff.php
//
// SlevomatUselessTernaryOperator drops "? true : false" after a boolean
// condition, negating it for "? false : true".
type SlevomatUselessTernaryOperator struct {
	assumeBoolean bool
}

func (SlevomatUselessTernaryOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\UselessTernaryOperatorSniff`
}

func (SlevomatUselessTernaryOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessTernaryOperator{}.Name())
}

func (r SlevomatUselessTernaryOperator) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["assumeAllConditionExpressionsAreAlreadyBoolean"].(bool); ok {
		r.assumeBoolean = v
	}
	return r
}

func (r SlevomatUselessTernaryOperator) Fix(s *tokens.Stream) bool {
	return slvRunTernaries(s, func(f *slvFile, t *slvTernary, then int) {
		n := len(f.toks)
		afterThen := f.nextEffective(then+1, n)
		elsePtr := afterThen
		if !t.isElse(afterThen) {
			if !f.slvIsBoolLiteral(afterThen) {
				return
			}
			elsePtr = f.nextEffective(afterThen+1, n)
			if !t.isElse(elsePtr) {
				return
			}
		}
		if !f.slvIsBoolLiteral(f.nextEffective(elsePtr+1, n)) {
			return
		}
		start := t.start(then)
		condEnd := f.prevEffective(then - 1)
		if !r.assumeBoolean && !f.slvCondReturnsBoolean(start, condEnd) {
			return
		}
		elseEnd := t.end(then, elsePtr)
		after := f.nextEffective(elseEnd+1, n)
		f.beginChangeset()
		if f.slvIsBoolLiteral(afterThen) && strings.EqualFold(f.toks[afterThen].orig, "false") {
			f.change(start, condEnd, f.slvCondNegative(start, condEnd, false))
		}
		if after >= 0 {
			f.removeBetweenIncluding(condEnd+1, after-1)
		}
		f.endChangeset()
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireTernaryOperatorSniff.php
//
// SlevomatRequireTernaryOperator turns an if/else that only returns or only
// assigns one variable into a ternary.
type SlevomatRequireTernaryOperator struct {
	ignoreMultiLine bool
}

func (SlevomatRequireTernaryOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireTernaryOperatorSniff`
}

func (SlevomatRequireTernaryOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireTernaryOperator{}.Name())
}

func (r SlevomatRequireTernaryOperator) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["ignoreMultiLine"].(bool); ok {
		r.ignoreMultiLine = v
	}
	return r
}

func (r SlevomatRequireTernaryOperator) Fix(s *tokens.Stream) bool {
	hasElse := false
	for i := range s.Len() {
		if t := s.At(i); t.Kind == token.Keyword && strings.EqualFold(t.Value, "else") {
			hasElse = true
			break
		}
	}
	if !hasElse {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		t := newSlvTernary(f)
		f.eachToken(func(i int) bool { return f.isKeyword(i, "if") }, func(ifPtr int) int {
			r.process(f, t, ifPtr)
			return 0
		})
	})
}

func (r SlevomatRequireTernaryOperator) process(f *slvFile, t *slvTernary, ifPtr int) {
	sc, n := t.sc, len(f.toks)
	if sc.closer[ifPtr] < 0 {
		return
	}
	elsePtr := f.nextEffective(sc.closer[ifPtr]+1, n)
	if !f.isKeyword(elsePtr, "else") || sc.closer[elsePtr] < 0 {
		return
	}
	if !r.compatibleScope(f, t, sc.opener[ifPtr], sc.closer[ifPtr]) || !r.compatibleScope(f, t, sc.opener[elsePtr], sc.closer[elsePtr]) {
		return
	}
	open := f.nextEffective(ifPtr+1, n)
	if !f.is(open, "(") || f.toks[open].closer < 0 {
		return
	}
	firstInIf := f.nextEffective(sc.opener[ifPtr]+1, n)
	firstInElse := f.nextEffective(sc.opener[elsePtr]+1, n)
	if f.isKeyword(firstInIf, "return") && f.isKeyword(firstInElse, "return") {
		r.withReturns(f, sc, ifPtr, elsePtr, open, firstInIf, firstInElse)
		return
	}
	r.withAssignments(f, sc, ifPtr, elsePtr, open, firstInIf, firstInElse)
}

func (SlevomatRequireTernaryOperator) blocked(f *slvFile, sc *phpcsScope, ifPtr, elsePtr, open int) bool {
	for _, owner := range []int{ifPtr, elsePtr} {
		for j := sc.opener[owner] + 1; j < sc.closer[owner]; j++ {
			if k := f.toks[j].kind; k == token.Comment || k == token.DocComment {
				return true
			}
		}
	}
	for j := open + 1; j < f.toks[open].closer; j++ {
		if f.isKeyword(j, "and") || f.isKeyword(j, "or") || f.isKeyword(j, "xor") {
			return true
		}
	}
	return false
}

func (f *slvFile) slvNextSemicolon(from int) int {
	for j := from; j < len(f.toks); j++ {
		if f.is(j, ";") {
			return j
		}
	}
	return -1
}

func (r SlevomatRequireTernaryOperator) withReturns(f *slvFile, sc *phpcsScope, ifPtr, elsePtr, open, returnInIf, returnInElse int) {
	if r.blocked(f, sc, ifPtr, elsePtr, open) {
		return
	}
	n := len(f.toks)
	closeParen := f.toks[open].closer
	afterReturnInIf := f.nextEffective(returnInIf+1, n)
	semicolonInIf := f.slvNextSemicolon(afterReturnInIf + 1)
	afterReturnInElse := f.nextEffective(returnInElse+1, n)
	semicolonInElse := f.slvNextSemicolon(afterReturnInElse + 1)
	if semicolonInIf < 0 || semicolonInElse < 0 {
		return
	}
	f.beginChangeset()
	f.replace(ifPtr, "return")
	if ifPtr+1 == open {
		f.add(ifPtr, " ")
	}
	f.replace(open, "")
	f.replace(closeParen, " ? ")
	f.removeBetweenIncluding(closeParen+1, afterReturnInIf-1)
	f.change(semicolonInIf, afterReturnInElse-1, " : ")
	f.removeBetweenIncluding(semicolonInElse+1, sc.closer[elsePtr])
	f.endChangeset()
}

func (r SlevomatRequireTernaryOperator) withAssignments(f *slvFile, sc *phpcsScope, ifPtr, elsePtr, open, firstInIf, firstInElse int) {
	n := len(f.toks)
	endInIf := f.identEnd(firstInIf)
	endInElse := f.identEnd(firstInElse)
	if endInIf < 0 || endInElse < 0 {
		return
	}
	identInIf := f.content(firstInIf, endInIf)
	if identInIf != f.content(firstInElse, endInElse) {
		return
	}
	assignInIf := f.nextEffective(endInIf+1, n)
	assignInElse := f.nextEffective(endInElse+1, n)
	if !f.is(assignInIf, "=") || !f.is(assignInElse, "=") {
		return
	}
	afterAssignInIf := f.nextEffective(assignInIf+1, n)
	afterAssignInElse := f.nextEffective(assignInElse+1, n)
	if f.is(afterAssignInIf, "&") || f.is(afterAssignInElse, "&") {
		return
	}
	if r.blocked(f, sc, ifPtr, elsePtr, open) {
		return
	}
	semicolonInIf := f.slvNextSemicolon(afterAssignInIf + 1)
	semicolonInElse := f.slvNextSemicolon(afterAssignInElse + 1)
	if semicolonInIf < 0 || semicolonInElse < 0 {
		return
	}
	f.beginChangeset()
	f.change(ifPtr, open, identInIf+" = ")
	f.change(f.toks[open].closer, afterAssignInIf-1, " ? ")
	f.change(semicolonInIf, afterAssignInElse-1, " : ")
	f.removeBetweenIncluding(semicolonInElse+1, sc.closer[elsePtr])
	f.endChangeset()
}

// compatibleScope is RequireTernaryOperatorSniff::isCompatibleScope.
func (r SlevomatRequireTernaryOperator) compatibleScope(f *slvFile, t *slvTernary, opener, closer int) bool {
	semicolon := -1
	for j := opener + 1; j < closer; j++ {
		if f.is(j, ";") {
			semicolon = j
			break
		}
	}
	if semicolon < 0 {
		return false
	}
	for j := opener + 1; j < semicolon; j++ {
		if t.isThen(j) {
			return false
		}
	}
	if r.ignoreMultiLine {
		first := f.nextEffective(opener+1, len(f.toks))
		for j := first + 1; j < semicolon; j++ {
			if f.toks[j].kind == token.Whitespace && f.toks[j].content == f.eol {
				return false
			}
		}
	}
	return f.nextEffective(semicolon+1, len(f.toks)) == closer
}
