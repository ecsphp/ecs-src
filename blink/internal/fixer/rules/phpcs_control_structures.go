package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsControlParens returns the condition parentheses of a control keyword at
// i that one of kws names, or -1, -1.
func phpcsControlParens(f *slvFile, i int, kws ...string) (int, int) {
	t := f.toks[i]
	if t.kind != token.Keyword {
		return -1, -1
	}
	if !slices.Contains(kws, strings.ToLower(t.orig)) {
		return -1, -1
	}
	return f.icsParens(i)
}

func (f *slvFile) isPhpcsComment(i int) bool {
	return f.toks[i].kind == token.Comment || f.toks[i].kind == token.DocComment
}

// addBeforeRaw is Fixer::addContentBefore.
func (f *slvFile) addBeforeRaw(i int, v string) {
	f.replaceToken(i, v+f.tokenContent(i))
}

var psr2ControlKeywords = []string{"if", "while", "foreach", "for", "switch", "elseif", "catch", "match"}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/ControlStructures/ControlStructureSpacingSniff.php
//
// PSR2ControlStructureSpacing removes the spaces just inside the condition
// parentheses of a control structure: "if ( $a )" -> "if ($a)".
type PSR2ControlStructureSpacing struct {
	RequiredSpacesAfterOpen   int
	RequiredSpacesBeforeClose int
}

func (PSR2ControlStructureSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\PSR2\Sniffs\ControlStructures\ControlStructureSpacingSniff`
}

func (PSR2ControlStructureSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR2/Sniffs/ControlStructures/ControlStructureSpacingSniff.php"
}

func (r PSR2ControlStructureSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.RequiredSpacesAfterOpen = configInt(config, "requiredSpacesAfterOpen", r.RequiredSpacesAfterOpen)
	r.RequiredSpacesBeforeClose = configInt(config, "requiredSpacesBeforeClose", r.RequiredSpacesBeforeClose)
	return r
}

func (r PSR2ControlStructureSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if open, closeIdx := phpcsControlParens(f, i, psr2ControlKeywords...); open >= 0 {
				r.process(f, open, closeIdx)
			}
		}
	})
}

func (r PSR2ControlStructureSpacing) process(f *slvFile, open, closeIdx int) {
	if next := f.nextNonWhitespace(open+1, len(f.toks)); next >= 0 && !f.isPhpcsComment(next) {
		after, newline := 0, false
		if f.toks[open+1].kind == token.Whitespace {
			if strings.Contains(f.toks[open+1].content, f.eol) {
				newline = true
			} else {
				after = f.toks[open+1].length
			}
		}
		if newline || after != r.RequiredSpacesAfterOpen {
			padding := strings.Repeat(" ", r.RequiredSpacesAfterOpen)
			switch {
			case newline:
				f.replaceToken(open+1, "")
			case after == 0:
				f.icsAdd(open, padding)
			default:
				f.replaceToken(open+1, padding)
			}
		}
	}

	prev := closeIdx - 1
	for prev > open && f.toks[prev].kind == token.Whitespace {
		prev--
	}
	if f.toks[prev].line != f.toks[closeIdx].line {
		return
	}
	before := 0
	if f.toks[closeIdx-1].kind == token.Whitespace {
		before = len(strings.TrimLeft(f.toks[closeIdx-1].content, f.eol))
	}
	if before == r.RequiredSpacesBeforeClose {
		return
	}
	padding := strings.Repeat(" ", r.RequiredSpacesBeforeClose)
	if before == 0 {
		f.addBeforeRaw(closeIdx, padding)
	} else {
		f.replaceToken(closeIdx-1, padding)
	}
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/ControlStructures/ControlStructureSpacingSniff.php
//
// PSR12ControlStructureSpacing follows PSR2 for a single-line condition, and
// for a multi-line one puts the first expression on the line after "(",
// indents each line once and puts ")" on its own line at the keyword's indent.
type PSR12ControlStructureSpacing struct {
	Indent int
}

func (PSR12ControlStructureSpacing) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\ControlStructures\ControlStructureSpacingSniff`
}

func (PSR12ControlStructureSpacing) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/ControlStructures/ControlStructureSpacingSniff.php"
}

func (r PSR12ControlStructureSpacing) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (r PSR12ControlStructureSpacing) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if open, closeIdx := phpcsControlParens(f, i, psr2ControlKeywords...); open >= 0 {
				r.process(f, i, open, closeIdx)
			}
		}
	})
}

func (r PSR12ControlStructureSpacing) process(f *slvFile, ptr, open, closeIdx int) {
	toks := f.toks
	if toks[open].line == toks[closeIdx].line {
		PSR2ControlStructureSpacing{}.process(f, open, closeIdx)
		return
	}
	next := f.nextNonWhitespace(open+1, closeIdx)
	if next < 0 {
		return
	}
	if toks[next].line != toks[open].line+1 {
		f.beginChangeset()
		if toks[next].line > toks[open].line+1 {
			for i := open + 1; i < next && toks[next].line != toks[i].line; i++ {
				f.replaceToken(i, "")
			}
		}
		f.icsAdd(open, f.eol)
		f.endChangeset()
	}

	first := f.firstNonWhitespaceOnLine(ptr)
	required := toks[first].column + r.Indent - 1
	for i := open; i < closeIdx; i++ {
		if toks[i].column != 1 || toks[i+1].line > toks[i].line || f.isPhpcsComment(i) {
			continue
		}
		if i+1 == closeIdx {
			break
		}
		if toks[i].kind == token.String || toks[i].kind == token.InlineHTML {
			continue
		}
		found := 0
		if toks[i].kind == token.Whitespace {
			found = toks[i].length
		}
		if found < required {
			padding := strings.Repeat(" ", required)
			if found == 0 {
				f.addBeforeRaw(i, padding)
			} else {
				f.replaceToken(i, padding)
			}
		}
	}

	prev := closeIdx - 1
	for prev > open && toks[prev].kind == token.Whitespace {
		prev--
	}
	if toks[closeIdx].line != toks[prev].line+1 {
		if toks[closeIdx].line == toks[prev].line {
			f.addBeforeRaw(closeIdx, f.eol)
		} else {
			f.beginChangeset()
			for i := prev + 1; i < closeIdx; i++ {
				if toks[i].line == toks[prev].line {
					continue
				}
				if toks[i].line == toks[closeIdx].line {
					break
				}
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}
	}
	if toks[closeIdx].line != toks[prev].line {
		required := toks[first].column - 1
		found := toks[closeIdx].column - 1
		if found != required {
			padding := strings.Repeat(" ", required)
			if found == 0 {
				f.addBeforeRaw(closeIdx, padding)
			} else {
				f.replaceToken(closeIdx-1, padding)
			}
		}
	}
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/ControlStructures/BooleanOperatorPlacementSniff.php
//
// BooleanOperatorPlacement puts every "&&" and "||" of a multi-line condition
// at the start of its line, or every one at the end, following the first one
// found there unless AllowOnly ("first" or "last") says which.
type BooleanOperatorPlacement struct {
	AllowOnly string
}

func (BooleanOperatorPlacement) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\ControlStructures\BooleanOperatorPlacementSniff`
}

func (BooleanOperatorPlacement) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/ControlStructures/BooleanOperatorPlacementSniff.php"
}

func (r BooleanOperatorPlacement) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["allowOnly"].(string); ok {
		r.AllowOnly = v
	}
	return r
}

func (r BooleanOperatorPlacement) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if open, closeIdx := phpcsControlParens(f, i, "if", "while", "switch", "elseif", "match"); open >= 0 {
				r.process(f, open, closeIdx)
			}
		}
	})
}

func (r BooleanOperatorPlacement) process(f *slvFile, open, closeIdx int) {
	toks := f.toks
	if toks[open].line == toks[closeIdx].line {
		return
	}
	position := ""
	if r.AllowOnly == "first" || r.AllowOnly == "last" {
		position = r.AllowOnly
	}
	prevOf := func(op int) int {
		for p := op - 1; p >= open; p-- {
			if toks[p].kind != token.Whitespace {
				return p
			}
		}
		return -1
	}
	isError := false
	var operators []int
	for op := open + 1; op < closeIdx; op++ {
		if !f.is(op, "&&") && !f.is(op, "||") {
			continue
		}
		prev := prevOf(op)
		if prev < 0 {
			return
		}
		next := f.nextNonWhitespace(op+1, closeIdx)
		if next < 0 {
			return
		}
		firstOnLine := toks[prev].line < toks[op].line
		lastOnLine := toks[next].line > toks[op].line
		if firstOnLine && lastOnLine {
			continue
		}
		operators = append(operators, op)
		if !firstOnLine && !lastOnLine {
			continue
		}
		want := "last"
		if firstOnLine {
			want = "first"
		}
		if position == "" {
			position = want
		}
		if position != want {
			isError = true
		}
	}
	if !isError {
		return
	}

	f.beginChangeset()
	for _, op := range operators {
		prev := prevOf(op)
		next := f.nextNonWhitespace(op+1, closeIdx)
		if position == "last" {
			if toks[next].line != toks[op].line {
				continue
			}
			if toks[op+1].kind == token.Whitespace {
				f.replaceToken(op+1, "")
			}
			if toks[prev].line == toks[op].line {
				first := f.firstNonWhitespaceOnLine(op)
				f.icsAdd(op, f.eol+strings.Repeat(" ", toks[first].column-1))
			} else {
				f.icsAdd(prev, " "+toks[op].content)
				f.replaceToken(op, "")
			}
			continue
		}
		if toks[prev].line != toks[op].line {
			continue
		}
		if toks[op-1].kind == token.Whitespace {
			f.replaceToken(op-1, "")
		}
		if toks[next].line == toks[op].line {
			first := f.firstNonWhitespaceOnLine(op)
			f.addBeforeRaw(op, f.eol+strings.Repeat(" ", toks[first].column-1))
		} else {
			f.addBeforeRaw(next, toks[op].content+" ")
			f.replaceToken(op, "")
		}
	}
	f.endChangeset()
}

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/ControlStructures/MultiLineConditionSniff.php
//
// MultiLineCondition lays out an if/elseif condition PEAR style: no space
// after "(", each continuation line indented once and starting with a boolean
// operator, ")" on its own line and one space before "{".
type MultiLineCondition struct {
	Indent int
}

func (MultiLineCondition) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\ControlStructures\MultiLineConditionSniff`
}

func (MultiLineCondition) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/ControlStructures/MultiLineConditionSniff.php"
}

func (r MultiLineCondition) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	return r
}

func (r MultiLineCondition) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if open, closeIdx := phpcsControlParens(f, i, "if", "elseif"); open >= 0 {
				r.process(f, i, open, closeIdx)
			}
		}
	})
}

func (f *slvFile) isBooleanOperator(i int) bool {
	return f.is(i, "&&") || f.is(i, "||") || f.isKeyword(i, "and") || f.isKeyword(i, "or") || f.isKeyword(i, "xor")
}

func (r MultiLineCondition) process(f *slvFile, ptr, open, closeIdx int) {
	toks := f.toks
	n := len(toks)
	if toks[open+1].kind == token.Whitespace {
		f.replaceToken(open+1, "")
	}

	statementIndent := 0
	i := ptr - 1
	for ; i >= 0; i-- {
		if toks[i].line != toks[ptr].line {
			i++
			break
		}
	}
	if i >= 0 && toks[i].kind == token.Whitespace {
		statementIndent = toks[i].length
	}

	prevLine := toks[open].line
	for i := open + 1; i <= closeIdx; i++ {
		if i == closeIdx && toks[open].line != toks[i].line {
			if prev := f.prevNonWhitespace(i - 1); toks[prev].line == toks[i].line {
				next := f.nextNonWhitespace(closeIdx+1, n)
				if next < 0 || toks[next].kind != token.Comment {
					f.addBeforeRaw(closeIdx, f.eol)
				} else {
					next = f.nextEffective(next+1, n)
					f.beginChangeset()
					f.replaceToken(closeIdx, "")
					f.addBeforeRaw(next, ")")
					f.endChangeset()
				}
			}
		}

		if toks[i].line != prevLine {
			expected := statementIndent + r.Indent
			if toks[i].line == toks[closeIdx].line && f.nextNonWhitespace(i, n) == closeIdx {
				expected = statementIndent
			}
			if toks[i].kind == token.Comment {
				prevLine = toks[i].line
				continue
			}
			found := 0
			if toks[i].kind == token.Whitespace {
				found = toks[i].length
			}
			if expected != found {
				spaces := strings.Repeat(" ", expected)
				if found == 0 {
					f.addBeforeRaw(i, spaces)
				} else {
					f.replaceToken(i, spaces)
				}
			}

			next := f.nextEffective(i, n)
			if next >= 0 && next != closeIdx && toks[next].line == toks[i].line && !f.isBooleanOperator(next) {
				prev := i - 1
				for prev > open && f.isEmpty(prev) {
					prev--
				}
				if f.isBooleanOperator(prev) {
					f.beginChangeset()
					f.replaceToken(prev, "")
					f.addBeforeRaw(next, toks[prev].content+" ")
					f.endChangeset()
				} else if f.nextNonWhitespace(prev+1, next) < 0 {
					for x := prev + 1; x < next; x++ {
						f.replaceToken(x, "")
					}
				}
			}
			prevLine = toks[i].line
		}

		if toks[i].kind == token.Ident {
			if next := f.nextNonWhitespace(i+1, n); f.is(next, "(") && toks[next].closer > next {
				i = toks[next].closer
				prevLine = toks[i].line
				continue
			}
		}
	}

	openBrace, _, ok := f.icsScope(ptr)
	if !ok || f.nextNonWhitespace(closeIdx+1, openBrace) >= 0 {
		return
	}
	length := 1
	switch {
	case toks[openBrace].line > toks[closeIdx].line:
		length = -1
	case openBrace == closeIdx+1:
		length = 0
	case openBrace == closeIdx+2 && toks[closeIdx+1].kind == token.Whitespace:
		length = toks[closeIdx+1].length
	}
	switch length {
	case 1:
	case 0:
		f.icsAdd(closeIdx, " ")
	default:
		f.replaceToken(closeIdx+1, " ")
	}
}
