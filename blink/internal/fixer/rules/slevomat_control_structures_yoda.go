package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/DisallowYodaComparisonSniff.php
//
// SlevomatDisallowYodaComparison puts the more dynamic side of a comparison on
// the left: "null === $a" -> "$a === null".
type SlevomatDisallowYodaComparison struct{}

func (SlevomatDisallowYodaComparison) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\DisallowYodaComparisonSniff`
}

func (SlevomatDisallowYodaComparison) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowYodaComparison{}.Name())
}

func (SlevomatDisallowYodaComparison) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		f.eachToken(f.slvYodaComparison, func(i int) int {
			left, right := f.slvYodaLeft(sc, i), f.slvYodaRight(sc, i)
			l, lok := f.slvYodaDynamism(sc, left)
			r, rok := f.slvYodaDynamism(sc, right)
			if !lok || !rok || l >= r || l >= 900 && r >= 900 {
				return 0
			}
			if f.is(f.nextEffective(right[len(right)-1]+1, len(f.toks)), "=") {
				return 0
			}
			f.slvYodaFix(left, right)
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireYodaComparisonSniff.php
//
// SlevomatRequireYodaComparison puts the more dynamic side of a comparison on
// the right: "$a === null" -> "null === $a".
type SlevomatRequireYodaComparison struct {
	alwaysVariableOnRight bool
}

func (SlevomatRequireYodaComparison) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireYodaComparisonSniff`
}

func (SlevomatRequireYodaComparison) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireYodaComparison{}.Name())
}

func (r SlevomatRequireYodaComparison) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["alwaysVariableOnRight"].(bool); ok {
		r.alwaysVariableOnRight = v
	}
	return r
}

func (r SlevomatRequireYodaComparison) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		f.eachToken(f.slvYodaComparison, func(i int) int {
			left, right := f.slvYodaLeft(sc, i), f.slvYodaRight(sc, i)
			l, lok := f.slvYodaDynamism(sc, left)
			rd, rok := f.slvYodaDynamism(sc, right)
			if !lok || !rok || l <= rd || !r.alwaysVariableOnRight && l >= 900 && rd >= 900 {
				return 0
			}
			if len(left) == 0 || len(right) == 0 {
				return 0
			}
			f.slvYodaFix(left, right)
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/UselessIfConditionWithReturnSniff.php
//
// SlevomatUselessIfConditionWithReturn turns an if returning true or false and
// its else (or the following return) returning the opposite into one return:
// "if ($a) { return true; } return false;" -> "return $a;".
type SlevomatUselessIfConditionWithReturn struct {
	assumeBoolean bool
}

func (SlevomatUselessIfConditionWithReturn) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\UselessIfConditionWithReturnSniff`
}

func (SlevomatUselessIfConditionWithReturn) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessIfConditionWithReturn{}.Name())
}

func (r SlevomatUselessIfConditionWithReturn) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["assumeAllConditionExpressionsAreAlreadyBoolean"].(bool); ok {
		r.assumeBoolean = v
	}
	return r
}

func (r SlevomatUselessIfConditionWithReturn) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		f.eachToken(func(i int) bool { return f.isKeyword(i, "if") }, func(i int) int {
			r.process(f, sc, i)
			return 0
		})
	})
}

func (r SlevomatUselessIfConditionWithReturn) process(f *slvFile, sc *phpcsScope, ifPtr int) {
	closer := sc.closer[ifPtr]
	if closer < 0 {
		return
	}
	ifBool := f.slvUselessIfBooleanInScope(sc.opener[ifPtr])
	if ifBool < 0 {
		return
	}
	pClose := sc.pCloser[ifPtr]
	if pClose < 0 {
		return
	}
	open := f.toks[pClose].closer
	newCondition := func() string {
		if strings.EqualFold(f.toks[ifBool].content, "true") {
			return f.content(open+1, pClose-1)
		}
		return f.slvCondNegative(open+1, pClose-1, false)
	}

	var end int
	if elsePtr := f.nextEffective(closer+1, len(f.toks)); f.isKeyword(elsePtr, "else") {
		if sc.closer[elsePtr] < 0 || f.slvUselessIfBooleanInScope(sc.opener[elsePtr]) < 0 {
			return
		}
		end = sc.closer[elsePtr]
	} else {
		ret := f.nextEffective(closer+1, len(f.toks))
		if !f.isKeyword(ret, "return") {
			return
		}
		if end = f.slvUselessIfSemicolonAfterBoolean(ret); end < 0 {
			return
		}
	}
	if !r.fixable(f, ifPtr, open, pClose, end) {
		return
	}
	f.beginChangeset()
	f.change(ifPtr, end, "return "+newCondition()+";")
	f.endChangeset()
}

func (r SlevomatUselessIfConditionWithReturn) fixable(f *slvFile, ifPtr, open, pClose, end int) bool {
	if f.slvCondFindNext(ifPtr+1, end, func(i int) bool {
		return f.toks[i].kind == token.Comment || f.toks[i].kind == token.DocComment
	}) >= 0 {
		return false
	}
	return r.assumeBoolean || f.slvCondReturnsBoolean(open+1, pClose-1)
}

// slvUselessIfBooleanAfterReturn is findBooleanAfterReturn: the true or false
// after the return at ret, or -1.
func (f *slvFile) slvUselessIfBooleanAfterReturn(ret int) int {
	b := f.nextEffective(ret+1, len(f.toks))
	if b >= 0 && f.toks[b].kind == token.Ident {
		switch strings.ToLower(strings.TrimPrefix(f.toks[b].orig, `\`)) {
		case "true", "false":
			if !f.lineCallIsName(b) {
				return b
			}
		}
	}
	return -1
}

// slvUselessIfBooleanInScope is findBooleanAfterReturnInScope.
func (f *slvFile) slvUselessIfBooleanInScope(opener int) int {
	ret := f.nextEffective(opener+1, len(f.toks))
	if !f.isKeyword(ret, "return") {
		return -1
	}
	b := f.slvUselessIfBooleanAfterReturn(ret)
	if b < 0 || !f.is(f.nextEffective(b+1, len(f.toks)), ";") {
		return -1
	}
	return b
}

// slvUselessIfSemicolonAfterBoolean is findSemicolonAfterReturnWithBoolean.
func (f *slvFile) slvUselessIfSemicolonAfterBoolean(ret int) int {
	b := f.slvUselessIfBooleanAfterReturn(ret)
	if b < 0 {
		return -1
	}
	semi := f.nextEffective(b+1, len(f.toks))
	if !f.is(semi, ";") {
		return -1
	}
	return semi
}
