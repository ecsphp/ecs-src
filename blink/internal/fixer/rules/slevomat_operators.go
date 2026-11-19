package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Operators/DisallowEqualOperatorsSniff.php
//
// SlevomatDisallowEqualOperators turns "==" into "===" and "!=", "<>" into "!==".
type SlevomatDisallowEqualOperators struct{}

func (SlevomatDisallowEqualOperators) Name() string {
	return `SlevomatCodingStandard\Sniffs\Operators\DisallowEqualOperatorsSniff`
}

func (SlevomatDisallowEqualOperators) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowEqualOperators{}.Name())
}

func (SlevomatDisallowEqualOperators) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool {
			return f.is(i, "==") || f.is(i, "!=") || f.is(i, "<>")
		}, func(i int) int {
			f.beginChangeset()
			if f.is(i, "==") {
				f.replace(i, "===")
			} else {
				f.replace(i, "!==")
			}
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Operators/NegationOperatorSpacingSniff.php
//
// SlevomatNegationOperatorSpacing sets the spaces after a unary minus
// (spacesCount, 0 by default).
type SlevomatNegationOperatorSpacing struct {
	spaces int
}

func (SlevomatNegationOperatorSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Operators\NegationOperatorSpacingSniff`
}

func (SlevomatNegationOperatorSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatNegationOperatorSpacing{}.Name())
}

func (r SlevomatNegationOperatorSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.spaces = configInt(config, "spacesCount", r.spaces)
	return r
}

func (r SlevomatNegationOperatorSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		kinds := slvcStringKinds(f)
		f.eachToken(func(i int) bool { return f.is(i, "-") }, func(i int) int {
			prev := f.prevEffective(i - 1)
			if prev < 0 || f.isMinusOperand(prev, kinds) || f.identStart(prev) >= 0 {
				return 0
			}
			ws := i + 1
			spaces := 0
			if ws < len(f.toks) && f.toks[ws].kind == token.Whitespace {
				spaces = len(f.toks[ws].content)
			}
			switch {
			case spaces == r.spaces:
			case r.spaces > spaces:
				f.add(i, " ")
			default:
				f.replace(ws, "")
			}
			return 0
		})
	})
}

// slvMagicOperands are the magic constants NegationOperatorSpacing counts as
// an operand before "-".
var slvMagicOperands = map[string]bool{
	"__class__": true, "__dir__": true, "__file__": true, "__function__": true,
	"__line__": true, "__method__": true, "__namespace__": true, "__trait__": true,
}

// isMinusOperand tells a token after which "-" is a binary minus.
func (f *slvFile) isMinusOperand(i int, kinds []uint8) bool {
	t := f.toks[i]
	switch {
	case f.isName(i), f.isVariable(i), t.kind == token.Number,
		f.is(i, ")"), f.is(i, "]"):
		return true
	case t.kind == token.Ident:
		return slvMagicOperands[strings.ToLower(t.orig)]
	case t.kind == token.String:
		return !t.heredoc && kinds[i] != slvcDoubleQuoted
	}
	return false
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Operators/RequireCombinedAssignmentOperatorSniff.php
//
// SlevomatRequireCombinedAssignmentOperator turns "$a = $a + 1" into "$a += 1".
type SlevomatRequireCombinedAssignmentOperator struct{}

func (SlevomatRequireCombinedAssignmentOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\Operators\RequireCombinedAssignmentOperatorSniff`
}

func (SlevomatRequireCombinedAssignmentOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireCombinedAssignmentOperator{}.Name())
}

var slvCombinedOperators = map[string]string{
	"&": "&=", "|": "|=", ".": ".=", "/": "/=", "-": "-=", "**": "**=",
	"%": "%=", "*": "*=", "+": "+=", "<<": "<<=", ">>": ">>=", "^": "^=",
}

// slvOperators is PHP_CodeSniffer's Tokens::OPERATORS.
var slvOperators = map[string]bool{
	"+": true, "-": true, "*": true, "/": true, "%": true, "**": true,
	"<=>": true, "??": true, "&": true, "|": true, "^": true, "<<": true, ">>": true,
}

func (SlevomatRequireCombinedAssignmentOperator) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		n := len(f.toks)
		f.eachToken(func(i int) bool { return f.is(i, "=") }, func(eq int) int {
			start := f.nextEffective(eq+1, n)
			end := f.identEnd(start)
			if end < 0 {
				return 0
			}
			op := f.nextEffective(end+1, n)
			if op < 0 || f.toks[op].kind != token.Punct {
				return 0
			}
			combined, ok := slvCombinedOperators[f.toks[op].orig]
			if !ok {
				return 0
			}
			fixable := true
			if f.is(end, "]") {
				after := f.nextEffective(op+1, n)
				if after >= 0 && f.toks[after].kind == token.String {
					return 0
				}
				fixable = after >= 0 && f.toks[after].kind == token.Number
			}
			beforeEnd := f.prevEffective(eq - 1)
			beforeStart := f.identStart(beforeEnd)
			if beforeStart < 0 || f.identContent(beforeStart, beforeEnd) != f.identContent(start, end) {
				return 0
			}
			semicolon := n
			for j := eq + 1; j < n; j++ {
				if f.is(j, ";") {
					semicolon = j
					break
				}
			}
			for j := op + 1; j < semicolon; j++ {
				if f.toks[j].kind == token.Punct && slvOperators[f.toks[j].orig] {
					return 0
				}
			}
			if !fixable {
				return 0
			}
			f.beginChangeset()
			f.change(eq, op, combined)
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Operators/SpreadOperatorSpacingSniff.php
//
// SlevomatSpreadOperatorSpacing sets the spaces after "..."
// (spacesCountAfterOperator, 0 by default).
type SlevomatSpreadOperatorSpacing struct {
	spaces int
}

func (SlevomatSpreadOperatorSpacing) Name() string {
	return `SlevomatCodingStandard\Sniffs\Operators\SpreadOperatorSpacingSniff`
}

func (SlevomatSpreadOperatorSpacing) SourceURL() string {
	return fixer.SourceURLFor(SlevomatSpreadOperatorSpacing{}.Name())
}

func (r SlevomatSpreadOperatorSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.spaces = configInt(config, "spacesCountAfterOperator", r.spaces)
	return r
}

func (r SlevomatSpreadOperatorSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.is(i, "...") }, func(i int) int {
			after := f.nextNonWhitespace(i+1, len(f.toks))
			if after < 0 || len(f.content(i+1, after-1)) == r.spaces {
				return 0
			}
			f.beginChangeset()
			f.add(i, strings.Repeat(" ", r.spaces))
			for j := i + 1; j < after; j++ {
				f.replace(j, "")
			}
			f.endChangeset()
			return 0
		})
	})
}

// isDollar tells T_DOLLAR, the "$" of "$$a" or "${'a'}".
func (f *slvFile) isDollar(i int) bool {
	return i >= 0 && i < len(f.toks) && f.toks[i].kind == token.Variable && f.toks[i].orig == "$"
}

func (f *slvFile) isVariable(i int) bool {
	return i >= 0 && i < len(f.toks) && f.toks[i].kind == token.Variable && f.toks[i].orig != "$"
}

// isName tells a TokenHelper::NAME_TOKEN_CODES token.
func (f *slvFile) isName(i int) bool {
	return i >= 0 && i < len(f.toks) && f.lineCallIsName(i)
}

// isClassKeyword tells a TokenHelper::CLASS_KEYWORD_CODES token.
func (f *slvFile) isClassKeyword(i int) bool {
	if i < 0 || f.isName(i) {
		return false
	}
	if f.isKeyword(i, "static") {
		return true
	}
	v := strings.ToLower(f.toks[i].orig)
	return f.toks[i].kind == token.Ident && (v == "self" || v == "parent")
}

func (f *slvFile) isMemberOperator(i int) bool {
	return f.is(i, "->") || f.is(i, "?->") || f.is(i, "::")
}

// isSquareClose tells a T_CLOSE_SQUARE_BRACKET from a short array's "]".
func (f *slvFile) isSquareClose(i int) bool {
	return f.is(i, "]") && f.toks[i].closer >= 0 && !f.isShortArrayOpen(f.toks[i].closer)
}

func (f *slvFile) isSquareOpen(i int) bool {
	return f.is(i, "[") && !f.isShortArrayOpen(i)
}

// identContent is IdentificatorHelper::getContent.
func (f *slvFile) identContent(start, end int) string {
	var b strings.Builder
	for i := start; i <= end; i++ {
		if !f.isEmpty(i) {
			b.WriteString(f.toks[i].content)
		}
	}
	return b.String()
}

// identStart is IdentificatorHelper::findStartPointer, or -1.
func (f *slvFile) identStart(end int) int {
	if end < 0 {
		return -1
	}
	if f.isName(end) {
		prev := f.prevEffective(end - 1)
		if f.isMemberOperator(prev) && !f.is(f.prevEffective(prev-1), ")") {
			return f.identStartBeforeOperator(prev)
		}
		return end
	}
	if (f.is(end, "}") || f.isSquareClose(end)) && f.toks[end].closer >= 0 {
		return f.identStartBeforePart(f.toks[end].closer)
	}
	if f.isVariable(end) {
		return f.identStartBeforePart(end)
	}
	return -1
}

func (f *slvFile) identStartBeforeOperator(op int) int {
	prev := f.prevEffective(op - 1)
	for prev > 0 && f.isName(prev) && f.isName(prev-1) {
		prev--
	}
	if f.is(op, "::") && (f.isClassKeyword(prev) || f.isName(prev)) {
		return prev
	}
	if f.isName(prev) {
		if p := f.prevEffective(prev - 1); f.is(p, "->") || f.is(p, "?->") {
			return f.identStartBeforeOperator(p)
		}
	}
	if (f.is(prev, "}") || f.isSquareClose(prev)) && f.toks[prev].closer >= 0 {
		return f.identStartBeforePart(f.toks[prev].closer)
	}
	return f.identStartBeforePart(prev)
}

func (f *slvFile) identStartBeforePart(part int) int {
	prev := f.prevEffective(part - 1)
	if f.isDollar(prev) {
		prev = f.prevEffective(prev - 1)
	}
	switch {
	case prev < 0:
	case f.isMemberOperator(prev):
		return f.identStartBeforeOperator(prev)
	case f.isSquareClose(prev):
		return f.identStartBeforePart(f.toks[prev].closer)
	case f.is(prev, "}") && f.toks[prev].closer >= 0 && f.braceOwner(f.toks[prev].closer) < 0:
		return f.identStartBeforePart(f.toks[prev].closer)
	case f.isVariable(prev) || f.isName(prev):
		return f.identStartBeforePart(prev)
	}
	return part
}

// identEnd is IdentificatorHelper::findEndPointer, or -1.
func (f *slvFile) identEnd(start int) int {
	if start < 0 {
		return -1
	}
	if f.isName(start) {
		for start+1 < len(f.toks) && f.isName(start+1) {
			start++
		}
	} else if f.isDollar(start) {
		if start = f.nextEffective(start+1, len(f.toks)); start < 0 {
			return -1
		}
	}
	next := f.nextEffective(start+1, len(f.toks))
	if (f.isClassKeyword(start) || f.isName(start)) && f.is(next, "::") {
		return f.identEndAfterOperator(next)
	}
	if !f.isVariable(start) {
		return -1
	}
	if f.isMemberOperator(next) {
		return f.identEndAfterOperator(next)
	}
	if f.isSquareOpen(next) {
		return f.identEndAfterPart(start)
	}
	return start
}

func (f *slvFile) identEndAfterOperator(op int) int {
	next := f.nextEffective(op+1, len(f.toks))
	if f.isDollar(next) {
		next = f.nextEffective(next+1, len(f.toks))
	}
	if f.is(next, "{") && f.toks[next].closer >= 0 {
		return f.identEndAfterPart(f.toks[next].closer)
	}
	return f.identEndAfterPart(next)
}

func (f *slvFile) identEndAfterPart(part int) int {
	if part < 0 {
		return part
	}
	next := f.nextEffective(part+1, len(f.toks))
	if f.isMemberOperator(next) {
		return f.identEndAfterOperator(next)
	}
	if f.isSquareOpen(next) && f.toks[next].closer >= 0 {
		return f.identEndAfterPart(f.toks[next].closer)
	}
	return part
}
