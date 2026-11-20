package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// slvlcIndent is IndentationHelper::getOneIndentationLevel with ECS's tabWidth of 4.
const slvlcIndent = "    "

// slvlcConditions is AbstractLineCondition's checkedControlStructures.
type slvlcConditions struct {
	checked []string // nil means the default: if, while, do
}

func (c slvlcConditions) checks(name string) bool {
	if c.checked == nil {
		return true
	}
	return slices.Contains(c.checked, name)
}

func (c slvlcConditions) withConfig(config map[string]any) slvlcConditions {
	if v, ok := config["checkedControlStructures"].([]any); ok {
		c.checked = []string{}
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				c.checked = append(c.checked, strings.TrimSpace(s))
			}
		}
	}
	return c
}

// registered is AbstractLineCondition::register.
func (c slvlcConditions) registered(f *slvFile, i int) bool {
	if f.toks[i].kind != token.Keyword {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "if", "elseif":
		return c.checks("if")
	case "while":
		return c.checks("while") || c.checks("do")
	}
	return false
}

// parens returns the parentheses of the control structure at i, or ok false
// when it has none (AbstractLineCondition::shouldBeSkipped's first check).
func (c slvlcConditions) parens(f *slvFile, sc *phpcsScope, i int) (open, closeIdx int, ok bool) {
	closeIdx = sc.pCloser[i]
	if closeIdx < 0 || f.toks[closeIdx].closer < 0 {
		return 0, 0, false
	}
	return f.toks[closeIdx].closer, closeIdx, true
}

// skipped is AbstractLineCondition::shouldBeSkipped.
func (c slvlcConditions) skipped(f *slvFile, sc *phpcsScope, i int) bool {
	_, closeIdx, ok := c.parens(f, sc, i)
	if !ok {
		return true
	}
	if !f.isKeyword(i, "while") {
		return false
	}
	partOfDo := !f.is(f.nextEffective(closeIdx+1, len(f.toks)), "{")
	if partOfDo {
		return !c.checks("do")
	}
	return !c.checks("while")
}

// lineStart is AbstractLineCondition::getLineStart.
func (f *slvFile) slvlcLineStart(i int) string { return f.content(f.firstOnLine(i), i) }

// lineEnd is AbstractLineCondition::getLineEnd.
func (f *slvFile) slvlcLineEnd(i int) string {
	return strings.TrimRight(f.content(i, f.lastOnLine(i)), phpTrimChars)
}

// condition is AbstractLineCondition::getCondition.
func (f *slvFile) slvlcCondition(open, closeIdx int) string {
	cond := f.content(open+1, closeIdx-1)
	var b strings.Builder
	for {
		n := strings.Index(cond, f.eol)
		if n < 0 {
			b.WriteString(cond)
			break
		}
		b.WriteString(cond[:n])
		b.WriteByte(' ')
		cond = strings.TrimLeft(cond[n+len(f.eol):], " \t")
	}
	return strings.Trim(b.String(), phpTrimChars)
}

// isBoolean tells PHP_CodeSniffer's Tokens::BOOLEAN_OPERATORS.
func (f *slvFile) slvlcIsBoolean(i int) bool {
	return f.is(i, "&&") || f.is(i, "||") || f.isKeyword(i, "and") || f.isKeyword(i, "or") || f.isKeyword(i, "xor")
}

// slvlcIsSpace matches PCRE's \s.
func slvlcIsSpace(v string) bool {
	if v == "" {
		return false
	}
	for i := range len(v) {
		switch v[i] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
		default:
			return false
		}
	}
	return true
}

// removeWhitespaceBefore is FixerHelper::removeWhitespaceBefore.
func (f *slvFile) slvlcRemoveWhitespaceBefore(p int) {
	for i := p - 1; i > 0 && slvlcIsSpace(f.tokenContent(i)); i-- {
		f.replaceToken(i, "")
	}
}

// removeWhitespaceAfter is FixerHelper::removeWhitespaceAfter.
func (f *slvFile) slvlcRemoveWhitespaceAfter(p int) {
	for i := p + 1; i < len(f.toks) && slvlcIsSpace(f.tokenContent(i)); i++ {
		f.replace(i, "")
	}
}

// removeBetween is FixerHelper::removeBetween.
func (f *slvFile) slvlcRemoveBetween(start, end int) { f.removeBetweenIncluding(start+1, end-1) }

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireMultiLineConditionSniff.php
//
// SlevomatRequireMultiLineCondition splits a long if/elseif/while condition so
// each boolean part sits on its own line.
type SlevomatRequireMultiLineCondition struct {
	conds                         slvlcConditions
	minLineLength                 int
	booleanOperatorOnPreviousLine bool
	alwaysSplitAllConditionParts  bool
}

func (SlevomatRequireMultiLineCondition) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireMultiLineConditionSniff`
}

func (SlevomatRequireMultiLineCondition) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireMultiLineCondition{}.Name())
}

func (r SlevomatRequireMultiLineCondition) WithConfig(config map[string]any) fixer.Fixer {
	r.conds = r.conds.withConfig(config)
	r.minLineLength = configInt(config, "minLineLength", r.minLineLength)
	if v, ok := config["booleanOperatorOnPreviousLine"].(bool); ok {
		r.booleanOperatorOnPreviousLine = v
	}
	if v, ok := config["alwaysSplitAllConditionParts"].(bool); ok {
		r.alwaysSplitAllConditionParts = v
	}
	return r
}

func (r SlevomatRequireMultiLineCondition) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		f.eachToken(func(i int) bool { return r.conds.registered(f, i) }, func(i int) int {
			r.process(f, sc, i)
			return 0
		})
	})
}

func (r SlevomatRequireMultiLineCondition) process(f *slvFile, sc *phpcsScope, i int) {
	if r.conds.skipped(f, sc, i) {
		return
	}
	open, closeIdx, _ := r.conds.parens(f, sc, i)
	booleans := 0
	for j := open + 1; j < closeIdx; j++ {
		if f.slvlcIsBoolean(j) {
			booleans++
		}
	}
	if booleans == 0 {
		return
	}
	start := f.nextEffective(open+1, len(f.toks))
	end := f.prevEffective(closeIdx - 1)
	startsOnNewLine := f.toks[open].line != f.toks[start].line
	endsOnNewLine := f.toks[closeIdx].line != f.toks[end].line
	lineStartAt, lineEndAt := open, closeIdx
	if startsOnNewLine {
		lineStartAt = start - 1
	}
	if endsOnNewLine {
		lineEndAt = end + 1
	}
	lineLength := len(f.slvlcLineStart(lineStartAt) + f.slvlcCondition(open, closeIdx) + f.slvlcLineEnd(lineEndAt))
	lines := f.toks[end].line - f.toks[start].line + 1
	if !r.shouldReport(lineLength, lines, booleans) {
		return
	}

	indentAt := f.firstNonWhitespaceOnLine(open)
	if startsOnNewLine {
		indentAt = start
	}
	structureIndent := f.indentation(indentAt)
	condIndent := structureIndent
	if !startsOnNewLine {
		condIndent += slvlcIndent
	}
	level := 0
	f.beginChangeset()
	if !startsOnNewLine {
		f.slvlcRemoveWhitespaceBefore(start)
		f.addBefore(start, f.eol+condIndent)
	}
	for j := start; j <= end; j++ {
		if f.is(j, "(") {
			level++
			inner := f.toks[j].closer
			hasBoolean := false
			for k := j + 1; k < inner; k++ {
				if f.slvlcIsBoolean(k) {
					hasBoolean = true
					break
				}
			}
			if hasBoolean {
				f.slvlcRemoveWhitespaceAfter(j)
				f.add(j, f.eol+condIndent+strings.Repeat(slvlcIndent, level))
				f.slvlcRemoveWhitespaceBefore(inner)
				f.addBefore(inner, f.eol+condIndent+strings.Repeat(slvlcIndent, level-1))
			}
			continue
		}
		if f.is(j, ")") {
			level--
			continue
		}
		if !f.slvlcIsBoolean(j) {
			continue
		}
		innerIndent := condIndent
		if level > 0 {
			innerIndent += strings.Repeat(slvlcIndent, level)
		}
		if r.booleanOperatorOnPreviousLine {
			f.add(j, f.eol+innerIndent)
			f.slvlcRemoveWhitespaceAfter(j)
			continue
		}
		f.slvlcRemoveWhitespaceBefore(j)
		f.addBefore(j, f.eol+innerIndent)
	}
	if !endsOnNewLine {
		f.slvlcRemoveWhitespaceAfter(end)
		f.add(end, f.eol+structureIndent)
	}
	f.endChangeset()
}

func (r SlevomatRequireMultiLineCondition) shouldReport(lineLength, lines, booleans int) bool {
	if lines == 1 {
		return r.minLineLength == 0 || lineLength >= r.minLineLength
	}
	return r.alwaysSplitAllConditionParts && lines < booleans+1
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireSingleLineConditionSniff.php
//
// SlevomatRequireSingleLineCondition joins a multi-line if/elseif/while
// condition onto one line when it fits.
type SlevomatRequireSingleLineCondition struct {
	conds                     slvlcConditions
	maxLineLength             int
	alwaysForSimpleConditions bool
}

func (SlevomatRequireSingleLineCondition) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireSingleLineConditionSniff`
}

func (SlevomatRequireSingleLineCondition) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireSingleLineCondition{}.Name())
}

func (r SlevomatRequireSingleLineCondition) WithConfig(config map[string]any) fixer.Fixer {
	r.conds = r.conds.withConfig(config)
	r.maxLineLength = configInt(config, "maxLineLength", r.maxLineLength)
	if v, ok := config["alwaysForSimpleConditions"].(bool); ok {
		r.alwaysForSimpleConditions = v
	}
	return r
}

func (r SlevomatRequireSingleLineCondition) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		f.eachToken(func(i int) bool { return r.conds.registered(f, i) }, func(i int) int {
			r.process(f, sc, i)
			return 0
		})
	})
}

func (r SlevomatRequireSingleLineCondition) process(f *slvFile, sc *phpcsScope, i int) {
	if r.conds.skipped(f, sc, i) {
		return
	}
	open, closeIdx, _ := r.conds.parens(f, sc, i)
	if f.toks[open].line == f.toks[closeIdx].line {
		return
	}
	simple := true
	for j := open + 1; j < closeIdx; j++ {
		if f.toks[j].kind == token.Comment {
			return
		}
		if f.slvlcIsBoolean(j) {
			simple = false
		}
	}
	cond := f.slvlcCondition(open, closeIdx)
	lineLength := len(f.slvlcLineStart(open) + cond + f.slvlcLineEnd(closeIdx))
	if !r.shouldReport(lineLength, simple) {
		return
	}
	f.beginChangeset()
	f.add(open, cond)
	f.slvlcRemoveBetween(open, closeIdx)
	f.endChangeset()
}

func (r SlevomatRequireSingleLineCondition) shouldReport(lineLength int, simple bool) bool {
	if r.maxLineLength == 0 || lineLength <= r.maxLineLength {
		return true
	}
	return simple && r.alwaysForSimpleConditions
}

// isInlineThen tells PHP_CodeSniffer's T_INLINE_THEN.
func (f *slvFile) slvlcIsInlineThen(sc *phpcsScope, i int) bool {
	return f.is(i, "?") && sc.isInlineThen(i)
}

// scopeLevel is ScopeHelper::isInSameScope's scope of i: one more than the
// level of the innermost function holding it, else 0.
func slvlcScopeLevel(sc *phpcsScope, i int) int {
	conds := sc.conditions(i)
	for _, c := range slices.Backward(conds) {
		if sc.code[c] == "function" {
			return len(sc.conditions(c)) + 1
		}
	}
	return 0
}

func slvlcSameScope(sc *phpcsScope, a, b int) bool {
	return slvlcScopeLevel(sc, a) == slvlcScopeLevel(sc, b)
}

// elsePointer is TernaryOperatorHelper::getElsePointer, or -1.
func (f *slvFile) slvlcElsePointer(sc *phpcsScope, then int) int {
	for p := then + 1; p < len(f.toks); p++ {
		switch {
		case (f.is(p, "(") || f.is(p, "[")) && f.toks[p].closer > p:
			p = f.toks[p].closer
		case sc.code[p] == "inline_else" && slvlcSameScope(sc, then, p):
			return p
		}
	}
	return -1
}

// endPointer is TernaryOperatorHelper::getEndPointer.
func (f *slvFile) slvlcEndPointer(sc *phpcsScope, then, els int) int {
	for p := els + 1; p < len(f.toks); p++ {
		switch {
		case f.is(p, ")"):
			if o := f.toks[p].closer; o >= 0 && o < then {
				return f.prevEffective(p - 1)
			}
		case f.is(p, "]"):
			if o := f.toks[p].closer; o >= 0 && o < then {
				return f.prevEffective(p - 1)
			}
		case f.is(p, ","):
			prev := -1
			for k := p - 1; k >= then; k-- {
				if f.is(k, "(") || f.isShortArrayOpen(k) {
					prev = k
					break
				}
			}
			if prev < 0 || f.toks[prev].closer < p {
				return f.prevEffective(p - 1)
			}
		case f.is(p, "=>") && sc.code[p] == "=>":
			prev := -1
			for k := p - 1; k >= then; k-- {
				if f.isShortArrayOpen(k) {
					prev = k
					break
				}
			}
			if prev < 0 {
				return f.prevEffective(p - 1)
			}
		case f.is(p, ";") || f.is(p, "??") || f.is(p, ":") && sc.code[p] == ":":
			if slvlcSameScope(sc, els, p) {
				return f.prevEffective(p - 1)
			}
		}
	}
	return f.prevEffective(len(f.toks) - 1)
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/DisallowTrailingMultiLineTernaryOperatorSniff.php
//
// SlevomatDisallowTrailingMultiLineTernaryOperator moves the "?" and ":" of a
// multi-line ternary from the end of a line to the start of the next.
type SlevomatDisallowTrailingMultiLineTernaryOperator struct{}

func (SlevomatDisallowTrailingMultiLineTernaryOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\DisallowTrailingMultiLineTernaryOperatorSniff`
}

func (SlevomatDisallowTrailingMultiLineTernaryOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowTrailingMultiLineTernaryOperator{}.Name())
}

func (SlevomatDisallowTrailingMultiLineTernaryOperator) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		f.eachToken(func(i int) bool { return f.slvlcIsInlineThen(sc, i) }, func(then int) int {
			next := f.nextEffective(then+1, len(f.toks))
			if next < 0 || sc.code[next] == "inline_else" || f.toks[then].line == f.toks[next].line {
				return 0
			}
			els := f.slvlcElsePointer(sc, then)
			if els < 0 {
				return 0
			}
			beforeThen := f.prevEffective(then - 1)
			afterThen := f.nextNonWhitespace(then+1, len(f.toks))
			beforeElse := f.prevEffective(els - 1)
			afterElse := f.nextNonWhitespace(els+1, len(f.toks))
			indent := f.indentation(f.firstNonWhitespaceOnLine(then)) + slvlcIndent

			f.beginChangeset()
			f.slvlcRemoveBetween(beforeThen, then)
			f.slvlcRemoveBetween(then, afterThen)
			f.addBefore(then, f.eol+indent)
			f.addBefore(afterThen, " ")
			f.slvlcRemoveBetween(beforeElse, els)
			f.slvlcRemoveBetween(els, afterElse)
			f.addBefore(els, f.eol+indent)
			f.addBefore(afterElse, " ")
			f.endChangeset()
			return 0
		})
	})
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/ControlStructures/RequireMultiLineTernaryOperatorSniff.php
//
// SlevomatRequireMultiLineTernaryOperator breaks a ternary longer than
// lineLengthLimit onto lines led by "?" and ":".
type SlevomatRequireMultiLineTernaryOperator struct {
	lineLengthLimit      int
	minExpressionsLength *int
}

func (SlevomatRequireMultiLineTernaryOperator) Name() string {
	return `SlevomatCodingStandard\Sniffs\ControlStructures\RequireMultiLineTernaryOperatorSniff`
}

func (SlevomatRequireMultiLineTernaryOperator) SourceURL() string {
	return fixer.SourceURLFor(SlevomatRequireMultiLineTernaryOperator{}.Name())
}

func (r SlevomatRequireMultiLineTernaryOperator) WithConfig(config map[string]any) fixer.Fixer {
	r.lineLengthLimit = configInt(config, "lineLengthLimit", r.lineLengthLimit)
	if _, ok := config["minExpressionsLength"]; ok {
		if v := configInt(config, "minExpressionsLength", -1); v >= 0 {
			r.minExpressionsLength = &v
		} else {
			r.minExpressionsLength = nil
		}
	}
	return r
}

func (r SlevomatRequireMultiLineTernaryOperator) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		f.eachToken(func(i int) bool { return f.slvlcIsInlineThen(sc, i) }, func(then int) int {
			r.process(f, sc, then)
			return 0
		})
	})
}

func (r SlevomatRequireMultiLineTernaryOperator) process(f *slvFile, sc *phpcsScope, then int) {
	if f.is(f.prevEffective(then-1), "(") {
		return
	}
	if next := f.nextEffective(then+1, len(f.toks)); next < 0 || sc.code[next] == "inline_else" {
		return
	}
	els := f.slvlcElsePointer(sc, then)
	if els < 0 || f.toks[then].line != f.toks[els].line {
		return
	}
	elseEnd := f.slvlcEndPointer(sc, then, els)
	afterEnd := f.nextEffective(elseEnd+1, len(f.toks))
	if afterEnd < 0 || !f.is(afterEnd, ";") {
		return
	}
	eolBefore := f.slvlcEndOfLineBefore(then)
	if eolBefore < 0 || len(f.content(eolBefore+1, afterEnd)) <= r.lineLengthLimit {
		return
	}
	if r.minExpressionsLength != nil && *r.minExpressionsLength >= len(f.content(then+1, afterEnd-1)) {
		return
	}
	after := f.nextNonWhitespace(eolBefore+1, len(f.toks))
	indent := f.content(eolBefore+1, after-1) + slvlcIndent
	beforeThen := f.prevEffective(then - 1)
	beforeElse := f.prevEffective(els - 1)

	f.beginChangeset()
	f.slvlcRemoveBetween(beforeThen, then)
	f.addBefore(then, f.eol+indent)
	f.slvlcRemoveBetween(beforeElse, els)
	f.addBefore(els, f.eol+indent)
	f.endChangeset()
}

// endOfLineBefore is RequireMultiLineTernaryOperatorSniff::getEndOfLineBefore,
// or -1 where slevomat would not find one.
func (f *slvFile) slvlcEndOfLineBefore(p int) int {
	for i := p - 1; i >= 0; i-- {
		t := f.toks[i]
		switch t.kind {
		case token.Whitespace:
			if t.content == f.eol {
				return i
			}
		case token.Comment:
			if strings.HasSuffix(t.content, f.eol) {
				return i
			}
		}
	}
	return -1
}
