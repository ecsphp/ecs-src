package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/ForEachLoopDeclarationSniff.php
//
// ForEachLoopDeclaration normalizes the spacing of a foreach header and
// lowercases "as": "foreach ( $a AS $k=>$v )" -> "foreach ($a as $k => $v)".
type ForEachLoopDeclaration struct {
	RequiredSpacesAfterOpen   int
	RequiredSpacesBeforeClose int
}

func (ForEachLoopDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\ControlStructures\ForEachLoopDeclarationSniff`
}

func (ForEachLoopDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/ControlStructures/ForEachLoopDeclarationSniff.php"
}

func (r ForEachLoopDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	r.RequiredSpacesAfterOpen = configInt(config, "requiredSpacesAfterOpen", r.RequiredSpacesAfterOpen)
	r.RequiredSpacesBeforeClose = configInt(config, "requiredSpacesBeforeClose", r.RequiredSpacesBeforeClose)
	return r
}

func (r ForEachLoopDeclaration) Fix(s *tokens.Stream) bool {
	if !s.Index().HasAnyWord("foreach") {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.isKeyword(i, "foreach") {
				r.process(f, i)
			}
		}
	})
}

func (r ForEachLoopDeclaration) process(f *slvFile, i int) {
	open := slvFindNext(f, i, len(f.toks), func(j int) bool { return f.is(j, "(") })
	if open < 0 || f.toks[open].closer < 0 {
		return
	}
	closeIdx := f.toks[open].closer
	slvRequiredSpaces(f, open+1, r.RequiredSpacesAfterOpen, func(pad string) { f.add(open, pad) })
	slvRequiredSpaces(f, closeIdx-1, r.RequiredSpacesBeforeClose, func(pad string) { f.addBefore(closeIdx, pad) })

	as := slvFindNext(f, open, len(f.toks), func(j int) bool { return f.isKeyword(j, "as") })
	if as < 0 {
		return
	}
	if v := f.toks[as].orig; v != strings.ToLower(v) {
		f.replaceToken(as, strings.ToLower(v))
	}
	if arrow := slvFindNext(f, as, closeIdx, func(j int) bool { return f.is(j, "=>") }); arrow >= 0 {
		slvOneSpace(f, arrow-1, func() { f.addBefore(arrow, " ") })
		slvOneSpace(f, arrow+1, func() { f.add(arrow, " ") })
	}
	slvOneSpace(f, as-1, func() { f.addBefore(as, " ") })
	slvOneSpace(f, as+1, func() { f.add(as, " ") })
}

// slvFindNext is File::findNext for the first token in [from, end) match accepts, or -1.
func slvFindNext(f *slvFile, from, end int, match func(int) bool) int {
	for j := from; j < end && j < len(f.toks); j++ {
		if match(j) {
			return j
		}
	}
	return -1
}

// slvRequiredSpaces wants exactly n spaces in the whitespace token at w (just
// inside a bracket); pad adds them when there is none.
func slvRequiredSpaces(f *slvFile, w, n int, pad func(string)) {
	isWs := f.toks[w].kind == token.Whitespace
	if n == 0 {
		if isWs {
			f.replaceToken(w, "")
		}
		return
	}
	found := 0
	if isWs {
		found = f.toks[w].length
	}
	if found == n {
		return
	}
	if found == 0 {
		pad(strings.Repeat(" ", n))
		return
	}
	f.replaceToken(w, strings.Repeat(" ", n))
}

// slvOneSpace wants a single space in the whitespace token at w; add inserts
// one when there is none.
func slvOneSpace(f *slvFile, w int, add func()) {
	if f.toks[w].kind != token.Whitespace {
		add()
		return
	}
	if f.toks[w].length != 1 {
		f.replaceToken(w, " ")
	}
}
