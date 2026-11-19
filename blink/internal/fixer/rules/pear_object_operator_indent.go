package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/WhiteSpace/ObjectOperatorIndentSniff.php
//
// PearObjectOperatorIndent indents each line of a chained call that starts
// with "->" or "?->" Indent spaces past the start of the statement, and moves
// an operator ending a line to the start of the next.
type PearObjectOperatorIndent struct {
	Indent     int
	Multilevel bool
}

func (PearObjectOperatorIndent) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\WhiteSpace\ObjectOperatorIndentSniff`
}

func (PearObjectOperatorIndent) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/WhiteSpace/ObjectOperatorIndentSniff.php"
}

func (r PearObjectOperatorIndent) WithConfig(config map[string]any) fixer.Fixer {
	r.Indent = configInt(config, "indent", r.Indent)
	if v, ok := config["multilevel"].(bool); ok {
		r.Multilevel = v
	}
	return r
}

func (r PearObjectOperatorIndent) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		sc := newPhpcsScope(f)
		for i := range f.toks {
			if f.is(i, "->") || f.is(i, "?->") {
				r.process(sc, i)
			}
		}
	})
}

func isObjectOperator(f *slvFile, i int) bool { return f.is(i, "->") || f.is(i, "?->") }

func (r PearObjectOperatorIndent) process(sc *phpcsScope, stackPtr int) {
	f := sc.f
	start := sc.startOfStatement(stackPtr)
	for i := stackPtr - 1; i >= start; i-- {
		if isObjectOperator(f, i) {
			return
		}
	}
	end := sc.endOfStatement(stackPtr)
	next := -1
	for i := stackPtr + 1; i < end; i++ {
		if isObjectOperator(f, i) {
			next = i
			break
		}
	}
	if next < 0 {
		return
	}
	i := start - 1
	for ; i >= 0; i-- {
		if f.toks[i].line != f.toks[start].line {
			i++
			break
		}
	}
	baseIndent := 0
	if i >= 0 && sc.code[i] == "whitespace" {
		baseIndent = f.toks[i].length
	}
	baseIndent += r.Indent
	origBrackets, origConditions := sc.nested[stackPtr], sc.conditions(stackPtr)
	if f.toks[stackPtr].line > f.toks[start].line {
		next = stackPtr
	}
	previousIndent := baseIndent
	for next >= 0 {
		if sc.nested[next] == origBrackets && slices.Equal(sc.conditions(next), origConditions) {
			prev := next - 1
			for prev >= stackPtr && sc.code[prev] == "whitespace" {
				prev--
			}
			if prev < stackPtr {
				prev = 0
			}
			indent := f.toks[next-1]
			if f.toks[prev].line != f.toks[next].line && sc.code[next-1] == "whitespace" {
				foundIndent := 0
				if indent.line == f.toks[next].line {
					foundIndent = len(indent.content)
				}
				minIndent, maxIndent, expectedIndent := previousIndent, previousIndent, previousIndent
				if r.Multilevel {
					minIndent = max(previousIndent-r.Indent, baseIndent)
					maxIndent = previousIndent + r.Indent
					expectedIndent = min(max(foundIndent, minIndent), maxIndent)
				}
				if foundIndent < minIndent || foundIndent > maxIndent {
					spaces := strings.Repeat(" ", expectedIndent)
					if foundIndent == 0 {
						f.replaceToken(next, spaces+f.tokenContent(next))
					} else {
						f.replaceToken(next-1, spaces)
					}
				}
				previousIndent = expectedIndent
			}
			content := next + 1
			for content < len(f.toks) && sc.code[content] == "whitespace" {
				content++
			}
			if content < len(f.toks) && f.toks[content].line != f.toks[next].line {
				f.beginChangeset()
				for x := next + 1; x < content; x++ {
					f.replaceToken(x, "")
				}
				f.replaceToken(next, f.eol+f.tokenContent(next))
				f.endChangeset()
			}
		}
		n := -1
		for i := next + 1; i < end; i++ {
			if isObjectOperator(f, i) {
				n = i
				break
			}
		}
		next = n
	}
}
