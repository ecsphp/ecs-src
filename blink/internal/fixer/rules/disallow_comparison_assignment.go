package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowComparisonAssignmentSniff.php
//
// DisallowComparisonAssignment reports assigning the value of a comparison,
// ternary or boolean operation to a variable. It fixes nothing.
type DisallowComparisonAssignment struct{}

func (DisallowComparisonAssignment) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\DisallowComparisonAssignmentSniff`
}

func (DisallowComparisonAssignment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/DisallowComparisonAssignmentSniff.php"
}

func (DisallowComparisonAssignment) Fix(*tokens.Stream) bool { return false }

func (r DisallowComparisonAssignment) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), r.process)
}

func (DisallowComparisonAssignment) process(f *slvFile, sc *phpcsScope, report func(i int, code, msg string)) {
	for i := range f.toks {
		if !f.is(i, "=") {
			continue
		}
		fn := squizReportPrevLocal(f, sc, i-1, func(j int) bool { return f.isKeyword(j, "function") && !f.isClosure(j) })
		if fn >= 0 && sc.pCloser[fn] >= 0 && f.toks[sc.pCloser[fn]].closer < i && i < sc.pCloser[fn] {
			continue
		}
		if n := squizReportNextNonEmpty(f, i+1); f.isKeyword(n, "array") || f.isKeyword(n, "match") {
			continue
		}
		next := i + 1
		for next < len(f.toks) && (squizReportIsPlainName(f, next) || f.is(next, "->") || f.is(next, "?->") ||
			f.toks[next].kind == token.Variable || f.toks[next].kind == token.Whitespace) {
			next++
		}
		if next < len(f.toks) && (f.isKeyword(next, "function") && f.isClosure(next) || f.is(next, "(") && squizReportIsPlainName(f, next-1)) {
			continue
		}
		end := sc.endOfStatement(i)
		for j := i + 1; j < end; j++ {
			if squizReportIsComparison(f, j) && !f.is(j, "??") || squizReportIsInlineThen(f, sc, j) {
				report(i, "AssignedComparison", "The value of a comparison must not be assigned to a variable")
				break
			}
			if squizReportIsBooleanOperator(f, j) || f.is(j, "!") {
				report(i, "AssignedBool", "The value of a boolean operation must not be assigned to a variable")
				break
			}
		}
	}
}
