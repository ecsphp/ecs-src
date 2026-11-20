package rules

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/EarlyExitSniff.php
//
// SlevomatEarlyExit inverts a trailing "if" into an early return or continue,
// and drops an "else" or "elseif" after a branch that already exits.
type SlevomatEarlyExit struct {
	ignoreStandaloneIfInScope          bool
	ignoreOneLineTrailingIf            bool
	ignoreTrailingIfWithOneInstruction bool
}

func (SlevomatEarlyExit) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\EarlyExitSniff`
}

func (SlevomatEarlyExit) SourceURL() string {
	return fixer.SourceURLFor(SlevomatEarlyExit{}.Name())
}

func (r SlevomatEarlyExit) WithConfig(config map[string]any) fixer.Fixer {
	for key, field := range map[string]*bool{
		"ignoreStandaloneIfInScope":          &r.ignoreStandaloneIfInScope,
		"ignoreOneLineTrailingIf":            &r.ignoreOneLineTrailingIf,
		"ignoreTrailingIfWithOneInstruction": &r.ignoreTrailingIfWithOneInstruction,
	} {
		if v, ok := config[key].(bool); ok {
			*field = v
		}
	}
	return r
}

func (r SlevomatEarlyExit) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		t := newSlvTernary(f)
		ee := slvEarlyExit{r: r, t: t, sc: t.sc}
		f.eachToken(func(i int) bool {
			c := ee.sc.code[i]
			return c == "if" || c == "elseif" || c == "else"
		}, func(i int) int {
			switch ee.sc.code[i] {
			case "if":
				ee.processIf(i)
			case "elseif":
				ee.processElseIf(i)
			default:
				ee.processElse(i)
			}
			return 0
		})
	})
}

type slvEarlyExit struct {
	r  SlevomatEarlyExit
	t  *slvTernary
	sc *phpcsScope
}

var errSlvEarlyExitUnsupported = errors.New("control structure without curly braces is not supported")

// isNamedFunction tells a T_FUNCTION from a closure's "function".
func (ee slvEarlyExit) isNamedFunction(i int) bool {
	return ee.sc.f.isKeyword(i, "function") && !ee.sc.f.isClosure(i)
}

func (ee slvEarlyExit) hasNamedFunction(cond int) bool {
	sc := ee.sc
	return sc.f.slvCondFindNext(sc.opener[cond]+1, sc.closer[cond], ee.isNamedFunction) >= 0
}

func (ee slvEarlyExit) parenthesis(cond int) (int, int) {
	f := ee.sc.f
	open := f.nextEffective(cond+1, len(f.toks))
	if !f.is(open, "(") {
		return -1, -1
	}
	return open, f.toks[open].closer
}

func (ee slvEarlyExit) processElse(elsePtr int) {
	sc := ee.sc
	f := sc.f
	if sc.opener[elsePtr] < 0 {
		return
	}
	all, err := ee.allConditions(elsePtr)
	if err != nil || ee.hasNamedFunction(elsePtr) {
		return
	}
	ifPtr := all[0]
	ifExit, elseExit, prevCond, prevExit := -1, -1, -1, -1
	for _, cond := range all {
		exit, err := ee.findEarlyExit(sc.opener[cond], sc.closer[cond])
		if err != nil {
			return
		}
		if cond == elsePtr {
			elseExit = exit
			continue
		}
		if len(all) > 2 && exit < 0 {
			return
		}
		prevCond, prevExit = cond, exit
		if cond == ifPtr {
			ifExit = exit
		}
	}
	if ifExit < 0 && elseExit < 0 {
		return
	}
	ifOpen, ifClose := ee.parenthesis(ifPtr)
	if elseExit >= 0 && prevExit < 0 {
		elseCode := f.content(sc.opener[elsePtr]+1, sc.closer[elsePtr]-1)
		negative := f.slvCondNegative(ifOpen, ifClose, false)
		afterIf := sc.slvRemoveIndentation(sc.opener[ifPtr]+1, sc.closer[ifPtr]-1, f.indentation(ifPtr))
		f.beginChangeset()
		f.change(ifPtr, sc.closer[elsePtr], fmt.Sprintf("if %s {%s}%s%s", negative, elseCode, f.eol, afterIf))
		f.endChangeset()
		return
	}
	afterIf := sc.slvRemoveIndentation(sc.opener[elsePtr]+1, sc.closer[elsePtr]-1, f.indentation(ifPtr))
	f.beginChangeset()
	f.change(sc.closer[prevCond]+1, sc.closer[elsePtr], f.eol+afterIf)
	f.endChangeset()
}

func (ee slvEarlyExit) processElseIf(elseIf int) {
	sc := ee.sc
	f := sc.f
	all, err := ee.allConditions(elseIf)
	if err != nil || ee.hasNamedFunction(elseIf) {
		return
	}
	for _, cond := range all {
		exit, err := ee.findEarlyExit(sc.opener[cond], sc.closer[cond])
		if err != nil {
			return
		}
		if cond == elseIf {
			break
		}
		if exit < 0 {
			return
		}
	}
	before := f.prevNonWhitespace(elseIf - 1)
	f.beginChangeset()
	for k := before + 1; k < elseIf; k++ {
		f.replaceToken(k, "")
	}
	f.add(before, f.eol)
	f.add(before, f.eol)
	f.replace(elseIf, f.indentation(all[0])+"if")
	f.endChangeset()
}

func (ee slvEarlyExit) processIf(ifPtr int) {
	sc := ee.sc
	f := sc.f
	opener, closer := sc.opener[ifPtr], sc.closer[ifPtr]
	if closer < 0 {
		return
	}
	next := f.nextNonWhitespace(closer+1, len(f.toks))
	if next < 0 || !f.is(next, "}") {
		return
	}
	prev := f.prevEffective(ifPtr - 1)
	if ee.r.ignoreStandaloneIfInScope && (f.is(prev, "{") || sc.isTColon(prev)) {
		return
	}
	if ee.r.ignoreOneLineTrailingIf && f.toks[opener].line+2 == f.toks[closer].line {
		return
	}
	if ee.r.ignoreTrailingIfWithOneInstruction {
		if last := f.prevEffective(closer - 1); f.is(last, ";") {
			ignore := true
			for search := opener + 1; ; {
				another := f.slvCondFindNext(search, last, func(i int) bool { return f.is(i, ";") })
				if another < 0 {
					break
				}
				if ee.t.sameScope(another, last) {
					ignore = false
					break
				}
				search = another + 1
			}
			if ignore {
				return
			}
		}
	}
	scope := sc.cond[next]
	if scope < 0 {
		return
	}
	exitCode := "return"
	switch sc.code[scope] {
	case "function":
	case "while", "do", "foreach", "for":
		exitCode = "continue"
	default:
		return
	}
	if exit, err := ee.findEarlyExit(opener, closer); err != nil || exit >= 0 {
		return
	}
	ifOpen, ifClose := ee.parenthesis(ifPtr)
	indentation := f.indentation(ifPtr)
	negative := f.slvCondNegative(ifOpen, ifClose, false)
	afterIf := sc.slvRemoveIndentation(opener+1, closer-1, indentation)
	content := fmt.Sprintf("if %s {%s%s    %s;%s%s}%s%s", negative, f.eol, indentation, exitCode, f.eol, indentation, f.eol, afterIf)
	f.beginChangeset()
	f.change(ifPtr, closer, content)
	f.endChangeset()
}

// findEarlyExit is findEarlyExitInScope: the exit statement ending the scope,
// or -1; an error stands for the exception upstream throws on an "if" without
// curly braces.
func (ee slvEarlyExit) findEarlyExit(start, end int) (int, error) {
	sc := ee.sc
	f := sc.f
	for p := start + 1; p < end; p++ {
		if sc.code[p] != "if" || sc.level(p)-1 != sc.level(start) {
			continue
		}
		conds, err := ee.allConditions(p)
		if err != nil {
			return -1, err
		}
		for _, cond := range conds {
			exit, err := ee.findEarlyExit(sc.opener[cond], sc.closer[cond])
			if err != nil {
				return -1, err
			}
			if exit < 0 {
				return -1, nil
			}
		}
	}
	last := -1
	for k := end - 1; k >= start; k-- {
		if !f.isEmpty(k) {
			last = k
			break
		}
	}
	if !f.is(last, ";") {
		return -1, nil
	}
	return sc.prevLocal(last-1, start, func(i int) bool {
		switch sc.code[i] {
		case "return", "continue", "break", "throw", "exit":
			return true
		}
		return false
	}), nil
}

// allConditions is getAllConditionsPointers: the if, elseif and else of one
// chain, in source order.
func (ee slvEarlyExit) allConditions(cond int) ([]int, error) {
	sc := ee.sc
	f := sc.f
	all := []int{cond}
	if sc.opener[cond] >= 0 && f.is(sc.opener[cond], ":") {
		return nil, errSlvEarlyExitUnsupported
	}
	if sc.code[cond] != "if" {
		for current := cond; sc.code[current] != "if"; {
			closeBrace := f.prevEffective(current - 1)
			if closeBrace < 0 || sc.cond[closeBrace] < 0 {
				return nil, errSlvEarlyExitUnsupported
			}
			current = sc.cond[closeBrace]
			all = append(all, current)
		}
	}
	if sc.code[cond] != "else" {
		if sc.closer[cond] < 0 {
			return nil, errSlvEarlyExitUnsupported
		}
		for current := f.nextEffective(sc.closer[cond]+1, len(f.toks)); current >= 0 && (sc.code[current] == "elseif" || sc.code[current] == "else"); {
			all = append(all, current)
			if sc.closer[current] < 0 {
				return nil, errSlvEarlyExitUnsupported
			}
			current = f.nextEffective(sc.closer[current]+1, len(f.toks))
		}
	}
	slices.Sort(all)
	return all, nil
}
