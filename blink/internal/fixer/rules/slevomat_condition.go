package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// slvBooleanOperator tells PHP_CodeSniffer's Tokens::BOOLEAN_OPERATORS.
func (f *slvFile) slvBooleanOperator(i int) bool {
	return f.is(i, "&&") || f.is(i, "||") || f.isKeyword(i, "and") || f.isKeyword(i, "or") || f.isKeyword(i, "xor")
}

// slvEquality tells PHP_CodeSniffer's Tokens::EQUALITY_TOKENS.
func (f *slvFile) slvEquality(i int) bool {
	if i < 0 || i >= len(f.toks) || f.toks[i].kind != token.Punct {
		return false
	}
	switch f.toks[i].orig {
	case "==", "!=", "<>", "===", "!==", "<=", ">=":
		return true
	}
	return false
}

// slvIsTString tells a T_STRING: a plain name, not true/false/null/self/parent.
func (f *slvFile) slvIsTString(i int) bool {
	if i < 0 || i >= len(f.toks) || f.toks[i].kind != token.Ident {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "true", "false", "null", "self", "parent":
		return false
	}
	return true
}

// slvConditionReturnsBoolean is ConditionHelper::conditionReturnsBoolean.
func (f *slvFile) slvConditionReturnsBoolean(start, end int) bool {
	content := strings.ToLower(strings.Trim(f.content(start, end), phpTrimChars))
	if content == "false" || content == "true" {
		return true
	}
	for i := start; i <= end && i < len(f.toks); i++ {
		if f.is(i, "(") {
			if f.toks[i].closer < 0 {
				return false
			}
			i = f.toks[i].closer
			continue
		}
		if f.is(i, "<") || f.is(i, ">") || f.slvBooleanOperator(i) || f.slvEquality(i) {
			return true
		}
	}
	return false
}

// slvNegativeCondition is ConditionHelper::getNegativeCondition.
func (f *slvFile) slvNegativeCondition(boundStart, boundEnd int, nested bool) string {
	start := f.nextEffective(boundStart, len(f.toks))
	end := f.prevEffective(boundEnd)
	if f.is(start, "(") && f.toks[start].closer == end {
		start = f.nextEffective(start+1, len(f.toks))
		end = f.prevEffective(end - 1)
	}
	var b strings.Builder
	if boundStart != start {
		b.WriteString(f.content(boundStart, start-1))
	}
	b.WriteString(f.slvNegativeConditionPart(start, end, nested))
	if boundEnd != end {
		b.WriteString(f.content(end+1, boundEnd))
	}
	return b.String()
}

var slvBooleanNotPrefix = regexp.MustCompile(`^!\s*`)

func (f *slvFile) slvNegativeConditionPart(start, end int, nested bool) string {
	condition := f.content(start, end)
	switch strings.ToLower(condition) {
	case "true":
		return "false"
	case "false":
		return "true"
	}
	first := f.nextEffective(start, len(f.toks))
	hasBoolean := false
	for i := start; i <= end; i++ {
		if f.slvBooleanOperator(i) {
			hasBoolean = true
			break
		}
	}
	if f.is(first, "!") {
		afterNot := f.nextEffective(first+1, len(f.toks))
		if f.is(afterNot, "(") && f.toks[afterNot].closer >= 0 {
			if nested && hasBoolean {
				return slvBooleanNotPrefix.ReplaceAllString(condition, "")
			}
			closer := f.toks[afterNot].closer
			after := f.nextEffective(closer+1, end+1)
			if after < 0 || after == end {
				return f.content(afterNot+1, closer-1)
			}
		}
	}
	if hasBoolean {
		return f.slvNegativeLogicalCondition(start, end)
	}
	if f.is(first, "!") {
		return slvBooleanNotPrefix.ReplaceAllString(condition, "")
	}
	for i := start; i <= end; i++ {
		if f.isKeyword(i, "instanceof") || f.is(i, "&") || f.is(i, "??") || f.is(i, "?") {
			return "!(" + condition + ")"
		}
	}
	if f.slvIsTString(first) {
		next := f.nextEffective(first+1, len(f.toks))
		if f.is(next, "(") && f.toks[next].closer == end {
			return "!" + condition
		}
		first = next
	}
	if f.isVariable(first) || f.isClassKeyword(first) {
		if identEnd := f.identEnd(first); identEnd >= 0 {
			after := f.nextEffective(identEnd+1, len(f.toks))
			if f.is(after, "(") && f.toks[after].closer == end {
				return "!" + condition
			}
		}
	}
	replacements := map[string]string{
		"==": "!=", "!=": "==", "<>": "==", "===": "!==", "!==": "===",
		">=": "<", "<=": ">", ">": "<=", "<": ">=",
	}
	hasComparison := false
	for i := start; i <= end; i++ {
		if f.toks[i].kind == token.Punct {
			if _, ok := replacements[f.toks[i].orig]; ok {
				hasComparison = true
				break
			}
		}
	}
	if !hasComparison {
		return "!" + condition
	}
	var b strings.Builder
	for i := start; i <= end; i++ {
		if f.is(i, "(") && f.toks[i].closer >= 0 {
			b.WriteString(f.content(i, f.toks[i].closer))
			i = f.toks[i].closer
			continue
		}
		if r, ok := replacements[f.toks[i].orig]; ok && f.toks[i].kind == token.Punct {
			b.WriteString(r)
			continue
		}
		b.WriteString(f.toks[i].content)
	}
	return b.String()
}

func (f *slvFile) slvNegativeLogicalCondition(start, end int) string {
	for i := start; i < end; i++ {
		if f.isKeyword(i, "xor") {
			return "!(" + f.content(start, end) + ")"
		}
	}
	replacements := map[string]string{"&&": "||", "||": "&&", "and": "or", "or": "and"}
	var b strings.Builder
	nestedStart := start
	level := 0
	levelOperator := ""
	for i := start; i <= end; i++ {
		switch {
		case f.is(i, "("):
			if f.slvIsTString(f.prevEffective(i-1)) && f.toks[i].closer >= 0 {
				i = f.toks[i].closer
			} else {
				level++
			}
			continue
		case f.is(i, ")"):
			level--
			continue
		case !f.slvBooleanOperator(i):
			continue
		}
		if level != 0 {
			continue
		}
		op := strings.ToLower(f.toks[i].orig)
		if levelOperator != "" && levelOperator != op {
			return fmt.Sprintf("!(%s)", f.content(start, end))
		}
		levelOperator = op
		b.WriteString(f.slvNegativeCondition(nestedStart, i-1, true))
		b.WriteString(replacements[op])
		nestedStart = i + 1
	}
	b.WriteString(f.slvNegativeCondition(nestedStart, end, true))
	return b.String()
}
