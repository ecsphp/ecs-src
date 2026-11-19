package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/AssignmentInConditionSniff.php
//
// AssignmentInCondition reports a variable assignment inside the condition of
// an if, elseif, for, switch, case or match. PHP_CodeSniffer raises it
// as a warning, which ECS escalates to an error. It fixes nothing.
type AssignmentInCondition struct{}

func (AssignmentInCondition) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\CodeAnalysis\AssignmentInConditionSniff`
}

func (AssignmentInCondition) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/CodeAnalysis/AssignmentInConditionSniff.php"
}

func (AssignmentInCondition) Fix(*tokens.Stream) bool { return false }

func (r AssignmentInCondition) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	sc := newPhpcsScope(newSlvFile(s.Tokens()))
	f := sc.f
	var errs []fixer.CodingStandardError
	for k := range f.toks {
		opener, closer := assignmentInConditionBounds(sc, k)
		if opener < 0 {
			continue
		}
		for start := opener; start < closer; {
			found := -1
			for j := start + 1; j < closer; j++ {
				if assignmentInConditionAssigns[sc.code[j]] {
					found = j
					break
				}
			}
			if found < 0 {
				break
			}
			condStart := start
			for j := found - 1; j >= start; j-- {
				if assignmentInConditionStarts[sc.code[j]] {
					condStart = j
					break
				}
			}
			for j := found; j > condStart; j-- {
				if f.isEmpty(j) {
					continue
				}
				if f.toks[j].kind == token.Variable || f.is(j, "]") {
					errs = append(errs, fixer.CodingStandardError{
						Line:    f.toks[found].line,
						Message: "Variable assignment found within a condition. Did you mean to do a comparison ?",
						Checker: r.Name() + ".Found",
					})
					break
				}
				if f.is(j, ")") {
					break
				}
			}
			start = found
		}
	}
	return errs
}

// assignmentInConditionAssigns is Tokens::ASSIGNMENT_TOKENS without "=>".
var assignmentInConditionAssigns = phpcsSet("=", "&=", "|=", ".=", "/=", "-=", "**=", "%=", "*=", "+=", "^=",
	"<<=", ">>=", "??=")

// assignmentInConditionStarts is Tokens::BOOLEAN_OPERATORS plus ";" and "(".
var assignmentInConditionStarts = phpcsSet("&&", "||", "and", "or", "xor", ";", "(")

// assignmentInConditionBounds returns the tokens bounding the condition of the
// control structure at k: its parentheses, the two ";" of a for, or a case's
// keyword and scope opener; -1 when k is none or has no condition.
func assignmentInConditionBounds(sc *phpcsScope, k int) (opener, closer int) {
	f := sc.f
	switch sc.code[k] {
	case "case":
		if sc.opener[k] < 0 {
			return -1, -1
		}
		return k, sc.opener[k]
	// a while reports FoundInWhileCondition, which ECS always drops
	case "if", "elseif", "for", "switch", "match":
	default:
		return -1, -1
	}
	pc := sc.pCloser[k]
	if pc < 0 {
		return -1, -1
	}
	po := f.toks[pc].closer
	if sc.code[k] != "for" {
		return po, pc
	}
	opener = -1
	for j := po + 1; j < pc; j++ {
		if sc.code[j] != ";" {
			continue
		}
		if opener < 0 {
			opener = j
			continue
		}
		return opener, j
	}
	return -1, -1
}
