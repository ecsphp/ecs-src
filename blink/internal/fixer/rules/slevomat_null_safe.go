package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireNullSafeObjectOperatorSniff.php
//
// SlevomatRequireNullSafeObjectOperator turns "$a !== null && $a->b()" and
// "$a !== null ? $a->b : null" into "$a?->b()" and "$a?->b".
type SlevomatRequireNullSafeObjectOperator struct {
	disabled bool
}

func (SlevomatRequireNullSafeObjectOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireNullSafeObjectOperatorSniff`
}

func (SlevomatRequireNullSafeObjectOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireNullSafeObjectOperator{}.Name())
}

func (r SlevomatRequireNullSafeObjectOperator) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["enable"].(bool); ok {
		r.disabled = !v
	}
	return r
}

func (r SlevomatRequireNullSafeObjectOperator) Fix(s *tokens.Stream) bool {
	if r.disabled {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		t := newSlvTernary(f)
		ns := slvNullSafe{t: t, sc: t.sc}
		f.eachToken(func(i int) bool { return f.is(i, "===") || f.is(i, "!==") }, ns.process)
	})
}

type slvNullSafe struct {
	t  *slvTernary
	sc *phpcsScope
}

func (ns slvNullSafe) process(identical int) int {
	sc := ns.sc
	f := sc.f
	before, after := f.prevEffective(identical-1), f.nextEffective(identical+1, len(f.toks))
	if !sc.isNull(before) && !sc.isNull(after) {
		return identical + 1
	}
	idStart, idEnd, condStart := ns.conditionData(before, after)
	if idStart < 0 || idEnd < 0 {
		return identical + 1
	}
	yoda := sc.isNull(before)
	identificator := f.identContent(idStart, idEnd)
	from := after
	if yoda {
		from = idEnd
	}
	afterCondition := f.nextEffective(from+1, len(f.toks))
	allowed := "||"
	if f.is(identical, "!==") {
		allowed = "&&"
	}
	if f.is(afterCondition, allowed) {
		return ns.checkNextCondition(identical, condStart, identificator, afterCondition)
	}
	if ns.t.isThen(afterCondition) {
		ns.checkTernary(identical, condStart, identificator, afterCondition)
		return afterCondition + 1
	}
	return identical + 1
}

func (ns slvNullSafe) isIdentical(i int) bool {
	return ns.sc.f.is(i, "===") || ns.sc.f.is(i, "!==")
}

func (ns slvNullSafe) isMember(i int) bool {
	f := ns.sc.f
	return f.is(i, "::") || f.is(i, "->") || f.is(i, "?->")
}

func (ns slvNullSafe) checkTernary(identical, condStart int, identificator string, then int) {
	sc := ns.sc
	f := sc.f
	search := ns.t.start(then)
	for {
		boolean := f.slvCondFindNext(search, then, f.isBooleanOperator)
		if boolean < 0 {
			break
		}
		identical = f.slvCondFindNext(search, boolean, ns.isIdentical)
		if identical < 0 || !sc.isNull(f.nextEffective(identical+1, len(f.toks))) {
			return
		}
		search = boolean + 1
	}

	if before := f.prevEffective(condStart - 1); f.is(before, "&&") || f.is(before, "||") {
		if prevIdentical := sc.prevLocal(before, 0, ns.isIdentical); prevIdentical >= 0 {
			pStart, pEnd, _ := ns.conditionData(f.prevEffective(prevIdentical-1), f.nextEffective(prevIdentical+1, len(f.toks)))
			if pStart >= 0 && pEnd >= 0 && !slvNullSafeCompatible(f.identContent(pStart, pEnd), identificator) {
				return
			}
		}
	}

	inlineElse := ns.t.elseOf(then)
	if inlineElse < 0 {
		return
	}
	elseEnd := ns.t.end(then, inlineElse)
	var difference, defaultContent string
	var condEnd int
	if f.is(identical, "!==") {
		nStart, nEnd := ns.nextIdentificator(then)
		if nEnd < 0 || !slvNullSafeCompatible(identificator, f.identContent(nStart, nEnd)) ||
			f.nextEffective(nEnd+1, len(f.toks)) != inlineElse {
			return
		}
		difference = ns.difference(identificator, nStart, nEnd)
		defaultContent = f.content(f.nextEffective(inlineElse+1, len(f.toks)), elseEnd)
		condEnd = elseEnd
	} else {
		null := f.nextEffective(then+1, len(f.toks))
		if !sc.isNull(null) || f.nextEffective(null+1, len(f.toks)) != inlineElse {
			return
		}
		nStart, nEnd := ns.nextIdentificator(inlineElse)
		if nEnd < 0 || nEnd != elseEnd || !slvNullSafeCompatible(identificator, f.identContent(nStart, nEnd)) {
			return
		}
		difference = ns.difference(identificator, nStart, nEnd)
		defaultContent = strings.TrimSpace(f.content(then+1, inlineElse-1))
		condEnd = nEnd
	}
	content := identificator + "?" + difference
	if strings.ToLower(defaultContent) != "null" {
		content += " ?? " + defaultContent
	}
	f.beginChangeset()
	f.change(condStart, condEnd, content)
	f.endChangeset()
}

func (ns slvNullSafe) checkNextCondition(identical, condStart int, identificator string, boolean int) int {
	sc := ns.sc
	f := sc.f
	nStart, nEnd := ns.nextIdentificator(boolean)
	if nEnd < 0 {
		return boolean
	}
	if !slvNullSafeCompatible(identificator, f.identContent(nStart, nEnd)) {
		return nEnd
	}
	afterNext := f.nextEffective(nEnd+1, len(f.toks))
	if !ns.isIdentical(afterNext) || !sc.isNull(f.nextEffective(afterNext+1, len(f.toks))) {
		return afterNext
	}
	difference := ns.difference(identificator, nStart, nEnd)
	ofTernary := ns.t.isConditionOf(identical)
	f.beginChangeset()
	f.change(condStart, nEnd, identificator+"?"+difference)
	f.endChangeset()
	if ofTernary {
		return f.slvCondFindNext(identical+1, len(f.toks), ns.t.isThen)
	}
	return afterNext
}

func (ns slvNullSafe) nextIdentificator(before int) (int, int) {
	start := ns.sc.f.nextEffective(before+1, len(ns.sc.f.toks))
	return start, ns.identificatorEnd(start)
}

func (ns slvNullSafe) identificatorStart(end int) int {
	f := ns.sc.f
	var start int
	if f.is(end, ")") {
		start = f.identStart(f.prevEffective(f.toks[end].closer - 1))
	} else {
		start = f.identStart(end)
	}
	if start >= 0 {
		if before := f.prevEffective(start - 1); ns.isMember(before) {
			return ns.identificatorStart(f.prevEffective(before - 1))
		}
	}
	return start
}

func (ns slvNullSafe) identificatorEnd(start int) int {
	sc := ns.sc
	f := sc.f
	if start < 0 {
		return -1
	}
	end := start
	if !sc.isTString(start) {
		end = f.identEnd(start)
	}
	if end >= 0 {
		after := f.nextEffective(end+1, len(f.toks))
		if f.is(after, "(") && f.toks[after].closer >= 0 {
			end = f.toks[after].closer
			after = f.nextEffective(end+1, len(f.toks))
		}
		if ns.isMember(after) {
			return ns.identificatorEnd(f.nextEffective(after+1, len(f.toks)))
		}
	}
	return end
}

func (ns slvNullSafe) conditionData(before, after int) (start, end, condStart int) {
	if ns.sc.isNull(before) {
		return after, ns.identificatorEnd(after), before
	}
	start = ns.identificatorStart(before)
	return start, before, start
}

func (ns slvNullSafe) difference(identificator string, nStart, nEnd int) string {
	f := ns.sc.f
	want := strings.Count(identificator, "->")
	count := 0
	diff := nStart
	for i := nStart; i <= nEnd; i++ {
		if f.is(i, "->") || f.is(i, "?->") {
			count++
		}
		if count > want {
			diff = i
			break
		}
	}
	return f.content(diff, nEnd)
}

// slvNullSafeSplit is preg_split('~(::|->|\?->)~') with the delimiters kept.
func slvNullSafeSplit(v string) []string {
	var parts []string
	last := 0
	for i := 0; i < len(v); {
		op := ""
		switch {
		case strings.HasPrefix(v[i:], "::"):
			op = "::"
		case strings.HasPrefix(v[i:], "->"):
			op = "->"
		case strings.HasPrefix(v[i:], "?->"):
			op = "?->"
		}
		if op == "" {
			i++
			continue
		}
		parts = append(parts, v[last:i], op)
		i += len(op)
		last = i
	}
	return append(parts, v[last:])
}

func slvNullSafeCompatible(first, second string) bool {
	a, b := slvNullSafeSplit(first), slvNullSafeSplit(second)
	n := min(len(a), len(b))
	for i := range n {
		if a[i] == "?->" && b[i] == "->" {
			continue
		}
		if a[i] != b[i] {
			return false
		}
	}
	return n < len(b) && b[n] == "->"
}
