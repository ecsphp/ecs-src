package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// PHP_CodeSniffer token facts and File::findPrevious($local) on phpcsScope,
// plus TernaryOperatorHelper::isConditionOfTernaryOperator.

// isTColon tells a T_COLON: a ":" that is no ternary else.
func (sc *phpcsScope) isTColon(i int) bool {
	return sc.f.is(i, ":") && sc.code[i] == ":"
}

// isNull tells T_NULL.
func (sc *phpcsScope) isNull(i int) bool {
	f := sc.f
	return i >= 0 && f.toks[i].kind == token.Ident && strings.EqualFold(f.toks[i].orig, "null") && !f.lineCallIsName(i)
}

// isTString tells a T_STRING: an unqualified name.
func (sc *phpcsScope) isTString(i int) bool {
	return i >= 0 && sc.f.lineCallIsPlainString(i)
}

// level is the PHP_CodeSniffer token level: how many scope conditions hold i.
func (sc *phpcsScope) level(i int) int {
	return len(sc.conditions(i))
}

// prevLocal is File::findPrevious with $local: it searches back from start to
// end (inclusive), skipping over bracketed blocks and stopping at a ";".
func (sc *phpcsScope) prevLocal(start, end int, match func(i int) bool) int {
	f := sc.f
	for i := start; i >= max(end, 0); i-- {
		if match(i) {
			return i
		}
		switch {
		case sc.closer[i] == i && sc.opener[i] >= 0:
			i = sc.opener[i]
		case (f.is(i, ")") || f.is(i, "]") || f.is(i, "}")) && f.toks[i].closer >= 0 && f.toks[i].closer < i:
			i = f.toks[i].closer
		case f.is(i, ";"):
			return -1
		}
	}
	return -1
}

// isConditionOf is TernaryOperatorHelper::isConditionOfTernaryOperator.
func (t *slvTernary) isConditionOf(p int) bool {
	then := t.f.slvCondFindNext(p+1, len(t.f.toks), t.isThen)
	return then >= 0 && t.start(then) <= p
}
