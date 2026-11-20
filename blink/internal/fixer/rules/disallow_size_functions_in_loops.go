package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowSizeFunctionsInLoopsSniff.php
//
// DisallowSizeFunctionsInLoops reports sizeof(), strlen() and count() in the
// condition of a while or for loop. It fixes nothing.
type DisallowSizeFunctionsInLoops struct{}

func (DisallowSizeFunctionsInLoops) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\DisallowSizeFunctionsInLoopsSniff`
}

func (DisallowSizeFunctionsInLoops) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowSizeFunctionsInLoopsSniff.php"
}

func (DisallowSizeFunctionsInLoops) Fix(*tokens.Stream) bool { return false }

func (r DisallowSizeFunctionsInLoops) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), func(f *slvFile, sc *phpcsScope, report func(i int, code, msg string)) {
		for loop := range f.toks {
			if !f.isKeyword(loop, "while") && !f.isKeyword(loop, "for") || sc.pCloser[loop] < 0 {
				continue
			}
			closeIdx := sc.pCloser[loop]
			start, end := f.toks[closeIdx].closer, closeIdx
			if f.isKeyword(loop, "for") {
				start, end = squizSizeLoopSemicolons(f, start, closeIdx)
			}
			for i := start + 1; i < end; i++ {
				name := f.toks[i].orig
				fn, _ := strings.CutPrefix(name, `\`)
				if !squizReportIsPlainName(f, i) || !squizSizeLoopFunctions[fn] {
					continue
				}
				if i > 0 && (f.is(i-1, "->") || f.is(i-1, "?->")) {
					continue
				}
				report(i, "Found", "The use of "+name+"() inside a loop condition is not allowed; assign the return value to a variable and use the variable in the loop condition instead")
			}
		}
	})
}

// squizSizeLoopFunctions is the sniff's FORBIDDEN_FUNCTIONS, matched case-sensitively.
var squizSizeLoopFunctions = map[string]bool{"sizeof": true, "strlen": true, "count": true}

// squizSizeLoopSemicolons bounds the condition of a for loop: the first ";"
// after its "(" and the last before its ")", searched without limit as the
// sniff does.
func squizSizeLoopSemicolons(f *slvFile, open, closeIdx int) (int, int) {
	start, end := len(f.toks), -1
	for i := open + 1; i < len(f.toks); i++ {
		if f.is(i, ";") {
			start = i
			break
		}
	}
	for i := closeIdx - 1; i >= 0; i-- {
		if f.is(i, ";") {
			end = i
			break
		}
	}
	return start, end
}
