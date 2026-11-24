package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/AssignmentInConditionSniff.php
//
// SlevomatAssignmentInCondition reports "=" in the condition of an if, elseif
// or do-while. It fixes nothing.
type SlevomatAssignmentInCondition struct {
	ignoreAssignmentsInsideFunctionCalls bool
}

func (SlevomatAssignmentInCondition) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\AssignmentInConditionSniff`
}

func (SlevomatAssignmentInCondition) SourceURL() string {
	return fixer.SourceURLFor(SlevomatAssignmentInCondition{}.Name())
}

func (r SlevomatAssignmentInCondition) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["ignoreAssignmentsInsideFunctionCalls"].(bool); ok {
		r.ignoreAssignmentsInsideFunctionCalls = v
	}
	return r
}

func (SlevomatAssignmentInCondition) Fix(*tokens.Stream) bool { return false }

func (r SlevomatAssignmentInCondition) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	sc := newPhpcsScope(newSlvFile(s.Tokens()))
	f := sc.f
	var errs []fixer.CodingStandardError
	report := func(conditionType string, equals int) {
		errs = append(errs, fixer.CodingStandardError{
			Line:    f.toks[equals].line,
			Message: "Assignment in " + conditionType + " condition is not allowed.",
			Checker: r.Name() + ".AssignmentInCondition",
		})
	}
	for i := range f.toks {
		var conditionType string
		owner := i
		switch sc.code[i] {
		case "if", "elseif":
			conditionType = sc.code[i]
		case "do":
			if sc.closer[i] < 0 {
				continue
			}
			owner = -1
			for j := sc.closer[i] + 1; j < len(f.toks); j++ {
				if sc.code[j] == "while" {
					owner = j
					break
				}
			}
			if owner < 0 {
				continue
			}
			conditionType = "do-while"
		default:
			continue
		}
		closer := sc.pCloser[owner]
		if closer < 0 {
			continue
		}
		opener := f.toks[closer].closer
		var equals []int
		for j := opener + 1; j < closer; j++ {
			if sc.code[j] == "=" {
				equals = append(equals, j)
			}
		}
		if len(equals) == 0 {
			continue
		}
		if !r.ignoreAssignmentsInsideFunctionCalls {
			report(conditionType, equals[0])
			continue
		}
		for _, e := range equals {
			inside := sc.nested[e]
			if inside == opener {
				report(conditionType, e)
				continue
			}
			functionCall := false
			for j := inside; j >= opener; j-- {
				if f.isName(j) {
					functionCall = true
					break
				}
			}
			if !functionCall {
				report(conditionType, e)
			}
		}
	}
	return errs
}
