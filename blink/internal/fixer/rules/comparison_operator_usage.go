package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Operators/ComparisonOperatorUsageSniff.php
//
// ComparisonOperatorUsage reports "==", "!=" and "!" in the conditions of if,
// elseif, while, for and ternaries, and conditions without an explicit
// comparison. It fixes nothing.
type ComparisonOperatorUsage struct{}

func (ComparisonOperatorUsage) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Operators\ComparisonOperatorUsageSniff`
}

func (ComparisonOperatorUsage) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Operators/ComparisonOperatorUsageSniff.php"
}

func (ComparisonOperatorUsage) Fix(*tokens.Stream) bool { return false }

func (r ComparisonOperatorUsage) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), r.process)
}

func (ComparisonOperatorUsage) process(f *slvFile, sc *phpcsScope, report func(i int, code, msg string)) {
	for ptr := range f.toks {
		var start, end int
		switch {
		case squizReportIsInlineThen(f, sc, ptr):
			end = squizReportPrevNonEmpty(f, ptr-1)
			if !f.is(end, ")") {
				i := end - 1
				for ; i >= 0; i-- {
					if f.is(i, ";") || f.toks[i].kind == token.OpenTag ||
						f.is(i, "}") && sc.opener[i] >= 0 ||
						f.is(i, "{") && sc.closer[i] >= 0 ||
						f.is(i, "(") && f.toks[i].closer >= ptr {
						break
					}
				}
				start = squizReportNextNonEmpty(f, i+1)
			} else {
				if f.toks[end].closer < 0 {
					continue
				}
				start = f.toks[end].closer
			}
		case f.isKeyword(ptr, "for"):
			if sc.pCloser[ptr] < 0 {
				continue
			}
			closeIdx := sc.pCloser[ptr]
			start, end = -1, -1
			for j := f.toks[closeIdx].closer; j < closeIdx; j++ {
				if f.is(j, ";") {
					if start < 0 {
						start = j
					} else {
						end = j
						break
					}
				}
			}
			if start < 0 || end < 0 {
				continue
			}
		case f.isKeyword(ptr, "if"), f.isKeyword(ptr, "elseif"), f.isKeyword(ptr, "while"):
			if sc.pCloser[ptr] < 0 {
				continue
			}
			end = sc.pCloser[ptr]
			start = f.toks[end].closer
		default:
			continue
		}
		required, found, booleans := 0, 0, 0
		lastNonEmpty := start
		for i := start; i <= end && i < len(f.toks); i++ {
			if alt, ok := squizComparisonInvalid(f, i); ok {
				report(i, "NotAllowed", fmt.Sprintf("Operator %s prohibited; use %s instead", f.toks[i].content, alt))
				found++
			} else if squizComparisonValid(f, i) {
				found++
			}
			if f.is(i, "(") && f.toks[i].closer >= 0 && squizComparisonFunctionName(f, lastNonEmpty) {
				i = f.toks[i].closer
				lastNonEmpty = i
				continue
			}
			if squizReportIsBool(f, i) {
				booleans++
			}
			if f.is(i, "&&") || f.is(i, "||") {
				required++
				if found > required {
					found = required
				}
				if required != found {
					report(ptr, "ImplicitTrue", "Implicit true comparisons prohibited; use === TRUE instead")
					found++
				}
			}
			if !f.isEmpty(i) {
				lastNonEmpty = i
			}
		}
		required++
		if found < required && required != booleans {
			report(ptr, "ImplicitTrue", "Implicit true comparisons prohibited; use === TRUE instead")
		}
	}
}

// squizComparisonInvalid is the sniff's INVALID_OPERATIONS.
func squizComparisonInvalid(f *slvFile, i int) (string, bool) {
	switch {
	case f.is(i, "=="):
		return "===", true
	case f.is(i, "!=") || f.is(i, "<>"):
		return "!==", true
	case f.is(i, "!"):
		return "=== FALSE", true
	}
	return "", false
}

// squizComparisonValid is the sniff's VALID_OPERATIONS.
func squizComparisonValid(f *slvFile, i int) bool {
	return f.is(i, "===") || f.is(i, "!==") || f.is(i, "<") || f.is(i, ">") || f.is(i, ">=") || f.is(i, "<=") || f.isKeyword(i, "instanceof")
}

// squizComparisonFunctionName is Tokens::FUNCTION_NAME_TOKENS.
func squizComparisonFunctionName(f *slvFile, i int) bool {
	if f.toks[i].kind == token.Ident {
		switch strings.ToLower(strings.TrimPrefix(f.toks[i].orig, `\`)) {
		case "true", "false", "null":
			return squizReportIsMember(f, i)
		}
		return true
	}
	for _, kw := range []string{"require_once", "require", "include_once", "include", "isset", "unset", "empty", "static", "exit", "die", "eval"} {
		if f.isKeyword(i, kw) {
			return true
		}
	}
	return f.isKeyword(i, "class") && f.isKeyword(f.prevEffective(i-1), "new")
}
