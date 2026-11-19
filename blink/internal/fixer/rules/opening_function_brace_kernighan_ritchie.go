package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Functions/OpeningFunctionBraceKernighanRitchieSniff.php
//
// OpeningFunctionBraceKernighanRitchie puts a function's opening brace on its
// declaration line, one space after it. Closures are left alone
// (checkClosures defaults to false).
type OpeningFunctionBraceKernighanRitchie struct{}

func (OpeningFunctionBraceKernighanRitchie) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Functions\OpeningFunctionBraceKernighanRitchieSniff`
}

func (OpeningFunctionBraceKernighanRitchie) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Functions/OpeningFunctionBraceKernighanRitchieSniff.php"
}

func (OpeningFunctionBraceKernighanRitchie) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if _, open := phpcsNamedFunctionOpener(f, i); open >= 0 {
				openingFunctionBraceKernighanRitchie(f, open)
			}
		}
	})
}

func openingFunctionBraceKernighanRitchie(f *slvFile, open int) {
	prev := f.prevEffective(open - 1)
	braceLine := f.toks[open].line
	lineDifference := braceLine - f.toks[prev].line
	if lineDifference > 0 {
		f.beginChangeset()
		f.replaceToken(prev, f.tokenContent(prev)+" {")
		f.replaceToken(open, "")
		if open+2 < len(f.toks) && f.toks[open+1].kind == token.Whitespace && f.toks[open+2].line > braceLine {
			f.replaceToken(open+1, "")
			if f.toks[open-1].kind == token.Whitespace && f.toks[open-1].line == braceLine && f.toks[open-2].line < braceLine {
				f.replaceToken(open-1, "")
			}
		}
		f.endChangeset()
	}
	phpcsContentAfterBrace(f, open, true)
	if lineDifference > 0 {
		return
	}
	ws := f.toks[open-1]
	switch {
	case ws.kind != token.Whitespace:
		f.replaceToken(open, " "+f.tokenContent(open))
	case ws.length != 1 || ws.orig == "\t":
		f.replaceToken(open-1, " ")
	}
}
