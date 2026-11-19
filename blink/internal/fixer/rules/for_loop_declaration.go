package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/ForLoopDeclarationSniff.php
//
// ForLoopDeclaration normalizes the spacing of a for header: no space inside
// the parentheses or before a semicolon, one after it:
// "for ( $i = 0 ;$i < 3 ; $i++ )" -> "for ($i = 0; $i < 3; $i++)".
type ForLoopDeclaration struct {
	RequiredSpacesAfterOpen   int
	RequiredSpacesBeforeClose int
	IgnoreNewlines            bool
}

func (ForLoopDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\ControlStructures\ForLoopDeclarationSniff`
}

func (ForLoopDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/ForLoopDeclarationSniff.php"
}

func (r ForLoopDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	r.RequiredSpacesAfterOpen = configInt(config, "requiredSpacesAfterOpen", r.RequiredSpacesAfterOpen)
	r.RequiredSpacesBeforeClose = configInt(config, "requiredSpacesBeforeClose", r.RequiredSpacesBeforeClose)
	if v, ok := config["ignoreNewlines"].(bool); ok {
		r.IgnoreNewlines = v
	}
	return r
}

func (r ForLoopDeclaration) Fix(s *tokens.Stream) bool {
	if !s.Index().HasAnyWord("for") {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.isKeyword(i, "for") {
				r.process(f, i)
			}
		}
	})
}

func (r ForLoopDeclaration) process(f *slvFile, i int) {
	open := slvFindNext(f, i, len(f.toks), func(j int) bool { return f.is(j, "(") })
	if open < 0 || f.toks[open].closer < 0 {
		return
	}
	closeIdx := f.toks[open].closer
	isWs := func(j int) bool { return f.toks[j].kind == token.Whitespace }
	// findNext/findPrevious(T_WHITESPACE, ..., true) give false, read as token 0
	nonWsAfter := func(from, end int) int {
		return max(slvFindNext(f, from, end, func(j int) bool { return !isWs(j) }), 0)
	}
	nonWsBefore := func(from, end int) int { // end is inclusive, as in findPrevious
		for j := from; j >= end; j-- {
			if !isWs(j) {
				return j
			}
		}
		return 0
	}
	line := func(j int) int { return f.toks[j].line }
	clear := func(from, to int) { // [from, to)
		for j := from; j < to; j++ {
			f.replaceToken(j, "")
		}
	}

	if r.RequiredSpacesAfterOpen == 0 && isWs(open+1) {
		next := nonWsAfter(open+1, closeIdx)
		if !r.IgnoreNewlines || line(next) == line(open) {
			f.beginChangeset()
			for j := open + 1; j < closeIdx && isWs(j); j++ {
				f.replaceToken(j, "")
			}
			f.endChangeset()
		}
	} else if r.RequiredSpacesAfterOpen > 0 {
		next := nonWsAfter(open+1, closeIdx)
		found, newline := 0, line(open) != line(next)
		if !newline && isWs(open+1) {
			found = f.toks[open+1].length
		}
		if (newline || found != r.RequiredSpacesAfterOpen) && (!newline || !r.IgnoreNewlines) {
			pad := strings.Repeat(" ", r.RequiredSpacesAfterOpen)
			if !newline && found == 0 {
				f.add(open, pad)
			} else {
				f.beginChangeset()
				f.replaceToken(open+1, pad)
				clear(open+2, next)
				f.endChangeset()
			}
		}
	}

	prev := nonWsBefore(closeIdx-1, open)
	fixable := line(prev) == line(closeIdx) || !f.isEmpty(prev)
	if r.RequiredSpacesBeforeClose == 0 && isWs(closeIdx-1) && (!r.IgnoreNewlines || line(prev) == line(closeIdx)) {
		if fixable {
			f.beginChangeset()
			for j := closeIdx - 1; j > open && isWs(j); j-- {
				f.replaceToken(j, "")
			}
			f.endChangeset()
		}
	} else if r.RequiredSpacesBeforeClose > 0 {
		found, newline := 0, line(closeIdx) != line(prev)
		if !newline && isWs(closeIdx-1) {
			found = f.toks[closeIdx-1].length
		}
		if (newline || found != r.RequiredSpacesBeforeClose) && (!newline || !r.IgnoreNewlines) && fixable {
			pad := strings.Repeat(" ", r.RequiredSpacesBeforeClose)
			if !newline && found == 0 {
				f.addBefore(closeIdx, pad)
			} else {
				f.beginChangeset()
				f.replaceToken(closeIdx-1, pad)
				clear(prev+1, closeIdx-1)
				f.endChangeset()
			}
		}
	}

	count := 0
	for semi := open; count < 2; {
		semi = slvFindNext(f, semi+1, closeIdx, func(j int) bool { return f.is(j, ";") })
		if semi < 0 {
			return
		}
		if f.enclosingScope(semi) > open {
			continue
		}
		count++
		if p := nonWsBefore(semi-1, open); (count != 1 || p != open) && isWs(semi-1) {
			f.beginChangeset()
			clear(p+1, semi)
			f.endChangeset()
		}
		next := nonWsAfter(semi+1, closeIdx+1)
		if count == 2 && next == closeIdx {
			continue
		}
		if !isWs(semi+1) && !f.is(semi+1, ";") {
			f.add(semi, " ")
		} else if isWs(semi+1) && !f.is(next, ";") {
			newline := line(semi) != line(next)
			if (newline || f.toks[semi+1].length != 1) && (!newline || !r.IgnoreNewlines) {
				f.beginChangeset()
				f.replaceToken(semi+1, " ")
				clear(semi+2, next)
				f.endChangeset()
			}
		}
	}
}
