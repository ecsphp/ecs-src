package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireNullCoalesceOperatorSniff.php
//
// SlevomatRequireNullCoalesceOperator turns "isset($a) ? $a : $b" and
// "$a !== null ? $a : $b" into "$a ?? $b".
type SlevomatRequireNullCoalesceOperator struct{}

func (SlevomatRequireNullCoalesceOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireNullCoalesceOperatorSniff`
}

func (SlevomatRequireNullCoalesceOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireNullCoalesceOperator{}.Name())
}

func (SlevomatRequireNullCoalesceOperator) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		t := newSlvTernary(f)
		f.eachToken(func(i int) bool {
			return f.isKeyword(i, "isset") || f.is(i, "===") || f.is(i, "!==")
		}, func(i int) int {
			if f.isKeyword(i, "isset") {
				slvncCheckIsset(t, i)
			} else {
				slvncCheckIdentical(t, i)
			}
			return 0
		})
	})
}

func slvncCheckIsset(t *slvTernary, isset int) {
	f := t.f
	prev := f.prevEffective(isset - 1)
	if f.is(prev, "!") || f.isBooleanOperator(prev) {
		return
	}
	open := f.nextEffective(isset+1, len(f.toks))
	if !f.is(open, "(") || f.toks[open].closer < 0 {
		return
	}
	closeIdx := f.toks[open].closer
	then := f.nextEffective(closeIdx+1, len(f.toks))
	if !t.isThen(then) {
		return
	}
	if f.slvCondFindNext(open+1, closeIdx, func(i int) bool { return f.is(i, ",") }) >= 0 {
		return
	}
	inlineElse := t.elseOf(then)
	if inlineElse < 0 {
		return
	}
	variable := f.identContent(open+1, closeIdx-1)
	if variable != f.identContent(then+1, inlineElse-1) {
		return
	}
	start := isset
	if f.parenCast(prev) != "" {
		start = prev
	}
	f.beginChangeset()
	f.change(start, inlineElse, variable+" ??")
	f.endChangeset()
}

func slvncCheckIdentical(t *slvTernary, identical int) {
	f, sc := t.f, t.sc
	before := f.prevEffective(identical - 1)
	after := f.nextEffective(identical+1, len(f.toks))
	if before < 0 || after < 0 || !sc.isNull(before) && !sc.isNull(after) {
		return
	}
	yoda := sc.isNull(before)
	varEnd := before
	if yoda {
		varEnd = after
	}
	tmp := varEnd
	for f.is(tmp, ")") && tmp >= 0 {
		tmp = f.prevEffective(f.toks[tmp].closer - 1)
	}
	varStart := f.identStart(tmp)
	if varStart < 0 {
		return
	}
	condStart := varStart
	if yoda {
		condStart = before
	}
	if f.isBooleanOperator(f.prevEffective(condStart - 1)) {
		return
	}
	thenFrom := after
	if yoda {
		thenFrom = varEnd
	}
	then := f.nextEffective(thenFrom+1, len(f.toks))
	if !t.isThen(then) {
		return
	}
	inlineElse := t.elseOf(then)
	if inlineElse < 0 {
		return
	}
	elseEnd := t.end(then, inlineElse)
	afterElseEnd := f.nextEffective(elseEnd+1, len(f.toks))

	isIdentical := f.is(identical, "===")
	compareFrom, compareTo := then, inlineElse
	if isIdentical {
		compareFrom, compareTo = inlineElse, afterElseEnd
	}
	if compareTo < 0 {
		return
	}
	compareStart := f.nextEffective(compareFrom+1, len(f.toks))
	compareEnd := f.prevEffective(compareTo - 1)
	if f.identContent(compareStart, compareEnd) != f.identContent(varStart, varEnd) {
		return
	}
	variable := strings.TrimSpace(f.content(varStart, varEnd))
	f.beginChangeset()
	f.replace(condStart, variable+" ??")
	if isIdentical {
		for k := condStart + 1; k <= then; k++ {
			f.replaceToken(k, "")
		}
		for k := f.prevEffective(inlineElse-1) + 1; k <= elseEnd; k++ {
			f.replaceToken(k, "")
		}
	} else {
		for k := condStart + 1; k <= inlineElse; k++ {
			f.replaceToken(k, "")
		}
	}
	f.endChangeset()
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireNullCoalesceEqualOperatorSniff.php
//
// SlevomatRequireNullCoalesceEqualOperator turns "$a = $a ?? $b" (and, with
// checkIfConditions, "if ($a === null) { $a = $b; }") into "$a ??= $b".
type SlevomatRequireNullCoalesceEqualOperator struct {
	disabled          bool
	checkIfConditions bool
}

func (SlevomatRequireNullCoalesceEqualOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireNullCoalesceEqualOperatorSniff`
}

func (SlevomatRequireNullCoalesceEqualOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireNullCoalesceEqualOperator{}.Name())
}

func (r SlevomatRequireNullCoalesceEqualOperator) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["enable"].(bool); ok {
		r.disabled = !v
	}
	if v, ok := config["checkIfConditions"].(bool); ok {
		r.checkIfConditions = v
	}
	return r
}

func (r SlevomatRequireNullCoalesceEqualOperator) Fix(s *tokens.Stream) bool {
	if r.disabled {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		t := newSlvTernary(f)
		f.eachToken(func(i int) bool { return f.is(i, "=") }, func(i int) int {
			slvncCheckCoalesce(t.f, i)
			if r.checkIfConditions {
				slvncCheckIf(t.sc, i)
			}
			return 0
		})
	})
}

// slvncIsOperator tells a Tokens::OPERATORS token.
func slvncIsOperator(f *slvFile, i int) bool {
	switch f.toks[i].orig {
	case "+", "-", "*", "/", "%", "**", "<=>", "??", "&", "|", "^", "<<", ">>":
		return f.is(i, f.toks[i].orig)
	}
	return false
}

func slvncCheckCoalesce(f *slvFile, equal int) {
	varStart := f.nextEffective(equal+1, len(f.toks))
	varEnd := f.identEnd(varStart)
	if varEnd < 0 {
		return
	}
	coalesce := f.nextEffective(varEnd+1, len(f.toks))
	if !f.is(coalesce, "??") {
		return
	}
	beforeEnd := f.prevEffective(equal - 1)
	beforeStart := f.identStart(beforeEnd)
	if beforeStart < 0 || f.identContent(beforeStart, beforeEnd) != f.identContent(varStart, varEnd) {
		return
	}
	semicolon := f.slvCondFindNext(equal+1, len(f.toks), func(i int) bool { return f.is(i, ";") })
	if semicolon < 0 {
		semicolon = len(f.toks)
	}
	if f.slvCondFindNext(coalesce+1, semicolon, func(i int) bool { return slvncIsOperator(f, i) }) >= 0 {
		return
	}
	f.beginChangeset()
	f.change(equal, coalesce, "??=")
	f.endChangeset()
}

func slvncCheckIf(sc *phpcsScope, equal int) {
	f := sc.f
	conds := sc.conditions(equal)
	if len(conds) == 0 {
		return
	}
	ifPtr := conds[len(conds)-1]
	if sc.code[ifPtr] != "if" {
		return
	}
	scopeCloser := sc.closer[ifPtr]
	if next := f.nextEffective(scopeCloser+1, len(f.toks)); next >= 0 && (sc.code[next] == "elseif" || sc.code[next] == "else") {
		return
	}
	open := f.nextEffective(ifPtr+1, len(f.toks))
	if !f.is(open, "(") || f.toks[open].closer < 0 {
		return
	}
	ifVarStart := f.nextEffective(open+1, len(f.toks))
	ifVarEnd := f.identEnd(ifVarStart)
	if ifVarEnd < 0 {
		return
	}
	next := f.nextEffective(ifVarEnd+1, len(f.toks))
	if !f.is(next, "===") {
		return
	}
	next = f.nextEffective(next+1, len(f.toks))
	if !sc.isNull(next) || f.nextEffective(next+1, len(f.toks)) != f.toks[open].closer {
		return
	}
	beforeStart := f.nextEffective(sc.opener[ifPtr]+1, len(f.toks))
	beforeEnd := f.identEnd(beforeStart)
	if beforeEnd < 0 || f.nextEffective(beforeEnd+1, len(f.toks)) != equal {
		return
	}
	variable := f.identContent(ifVarStart, ifVarEnd)
	if variable != f.identContent(beforeStart, beforeEnd) {
		return
	}
	semicolon := f.slvCondFindNext(equal+1, len(f.toks), func(i int) bool { return f.is(i, ";") })
	if semicolon < 0 || f.nextEffective(semicolon+1, len(f.toks)) != scopeCloser {
		return
	}
	codeStart := f.nextEffective(equal+1, len(f.toks))
	code := sc.slvRemoveIndentation(codeStart, semicolon, f.indentation(ifPtr))
	f.beginChangeset()
	f.change(ifPtr, scopeCloser, variable+" ??= "+strings.TrimSpace(code))
	f.endChangeset()
}
