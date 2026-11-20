package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/DisallowContinueWithoutIntegerOperandInSwitchSniff.php
//
// SlevomatDisallowContinueWithoutIntegerOperandInSwitch turns a bare "continue"
// directly in a switch into "break".
type SlevomatDisallowContinueWithoutIntegerOperandInSwitch struct{}

func (SlevomatDisallowContinueWithoutIntegerOperandInSwitch) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\DisallowContinueWithoutIntegerOperandInSwitchSniff`
}

func (SlevomatDisallowContinueWithoutIntegerOperandInSwitch) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowContinueWithoutIntegerOperandInSwitch{}.Name())
}

func (SlevomatDisallowContinueWithoutIntegerOperandInSwitch) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		var conds [][]phpcsCond
		var sc *phpcsScope
		f.eachToken(func(i int) bool { return f.isKeyword(i, "continue") }, func(i int) int {
			if operand := f.nextEffective(i+1, len(f.toks)); operand >= 0 && slvccIsLNumber(f.toks[operand]) {
				return 0
			}
			if sc == nil {
				sc = newPhpcsScope(f)
				conds, _ = sc.levelMap()
			}
			c := conds[i]
			if len(c) == 0 || c[len(c)-1].caseIndex || sc.code[c[len(c)-1].key] != "switch" {
				return 0
			}
			f.beginChangeset()
			f.replace(i, "break")
			f.endChangeset()
			return 0
		})
	})
}

// slvccIsLNumber tells PHP's T_LNUMBER (an integer literal) from T_DNUMBER.
func slvccIsLNumber(t slvTok) bool {
	if t.kind != token.Number {
		return false
	}
	v := strings.ToLower(t.orig)
	if strings.HasPrefix(v, "0x") {
		return true
	}
	return !strings.ContainsAny(v, ".e")
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/LanguageConstructWithParenthesesSniff.php
//
// SlevomatLanguageConstructWithParentheses drops the parentheses around the
// operand of echo, print, return, include, ...: "echo('a');" -> "echo 'a';".
type SlevomatLanguageConstructWithParentheses struct{}

func (SlevomatLanguageConstructWithParentheses) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\LanguageConstructWithParenthesesSniff`
}

func (SlevomatLanguageConstructWithParentheses) SourceURL() string {
	return fixer.SourceURLFor(SlevomatLanguageConstructWithParentheses{}.Name())
}

var slvccLanguageConstructs = map[string]bool{
	"break": true, "continue": true, "echo": true, "include": true, "include_once": true,
	"print": true, "require": true, "require_once": true, "return": true, "throw": true,
	"yield": true, "yield from": true,
}

func (SlevomatLanguageConstructWithParentheses) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		n := len(f.toks)
		var sc *phpcsScope
		isExit := func(i int) bool {
			if f.toks[i].kind != token.Ident {
				return false
			}
			if sc == nil {
				sc = newPhpcsScope(f)
			}
			return sc.code[i] == "exit"
		}
		f.eachToken(func(i int) bool {
			t := f.toks[i]
			return t.kind == token.Keyword && slvccLanguageConstructs[strings.Join(strings.Fields(strings.ToLower(t.orig)), " ")] || isExit(i)
		}, func(i int) int {
			open := f.nextEffective(i+1, n)
			if !f.is(open, "(") || f.toks[open].closer < 0 {
				return 0
			}
			closeIdx := f.toks[open].closer
			after := f.nextEffective(closeIdx+1, n)
			closesStatement := f.is(after, ";") || f.is(after, ")") || f.is(after, "]") && f.isShortArrayOpen(f.toks[after].closer)
			if !closesStatement {
				return 0
			}
			hasContent := f.nextEffective(open+1, closeIdx) >= 0
			if hasContent && f.toks[i].kind == token.Ident {
				return 0
			}
			f.beginChangeset()
			f.replace(open, "")
			if f.toks[open-1].kind != token.Whitespace && hasContent {
				f.add(open, " ")
			}
			f.replace(closeIdx, "")
			f.endChangeset()
			return 0
		})
	})
}

// slvccNewStop finds, from i on, the first token NewWithParentheses and
// NewWithoutParentheses look for after the class name of a "new" at newIdx,
// stepping over the "[...]" of an array access in the name; -1 when none.
func slvccNewStop(f *slvFile, sc *phpcsScope, newIdx, i int) int {
	for ; i < len(f.toks); i++ {
		t := f.toks[i]
		switch {
		case f.is(i, "]"):
			if f.isShortArrayOpen(t.closer) || t.closer < 0 || t.closer <= newIdx {
				return i
			}
		case f.is(i, "?"):
			if sc.isInlineThen(i) {
				return i
			}
		case f.is(i, ":"):
			if sc.code[i] == "inline_else" {
				return i
			}
		case f.is(i, "=>"):
			if f.isDoubleArrow(i) {
				return i
			}
		case f.is(i, "(") || f.is(i, ";") || f.is(i, ",") || f.is(i, "??") || f.is(i, ")"):
			return i
		}
	}
	return -1
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/NewWithParenthesesSniff.php
//
// SlevomatNewWithParentheses adds the missing "()" to "new Foo".
type SlevomatNewWithParentheses struct{}

func (SlevomatNewWithParentheses) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\NewWithParenthesesSniff`
}

func (SlevomatNewWithParentheses) SourceURL() string {
	return fixer.SourceURLFor(SlevomatNewWithParentheses{}.Name())
}

func (SlevomatNewWithParentheses) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		n := len(f.toks)
		var sc *phpcsScope
		f.eachToken(func(i int) bool { return f.isKeyword(i, "new") }, func(newIdx int) int {
			next := f.nextEffective(newIdx+1, n)
			if next < 0 {
				return 0
			}
			if f.toks[next].kind == token.AttributeStart {
				if next = f.attributeTarget(next); next < 0 {
					return 0
				}
			}
			if f.isKeyword(next, "class") || f.isKeyword(next, "readonly") {
				return 0
			}
			if f.is(next, "(") && f.toks[next].closer >= 0 {
				next = f.toks[next].closer
			}
			if sc == nil {
				sc = newPhpcsScope(f)
			}
			stop := slvccNewStop(f, sc, newIdx, next+1)
			if f.is(stop, "(") {
				return 0
			}
			if stop < 0 {
				return 0
			}
			end := f.prevEffective(stop - 1)
			if end < 0 {
				return 0
			}
			f.beginChangeset()
			f.add(end, "()")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/NewWithoutParenthesesSniff.php
//
// SlevomatNewWithoutParentheses removes the empty "()" of "new Foo()".
type SlevomatNewWithoutParentheses struct{}

func (SlevomatNewWithoutParentheses) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\NewWithoutParenthesesSniff`
}

func (SlevomatNewWithoutParentheses) SourceURL() string {
	return fixer.SourceURLFor(SlevomatNewWithoutParentheses{}.Name())
}

func (SlevomatNewWithoutParentheses) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		n := len(f.toks)
		var sc *phpcsScope
		f.eachToken(func(i int) bool { return f.isKeyword(i, "new") }, func(newIdx int) int {
			next := f.nextEffective(newIdx+1, n)
			if next < 0 || f.isKeyword(next, "class") {
				return 0
			}
			if sc == nil {
				sc = newPhpcsScope(f)
			}
			open := slvccNewStop(f, sc, newIdx, next+1)
			if !f.is(open, "(") || f.toks[open].closer < 0 {
				return 0
			}
			closeIdx := f.toks[open].closer
			if f.nextNonWhitespace(open+1, n) != closeIdx {
				return 0
			}
			if after := f.nextEffective(closeIdx+1, n); after >= 0 &&
				(f.is(after, "->") || f.is(after, "?->") || f.is(after, "::") || f.is(after, "[") || f.is(after, "(")) {
				return 0
			}
			f.beginChangeset()
			for j := open; j <= closeIdx; j++ {
				f.replaceToken(j, "")
			}
			f.endChangeset()
			return 0
		})
	})
}
