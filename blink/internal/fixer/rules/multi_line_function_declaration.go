package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/MultiLineFunctionDeclarationSniff.php
//
// MultiLineFunctionDeclaration is PEAR FunctionDeclaration that also calls a
// declaration multi-line when its parameters span lines, puts each parameter
// of such a declaration on its own line and keeps a single-line one's ")" on
// the last parameter's line.
type MultiLineFunctionDeclaration struct {
	Indent int
}

func (MultiLineFunctionDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Functions\MultiLineFunctionDeclarationSniff`
}

func (MultiLineFunctionDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Functions/MultiLineFunctionDeclarationSniff.php"
}

func (m MultiLineFunctionDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	m.Indent = configInt(config, "indent", m.Indent)
	return m
}

func (m MultiLineFunctionDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		var sc *phpcsScope
		for i := range f.toks {
			fn, ok := phpcsFunctionAt(f, i)
			if !ok {
				continue
			}
			if sc == nil {
				sc = newPhpcsScope(f)
			}
			m.process(f, sc, fn)
		}
	})
}

func (m MultiLineFunctionDeclaration) process(f *slvFile, sc *phpcsScope, fn phpcsFn) {
	pear := PEARFunctionDeclaration(m)
	pear.processHead(f, fn)
	if !m.isMultiLine(f, sc, fn) {
		pearSingleLineBrace(f, fn)
		m.closerOnParamLine(f, fn)
		return
	}
	pear.processMultiLine(f, fn)
	m.processBracket(f, fn.openParen)
	if open, _ := fn.useParens(f); open >= 0 {
		m.processBracket(f, open)
	}
}

func (MultiLineFunctionDeclaration) isMultiLine(f *slvFile, sc *phpcsScope, fn phpcsFn) bool {
	type bracket struct{ ptr, open int }
	check := []bracket{{fn.ptr, fn.openParen}}
	if open, _ := fn.useParens(f); open >= 0 {
		check = append(check, bracket{fn.use, open})
	}
next:
	for _, b := range check {
		if n := f.nextEffective(b.open+1, len(f.toks)); n < 0 || f.toks[n].line != f.toks[b.ptr].line {
			return true
		}
		closeIdx := f.toks[b.open].closer
		end := sc.endOfStatement(b.open + 1)
		for f.is(end, ",") {
			n := f.nextEffective(end+1, closeIdx)
			if n < 0 {
				continue next
			}
			if f.toks[n].line != f.toks[end].line {
				return true
			}
			end = sc.endOfStatement(n)
		}
		if n := f.nextEffective(end+1, closeIdx); n >= 0 && f.toks[n].line != f.toks[end].line {
			return true
		}
	}
	return false
}

// closerOnParamLine removes a newline before the ")" of a single-line
// declaration.
func (MultiLineFunctionDeclaration) closerOnParamLine(f *slvFile, fn phpcsFn) {
	open, closeIdx := fn.openParen, fn.closeParen
	prev := f.prevNot(f.isWS, closeIdx-1, open-1)
	if prev < 0 || f.toks[prev].line == f.toks[closeIdx].line || f.isEmpty(prev) {
		return
	}
	f.beginChangeset()
	for i := closeIdx - 1; i > open && f.isWS(i); i-- {
		f.replaceToken(i, "")
	}
	f.endChangeset()
}

func (MultiLineFunctionDeclaration) processBracket(f *slvFile, open int) {
	closeIdx := f.toks[open].closer
	if f.toks[open].line != f.toks[closeIdx].line {
		if n := f.nextEffective(open+1, len(f.toks)); n >= 0 && f.toks[n].line == f.toks[open].line {
			f.addContent(open, f.eol)
		}
	}
	for i := open + 1; i < closeIdx; i++ {
		t := f.toks[i]
		if (f.is(i, "(") || f.is(i, "[") || f.is(i, "{") || t.kind == token.AttributeStart) && t.closer > i {
			i = t.closer
			continue
		}
		if !f.is(i, ",") {
			continue
		}
		if n := f.nextEffective(i+1, len(f.toks)); n >= 0 && f.toks[n].line == t.line {
			f.addContent(i, f.eol)
		}
	}
}
