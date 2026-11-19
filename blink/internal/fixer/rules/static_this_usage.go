package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Scope/StaticThisUsageSniff.php
//
// StaticThisUsage reports "$this" used in a static method or closure.
type StaticThisUsage struct{}

func (StaticThisUsage) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Scope\StaticThisUsageSniff`
}

func (StaticThisUsage) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Scope/StaticThisUsageSniff.php"
}

func (StaticThisUsage) Fix(*tokens.Stream) bool { return false }

func (st StaticThisUsage) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	a := abstractScopeLoad(s, st.Name())
	f := a.f
	var check func(next, end int, kind string)
	check = func(next, end int, kind string) {
		for next++; next < end; next++ {
			t := f.toks[next]
			switch c := a.code(next); {
			case (c == "closure" || c == "anon_class") && a.sc.opener[next] >= 0:
				check(next, a.sc.opener[next], kind)
				next = a.sc.closer[next]
				if next < 0 {
					return
				}
			case t.kind == token.Variable && !t.heredoc && !t.interp && t.orig == "$this":
				a.addError(next, "Found", `Usage of "$this" in a static %s will cause runtime errors`, kind)
			}
		}
	}
	a.run(phpcsSet("class", "trait", "enum", "anon_class"), phpcsSet("function", "closure"), func(i, scope int) {
		kind := "closure"
		if a.code(i) == "function" {
			if !a.deepest(i, scope) || a.sc.closer[i] < 0 {
				return
			}
			if next := f.nextEffective(i+1, len(f.toks)); next < 0 || !a.isTString(next) {
				return
			}
			kind = "method"
		}
		if a.methodProps(i).isStatic {
			check(i, a.sc.closer[i], kind)
		}
	}, func(i int) {
		if a.code(i) == "closure" && a.methodProps(i).isStatic {
			check(i, a.sc.closer[i], "closure")
		}
	})
	return a.errors
}
