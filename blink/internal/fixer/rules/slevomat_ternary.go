package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// slvTernary is the T_INLINE_THEN / T_INLINE_ELSE view TernaryOperatorHelper
// needs: PHP_CodeSniffer's ternary tokens plus ScopeHelper's function scopes.
type slvTernary struct {
	f      *slvFile
	sc     *phpcsScope
	scopes []int
}

func newSlvTernary(f *slvFile) *slvTernary {
	return &slvTernary{f: f, sc: newPhpcsScope(f), scopes: slvcFunctionScopes(f)}
}

func (t *slvTernary) isThen(i int) bool {
	return t.f.is(i, "?") && t.sc.isInlineThen(i)
}

func (t *slvTernary) isElse(i int) bool {
	return i >= 0 && i < len(t.f.toks) && t.sc.code[i] == "inline_else"
}

// isDoubleArrow tells T_DOUBLE_ARROW: not the "=>" of an arrow function or a
// match arm, which PHP_CodeSniffer tokenizes as T_FN_ARROW / T_MATCH_ARROW.
func (t *slvTernary) isDoubleArrow(i int) bool {
	return t.f.is(i, "=>") && t.sc.code[i] != "fn_arrow" && t.sc.code[i] != "match_arrow"
}

func (t *slvTernary) sameScope(a, b int) bool { return t.scopes[a] == t.scopes[b] }

// start is TernaryOperatorHelper::getStartPointer.
func (t *slvTernary) start(then int) int {
	f := t.f
	p := then - 1
	for ; p >= 0; p-- {
		tok := f.toks[p]
		switch {
		case f.is(p, "=") || t.isDoubleArrow(p) || f.is(p, ",") || tok.kind == token.OpenTag ||
			f.isKeyword(p, "return") || f.isKeyword(p, "throw") || f.isKeyword(p, "case"):
		case f.is(p, "[") || f.is(p, "("):
			if tok.closer >= 0 && tok.closer < then {
				continue
			}
		default:
			continue
		}
		break
	}
	return f.nextEffective(p+1, len(f.toks))
}

// elseOf is TernaryOperatorHelper::getElsePointer, or -1.
func (t *slvTernary) elseOf(then int) int {
	f := t.f
	for p := then + 1; p < len(f.toks); p++ {
		switch {
		case (f.is(p, "(") || f.is(p, "[")) && f.toks[p].closer >= 0:
			p = f.toks[p].closer
		case t.isElse(p):
			if t.sameScope(then, p) {
				return p
			}
		}
	}
	return -1
}

// end is TernaryOperatorHelper::getEndPointer.
func (t *slvTernary) end(then, elsePtr int) int {
	f := t.f
	for p := elsePtr + 1; p < len(f.toks); p++ {
		switch {
		case f.is(p, ")") || f.is(p, "]"):
			if f.toks[p].closer < then {
				return f.prevEffective(p - 1)
			}
		case f.is(p, ","):
			prev := -1
			for q := p - 1; q >= then; q-- {
				if f.is(q, "(") || f.is(q, "[") && f.isShortArrayOpen(q) {
					prev = q
					break
				}
			}
			if prev < 0 || f.toks[prev].closer < p {
				return f.prevEffective(p - 1)
			}
		case t.isDoubleArrow(p):
			found := false
			for q := p - 1; q >= then; q-- {
				if f.is(q, "[") && f.isShortArrayOpen(q) {
					found = true
					break
				}
			}
			if !found {
				return f.prevEffective(p - 1)
			}
		case f.is(p, ";") || f.is(p, "??") || f.is(p, ":") && !t.isElse(p):
			if t.sameScope(elsePtr, p) {
				return f.prevEffective(p - 1)
			}
		}
	}
	return f.prevEffective(len(f.toks) - 1)
}
