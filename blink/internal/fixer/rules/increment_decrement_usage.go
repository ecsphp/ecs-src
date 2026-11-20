package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Operators/IncrementDecrementUsageSniff.php
//
// IncrementDecrementUsage reports "$i = $i + 1" and "$i += 1" where "++$i"
// would do, and "++"/"--" inside arithmetic or unbracketed in a concatenation.
// It fixes nothing.
type IncrementDecrementUsage struct{}

func (IncrementDecrementUsage) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Operators\IncrementDecrementUsageSniff`
}

func (IncrementDecrementUsage) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Operators/IncrementDecrementUsageSniff.php"
}

func (IncrementDecrementUsage) Fix(*tokens.Stream) bool { return false }

func (r IncrementDecrementUsage) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), r.process)
}

func (IncrementDecrementUsage) process(f *slvFile, sc *phpcsScope, report func(i int, code, msg string)) {
	for ptr := range f.toks {
		switch {
		case f.is(ptr, "++") || f.is(ptr, "--"):
			squizIncDecProcess(f, ptr, report)
		case f.is(ptr, "=") || f.is(ptr, "+=") || f.is(ptr, "-="):
			squizIncDecAssignment(f, ptr, report)
		}
	}
}

func squizIncDecProcess(f *slvFile, ptr int, report func(i int, code, msg string)) {
	start := ptr + 2
	if ptr > 0 && (f.toks[ptr-1].kind == token.Variable ||
		f.toks[ptr-1].kind == token.Ident && ptr > 1 && (f.is(ptr-2, "->") || f.is(ptr-2, "?->"))) {
		start = ptr + 1
	}
	next := squizReportNextNonEmpty(f, start)
	if next < 0 {
		return
	}
	if squizReportIsArithmetic(f, next) {
		report(ptr, "NotAllowed", "Increment and decrement operators cannot be used in an arithmetic operation")
		return
	}
	prev := squizReportPrevNonEmpty(f, start-3)
	if prev < 0 {
		return
	}
	if f.is(next, ".") || f.is(prev, ".") {
		report(ptr, "NoBrackets", "Increment and decrement operators must be bracketed when used in string concatenation")
	}
}

func squizIncDecAssignment(f *slvFile, ptr int, report func(i int, code, msg string)) {
	assigned := squizReportPrevNonEmpty(f, ptr-1)
	if assigned < 0 || f.toks[assigned].kind != token.Variable {
		return
	}
	end := len(f.toks)
	for j := ptr; j < len(f.toks); j++ {
		if f.is(j, ";") || f.is(j, ")") || f.is(j, "]") || f.is(j, "}") {
			end = j
			break
		}
	}
	isInt := func(j int) bool { return f.toks[j].kind == token.Number && squizIncDecInt.MatchString(f.toks[j].orig) }
	for j := ptr + 1; j < end; j++ {
		if !f.isEmpty(j) && !isInt(j) && f.toks[j].kind != token.Variable && !f.is(j, "+") && !f.is(j, "-") && !f.is(j, "(") {
			return
		}
	}
	nextVar := ptr
	if !f.is(ptr, "=") {
		for j := ptr + 1; j < end; j++ {
			if f.toks[j].kind == token.Variable {
				return
			}
		}
	} else {
		count := 0
		for j := ptr + 1; j < end; j++ {
			if f.toks[j].kind == token.Variable {
				nextVar = j
				count++
			}
		}
		if count != 1 || f.toks[nextVar].content != f.toks[assigned].content {
			return
		}
	}
	number, count := ptr, 0
	for j := ptr + 1; j < end; j++ {
		if isInt(j) {
			number = j
			count++
		}
	}
	if count != 1 || f.toks[number].content != "1" {
		return
	}
	var operator string
	if f.is(ptr, "=") {
		op := -1
		for j := nextVar + 1; j < end; j++ {
			if f.is(j, "+") || f.is(j, "-") {
				op = j
				break
			}
		}
		if op < 0 {
			return
		}
		operator = f.toks[op].content
	} else {
		operator = f.toks[ptr].content[:1]
		for j := number - 1; j >= ptr; j-- {
			if f.is(j, "-") {
				operator = map[string]string{"+": "-", "-": "+"}[operator]
				break
			}
		}
	}
	expected := operator + operator + f.toks[assigned].content
	last := min(end, len(f.toks)-1)
	var found strings.Builder
	for j := assigned; j <= last; j++ {
		found.WriteString(f.toks[j].content)
	}
	kind := "Decrement"
	if operator == "+" {
		kind = "Increment"
	}
	report(ptr, "Found", fmt.Sprintf(`%s operators should be used where possible; found "%s" but expected "%s"`, kind, found.String(), expected))
}

// squizIncDecInt is an integer literal (T_LNUMBER).
var squizIncDecInt = regexp.MustCompile(`^(0[xX][0-9a-fA-F_]+|0[bB][01_]+|0[oO]?[0-7_]*|[1-9][0-9_]*)$`)
