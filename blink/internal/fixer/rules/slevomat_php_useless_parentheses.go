package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/PHP/UselessParenthesesSniff.php
//
// SlevomatUselessParentheses removes parentheses that change nothing: around
// a ternary condition, a case value, a variable or call, a string, an
// expression of one operator group, or "new" (ignoreComplexTernaryConditions,
// enableCheckAroundNew).
type SlevomatUselessParentheses struct {
	ignoreComplexTernary bool
	checkAroundNew       bool
}

func (SlevomatUselessParentheses) Name() string {
	return `SlevomatCodingStandard\Sniffs\PHP\UselessParenthesesSniff`
}

func (SlevomatUselessParentheses) SourceURL() string {
	return fixer.SourceURLFor(SlevomatUselessParentheses{}.Name())
}

func (r SlevomatUselessParentheses) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["ignoreComplexTernaryConditions"].(bool); ok {
		r.ignoreComplexTernary = v
	}
	if v, ok := config["enableCheckAroundNew"].(bool); ok {
		r.checkAroundNew = v
	}
	return r
}

// slvupOperatorGroups are the sniff's OPERATORS with their OPERATOR_GROUPS.
var slvupOperatorGroups = map[string]int{
	"**": 1, "*": 2, "/": 2, "%": 3, "+": 4, "-": 4, ".": 5,
}

// slvupComparison is PHP_CodeSniffer's Tokens::COMPARISON_TOKENS.
var slvupComparison = map[string]bool{
	"==": true, "!=": true, "<>": true, "===": true, "!==": true, "<=": true, ">=": true,
	"<": true, ">": true, "<=>": true, "??": true,
}

// slvupSkippedBefore are the keywords after which "(" is never checked.
var slvupSkippedBefore = map[string]bool{
	"isset": true, "unset": true, "empty": true, "function": true, "fn": true, "use": true,
	"class": true, "new": true, "static": true, "exit": true, "die": true, "eval": true,
	"list": true, "include": true, "include_once": true, "require": true, "require_once": true,
	"match": true,
}

// slvupOwners are PHP_CodeSniffer's Tokens::PARENTHESIS_OPENERS keywords.
var slvupOwners = map[string]bool{
	"array": true, "list": true, "function": true, "fn": true, "use": true, "class": true,
	"while": true, "for": true, "foreach": true, "switch": true, "if": true, "elseif": true,
	"catch": true, "declare": true, "match": true, "isset": true, "empty": true, "unset": true,
	"eval": true, "exit": true, "die": true,
}

// slvupSkippedCasts are the casts after which "(" is never checked.
var slvupSkippedCasts = map[string]bool{
	"int": true, "integer": true, "double": true, "float": true, "real": true, "string": true,
	"array": true, "object": true, "bool": true, "boolean": true, "unset": true,
}

func (r SlevomatUselessParentheses) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		kinds := slvcStringKinds(f)
		f.eachToken(func(i int) bool { return f.is(i, "(") }, func(open int) int {
			closeIdx := f.toks[open].closer
			if closeIdx < 0 || f.parenHasOwner(open) {
				return 0
			}
			before := f.prevEffective(open - 1)
			if f.parenSkippedBefore(before) {
				return 0
			}
			after := f.nextEffective(open+1, len(f.toks))
			if f.parenIsKeyword(after, "clone", "yield", "yield from", "require", "require_once", "include", "include_once") ||
				f.parenCast(after) == "array" {
				return 0
			}
			for j := open + 1; j < closeIdx; j++ {
				if f.is(j, "=") {
					return 0
				}
			}
			if f.is(f.nextEffective(closeIdx+1, len(f.toks)), "(") {
				return 0
			}
			if f.identStart(before) >= 0 {
				return 0
			}
			r.parenTernaryCondition(f, open)
			r.parenCaseInSwitch(f, open)
			r.parenVariableOrCall(f, open, kinds)
			r.parenString(f, open, kinds)
			r.parenOperators(f, open)
			r.parenNew(f, open)
			return 0
		})
	})
}

func (r SlevomatUselessParentheses) parenTernaryCondition(f *slvFile, open int) {
	closeIdx := f.toks[open].closer
	if !f.parenIsInlineThen(f.nextEffective(closeIdx+1, len(f.toks))) {
		return
	}
	for j := open + 1; j < closeIdx; j++ {
		if f.parenIsKeyword(j, "and", "or", "xor") {
			return
		}
	}
	before := f.prevEffective(open - 1)
	if f.is(before, "!") || f.parenIsComparison(before) || f.parenIsBoolean(before) {
		return
	}
	if r.ignoreComplexTernary {
		for j := open + 1; j < closeIdx; j++ {
			if f.parenIsBoolean(j) || f.toks[j].kind == token.Whitespace && f.toks[j].content == f.eol {
				return
			}
		}
	}
	start := f.nextEffective(open+1, len(f.toks))
	end := f.prevEffective(closeIdx - 1)
	for j := start; j <= end; j++ {
		if f.parenIsInlineThen(j) {
			return
		}
	}
	f.parenRemove(open, start, end)
}

func (SlevomatUselessParentheses) parenCaseInSwitch(f *slvFile, open int) {
	closeIdx := f.toks[open].closer
	if !f.isKeyword(f.prevEffective(open-1), "case") || !f.is(f.nextEffective(closeIdx+1, len(f.toks)), ":") {
		return
	}
	f.parenRemove(open, f.nextEffective(open+1, len(f.toks)), f.prevEffective(closeIdx-1))
}

func (SlevomatUselessParentheses) parenVariableOrCall(f *slvFile, open int, kinds []uint8) {
	closeIdx := f.toks[open].closer
	n := len(f.toks)
	start := f.nextEffective(open+1, n)
	if start < 0 || f.isKeyword(start, "new") || f.is(start, "(") {
		return
	}
	for j := open + 1; j < closeIdx; j++ {
		if f.parenIsOperator(j) {
			return
		}
	}
	before := f.prevEffective(open - 1)
	if f.isKeyword(before, "case") || f.parenIsBoolean(before) {
		return
	}
	after := f.nextEffective(closeIdx+1, n)
	if f.parenIsInlineThen(after) || f.is(after, "(") || f.is(after, ">>") {
		return
	}
	if f.parenIsConstantString(start, kinds) {
		return
	}
	operand := start
	if f.is(start, "!") {
		operand = f.nextEffective(start+1, n)
	}
	var end int
	if f.isClassKeyword(operand) || f.isVariable(operand) || f.isDollar(operand) || f.isName(operand) {
		end = f.identEnd(operand)
		if end < 0 && f.isName(operand) && f.is(f.nextEffective(start+1, n), "(") {
			end = start
		}
		if end < 0 {
			return
		}
		for {
			next := f.nextEffective(end+1, n)
			if !f.is(next, "(") || f.toks[next].closer < 0 {
				break
			}
			end = f.toks[next].closer
		}
	} else {
		next := -1
		for j := operand + 1; j < n; j++ {
			if f.is(j, "(") {
				next = j
				break
			}
		}
		if next < 0 || f.toks[next].closer < 0 {
			return
		}
		end = f.toks[next].closer
	}
	if f.nextEffective(end+1, n) != closeIdx {
		return
	}
	f.parenRemove(open, start, end)
}

func (SlevomatUselessParentheses) parenString(f *slvFile, open int, kinds []uint8) {
	closeIdx := f.toks[open].closer
	str := f.nextEffective(open+1, len(f.toks))
	if !f.parenIsConstantString(str, kinds) || f.nextEffective(str+1, len(f.toks)) != closeIdx {
		return
	}
	f.parenRemove(open, str, str)
}

func (SlevomatUselessParentheses) parenOperators(f *slvFile, open int) {
	closeIdx := f.toks[open].closer
	n := len(f.toks)
	if f.isKeyword(f.nextEffective(open+1, n), "new") {
		return
	}
	before := f.prevEffective(open - 1)
	after := f.nextEffective(closeIdx+1, n)
	if f.is(before, "-") {
		if p := f.prevEffective(before - 1); p < 0 || f.toks[p].kind != token.Number {
			return
		}
	}
	if f.parenIsBoolean(before) || f.parenIsBoolean(after) || f.is(before, "!") {
		return
	}
	var operators []int
	for j := open + 1; j < closeIdx; j++ {
		switch {
		case f.parenIsInlineThen(j), f.is(j, "??"), f.is(j, "&"), f.is(j, "|"), f.is(j, "^"),
			f.is(j, "<<"), f.is(j, ">>"), f.parenIsComparison(j):
			return
		case f.is(j, "(") && f.toks[j].closer >= 0:
			j = f.toks[j].closer
		case f.parenIsOperator(j):
			operators = append(operators, j)
		}
	}
	if len(operators) == 0 {
		return
	}
	if (!f.is(before, "=") || !f.is(after, ";")) && len(operators) > 1 {
		return
	}
	group := func(i int) int { return slvupOperatorGroups[f.toks[i].orig] }
	first, last := operators[0], operators[len(operators)-1]
	if f.parenIsOperator(before) {
		if group(first) != group(before) {
			return
		}
		if f.is(before, "-") && (f.is(first, "+") || f.is(first, "-")) {
			return
		}
		if f.is(before, "/") && (f.is(first, "/") || f.is(first, "*")) {
			return
		}
		if f.is(before, "%") && f.is(first, "%") {
			return
		}
	}
	if f.parenIsOperator(after) && group(last) != group(after) {
		return
	}
	f.parenRemove(open, f.nextEffective(open+1, n), f.prevEffective(closeIdx-1))
}

func (r SlevomatUselessParentheses) parenNew(f *slvFile, open int) {
	closeIdx := f.toks[open].closer
	n := len(f.toks)
	if !f.isKeyword(f.nextEffective(open+1, n), "new") {
		return
	}
	after := f.nextEffective(closeIdx+1, n)
	closesShortArray := f.is(after, "]") && f.toks[after].closer >= 0 && f.isShortArrayOpen(f.toks[after].closer)
	if !f.is(after, ",") && !f.is(after, ";") && !closesShortArray {
		if !f.is(after, "->") && !f.is(after, "?->") || !r.checkAroundNew {
			return
		}
		if !f.is(f.prevEffective(closeIdx-1), ")") {
			return
		}
	}
	f.parenRemove(open, f.nextEffective(open+1, n), f.prevEffective(closeIdx-1))
}

func (f *slvFile) parenRemove(open, start, end int) {
	closeIdx := f.toks[open].closer
	f.beginChangeset()
	f.removeBetweenIncluding(open, start-1)
	f.removeBetweenIncluding(end+1, closeIdx)
	f.endChangeset()
}

// parenHasOwner tells a "(" PHP_CodeSniffer gives a parenthesis_owner.
func (f *slvFile) parenHasOwner(open int) bool {
	p := f.prevEffective(open - 1)
	if f.is(p, "&") {
		q := f.prevEffective(p - 1)
		return f.isKeyword(q, "function") || f.isKeyword(q, "fn")
	}
	if p >= 0 && f.toks[p].kind == token.Ident {
		q := f.prevEffective(p - 1)
		if f.is(q, "&") {
			q = f.prevEffective(q - 1)
		}
		if f.isKeyword(q, "function") {
			return true
		}
		if f.is(q, "->") || f.is(q, "?->") || f.is(q, "::") {
			return false
		}
	}
	return p >= 0 && (f.toks[p].kind == token.Keyword || f.toks[p].kind == token.Ident) &&
		slvupOwners[strings.ToLower(f.toks[p].orig)]
}

func (f *slvFile) parenSkippedBefore(p int) bool {
	if p < 0 {
		return false
	}
	t := f.toks[p]
	switch {
	case t.kind == token.Ident, f.isVariable(p), f.is(p, ")"), f.is(p, "~"):
		return true
	case t.kind == token.Keyword:
		return slvupSkippedBefore[strings.ToLower(t.orig)]
	}
	return slvupSkippedCasts[f.parenCast(p)]
}

// parenCast returns the lowercased type of a cast token, or "".
func (f *slvFile) parenCast(i int) string {
	if i < 0 || i >= len(f.toks) || f.toks[i].kind != token.Punct {
		return ""
	}
	v := f.toks[i].orig
	if len(v) < 3 || v[0] != '(' || v[len(v)-1] != ')' {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(v[1 : len(v)-1]))
}

func (f *slvFile) parenIsKeyword(i int, words ...string) bool {
	if i < 0 || i >= len(f.toks) || f.toks[i].kind != token.Keyword {
		return false
	}
	v := strings.ToLower(strings.Join(strings.Fields(f.toks[i].orig), " "))
	return slices.Contains(words, v)
}

func (f *slvFile) parenIsComparison(i int) bool {
	return i >= 0 && f.toks[i].kind == token.Punct && slvupComparison[f.toks[i].orig]
}

// parenIsBoolean tells PHP_CodeSniffer's Tokens::BOOLEAN_OPERATORS.
func (f *slvFile) parenIsBoolean(i int) bool {
	return f.is(i, "&&") || f.is(i, "||") || f.parenIsKeyword(i, "and", "or", "xor")
}

func (f *slvFile) parenIsOperator(i int) bool {
	if i < 0 || f.toks[i].kind != token.Punct {
		return false
	}
	_, ok := slvupOperatorGroups[f.toks[i].orig]
	return ok
}

// parenIsInlineThen tells a ternary "?" from a nullable type's.
func (f *slvFile) parenIsInlineThen(i int) bool {
	if !f.is(i, "?") {
		return false
	}
	p := f.prevEffective(i - 1)
	if p < 0 || f.is(p, "(") || f.is(p, ",") || f.is(p, ":") || f.toks[p].kind == token.AttributeEnd {
		return false
	}
	return !f.parenIsKeyword(p, "public", "protected", "private", "readonly", "static", "var", "const", "final", "abstract")
}

// parenIsConstantString tells a T_CONSTANT_ENCAPSED_STRING.
func (f *slvFile) parenIsConstantString(i int, kinds []uint8) bool {
	return i >= 0 && f.toks[i].kind == token.String && !f.toks[i].heredoc && kinds[i] != slvcDoubleQuoted
}
