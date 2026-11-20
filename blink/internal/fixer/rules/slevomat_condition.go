package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// slevomat's ConditionHelper.

var slvCondBooleanOps = map[string]bool{"&&": true, "||": true, "and": true, "or": true, "xor": true}

func (f *slvFile) slvCondBoolOp(i int) string {
	t := f.toks[i]
	switch {
	case t.kind == token.Punct && (t.orig == "&&" || t.orig == "||"):
		return t.orig
	case t.kind == token.Keyword && slvCondBooleanOps[strings.ToLower(t.orig)]:
		return strings.ToLower(t.orig)
	}
	return ""
}

// slvCondFindNext is TokenHelper::findNext: the first token in [start, end)
// match accepts, or -1.
func (f *slvFile) slvCondFindNext(start, end int, match func(i int) bool) int {
	for i := max(start, 0); i < end && i < len(f.toks); i++ {
		if match(i) {
			return i
		}
	}
	return -1
}

var slvCondEquality = phpcsSet("==", "!=", "<>", "===", "!==", "<=", ">=")

// slvCondReturnsBoolean is ConditionHelper::conditionReturnsBoolean.
func (f *slvFile) slvCondReturnsBoolean(start, end int) bool {
	switch strings.ToLower(strings.TrimSpace(f.content(start, end))) {
	case "true", "false":
		return true
	}
	for i := start; ; {
		i = f.slvCondFindNext(i, end+1, func(j int) bool {
			return f.is(j, "(") || f.is(j, "<") || f.is(j, ">") || f.slvCondBoolOp(j) != "" ||
				f.toks[j].kind == token.Punct && slvCondEquality[f.toks[j].orig]
		})
		if i < 0 {
			return false
		}
		if f.is(i, "(") {
			i = f.toks[i].closer
			continue
		}
		return true
	}
}

// slvCondNegative is ConditionHelper::getNegativeCondition.
func (f *slvFile) slvCondNegative(start, end int, nested bool) string {
	condStart := f.nextEffective(start, len(f.toks))
	condEnd := f.prevEffective(end)
	if f.is(condStart, "(") && f.toks[condStart].closer == condEnd {
		condStart = f.nextEffective(condStart+1, len(f.toks))
		condEnd = f.prevEffective(condEnd - 1)
	}
	var b strings.Builder
	if start != condStart {
		b.WriteString(f.content(start, condStart-1))
	}
	b.WriteString(f.slvCondNegativePart(condStart, condEnd, nested))
	if end != condEnd {
		b.WriteString(f.content(condEnd+1, end))
	}
	return b.String()
}

var slvCondBooleanNot = regexp.MustCompile(`^!\s*`)

var slvCondComparisonNegation = map[string]string{
	"==": "!=", "!=": "==", "<>": "==", "===": "!==", "!==": "===",
	">=": "<", "<=": ">", ">": "<=", "<": ">=",
}

func (f *slvFile) slvCondNegativePart(start, end int, nested bool) string {
	condition := f.content(start, end)
	switch strings.ToLower(condition) {
	case "true":
		return "false"
	case "false":
		return "true"
	}

	first := f.nextEffective(start, len(f.toks))
	hasBooleanOp := f.slvCondFindNext(start, end+1, func(i int) bool { return f.slvCondBoolOp(i) != "" }) >= 0

	if f.is(first, "!") {
		afterNot := f.nextEffective(first+1, len(f.toks))
		if f.is(afterNot, "(") {
			if nested && hasBooleanOp {
				return slvCondBooleanNot.ReplaceAllString(condition, "")
			}
			closer := f.toks[afterNot].closer
			after := f.nextEffective(closer+1, end+1)
			if after < 0 || after == end {
				return f.content(afterNot+1, closer-1)
			}
		}
	}

	if hasBooleanOp {
		return f.slvCondNegativeLogical(start, end)
	}

	if f.is(first, "!") {
		return slvCondBooleanNot.ReplaceAllString(condition, "")
	}

	if f.slvCondFindNext(start, end+1, func(i int) bool {
		return f.isKeyword(i, "instanceof") || f.is(i, "&") || f.is(i, "??") || f.is(i, "?")
	}) >= 0 {
		return "!(" + condition + ")"
	}

	ptr := first
	if ptr >= 0 && f.lineCallIsPlainString(ptr) {
		ptr = f.nextEffective(ptr+1, len(f.toks))
		if f.is(ptr, "(") && f.toks[ptr].closer == end {
			return "!" + condition
		}
	}

	if ptr >= 0 && (f.isVariable(ptr) || f.isClassKeyword(ptr)) {
		identEnd := f.identEnd(ptr)
		// IdentificatorHelper::findEndPointer returns null here, and null + 1 is 1
		from := identEnd + 1
		if identEnd < 0 {
			from = 1
		}
		after := f.nextEffective(from, len(f.toks))
		if f.is(after, "(") && f.toks[after].closer == end {
			return "!" + condition
		}
	}

	isComparison := func(i int) bool {
		t := f.toks[i]
		_, ok := slvCondComparisonNegation[t.orig]
		return t.kind == token.Punct && ok
	}
	if f.slvCondFindNext(start, end+1, isComparison) >= 0 {
		var b strings.Builder
		for i := start; i <= end; i++ {
			if f.is(i, "(") {
				b.WriteString(f.content(i, f.toks[i].closer))
				i = f.toks[i].closer
				continue
			}
			if isComparison(i) {
				b.WriteString(slvCondComparisonNegation[f.toks[i].orig])
			} else {
				b.WriteString(f.toks[i].content)
			}
		}
		return b.String()
	}

	return "!" + condition
}

var slvCondLogicalNegation = map[string]string{"&&": "||", "||": "&&", "and": "or", "or": "and"}

// slvCondNegativeLogical is ConditionHelper::getNegativeLogicalCondition.
func (f *slvFile) slvCondNegativeLogical(start, end int) string {
	if f.slvCondFindNext(start, end, func(i int) bool { return f.slvCondBoolOp(i) == "xor" }) >= 0 {
		return "!(" + f.content(start, end) + ")"
	}

	var b strings.Builder
	nestedStart := start
	level := 0
	levelOp := ""
	for i := start; ; {
		i = f.slvCondFindNext(i, end+1, func(j int) bool {
			return f.is(j, "(") || f.is(j, ")") || f.slvCondBoolOp(j) != ""
		})
		if i < 0 {
			break
		}
		if f.is(i, "(") {
			if p := f.prevEffective(i - 1); p >= 0 && f.lineCallIsPlainString(p) {
				i = f.toks[i].closer + 1
				continue
			}
			level++
			i++
			continue
		}
		if f.is(i, ")") {
			level--
			i++
			continue
		}
		if level != 0 {
			i++
			continue
		}
		op := f.slvCondBoolOp(i)
		if levelOp != "" && levelOp != op {
			return "!(" + f.content(start, end) + ")"
		}
		levelOp = op
		b.WriteString(f.slvCondNegative(nestedStart, i-1, true))
		b.WriteString(slvCondLogicalNegation[op])
		nestedStart = i + 1
		i++
	}
	b.WriteString(f.slvCondNegative(nestedStart, end, true))
	return b.String()
}
