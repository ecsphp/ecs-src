package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Objects/ObjectInstantiationSniff.php
//
// ObjectInstantiation reports a "new" expression whose object is not assigned,
// returned or thrown.
type ObjectInstantiation struct{}

func (ObjectInstantiation) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Objects\ObjectInstantiationSniff`
}

func (ObjectInstantiation) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Objects/ObjectInstantiationSniff.php"
}

func (ObjectInstantiation) Fix(*tokens.Stream) bool { return false }

func (r ObjectInstantiation) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(s.Tokens())
	sc := newPhpcsScope(f)
	allowed := func(j int) bool {
		return f.is(j, "=") || f.is(j, "??=") || f.is(j, "=>") || f.isKeyword(j, "throw") || f.isKeyword(j, "return")
	}
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if !f.isKeyword(i, "new") {
			continue
		}
		prev := i - 1
		for prev >= 0 && (f.isEmpty(prev) || f.is(prev, "&")) {
			prev--
		}
		if prev < 0 {
			prev = 0
		}
		if allowed(prev) {
			continue
		}
		if f.is(prev, "??") || f.is(prev, "?") && sc.isInlineThen(prev) || sc.code[prev] == "inline_else" {
			if objectInstantiationAllowedBefore(f, sc, prev-1, allowed) {
				continue
			}
		}
		errs = append(errs, squizReportError(r.Name(), "NotAssigned", f.toks[i].line, "New objects must be assigned to a variable"))
	}
	return errs
}

// objectInstantiationAllowedBefore is File::findPrevious($allowed, $start,
// null, false, null, true): it searches back within the statement, jumping
// over scopes, brackets and parentheses it meets by their closer.
func objectInstantiationAllowedBefore(f *slvFile, sc *phpcsScope, start int, allowed func(int) bool) bool {
	for j := start; j >= 0; j-- {
		if allowed(j) {
			return true
		}
		switch {
		case sc.cond[j] >= 0 && sc.closer[j] == j && sc.opener[j] >= 0:
			j = sc.opener[j]
		case (f.is(j, "]") || f.is(j, ")")) && f.toks[j].closer >= 0:
			j = f.toks[j].closer
		case f.is(j, ";"):
			return false
		}
	}
	return false
}
