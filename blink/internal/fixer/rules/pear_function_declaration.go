package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Functions/FunctionDeclarationSniff.php
//
// PEARFunctionDeclaration fixes the spacing of function and closure
// declarations, indents multi-line parameter lists by Indent spaces and
// places the opening brace (BSD/Allman for functions, K&R for closures).
type PEARFunctionDeclaration struct {
	Indent int
}

func (PEARFunctionDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Functions\FunctionDeclarationSniff`
}

func (PEARFunctionDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Functions/FunctionDeclarationSniff.php"
}

func (f PEARFunctionDeclaration) WithConfig(config map[string]any) fixer.Fixer {
	f.Indent = configInt(config, "indent", f.Indent)
	return f
}

// phpcsFn is a T_FUNCTION or T_CLOSURE token with the pointers PHP_CodeSniffer
// gives it; scopeOpener and use are -1 when absent.
type phpcsFn struct {
	ptr, openParen, closeParen, scopeOpener, use int
	closure                                      bool
}

// phpcsFunctionAt returns the function or closure declared by the "function"
// keyword at i; ok is false when it has no closed parameter list.
func phpcsFunctionAt(f *slvFile, i int) (fn phpcsFn, ok bool) {
	if !f.isKeyword(i, "function") {
		return fn, false
	}
	fn = phpcsFn{ptr: i, openParen: -1, scopeOpener: -1, use: -1}
	for j := i + 1; j < len(f.toks); j++ {
		if f.is(j, "(") {
			fn.openParen = j
			break
		}
		if f.is(j, ";") || f.is(j, "{") || f.is(j, "}") {
			return fn, false
		}
	}
	if fn.openParen < 0 || f.toks[fn.openParen].closer < 0 {
		return fn, false
	}
	fn.closeParen = f.toks[fn.openParen].closer
	for j := fn.closeParen + 1; j < len(f.toks); j++ {
		switch {
		case f.isKeyword(j, "use") && fn.use < 0:
			fn.use = j
		case f.is(j, "(") || f.is(j, "["):
			if f.toks[j].closer < 0 {
				j = len(f.toks)
				continue
			}
			j = f.toks[j].closer
		case f.is(j, "{"):
			if f.toks[j].closer >= 0 {
				fn.scopeOpener = j
			}
			j = len(f.toks)
		case f.is(j, ";") || f.is(j, "}") || f.is(j, ")"):
			j = len(f.toks)
		}
	}
	// an unfinished closure is tokenized as T_FUNCTION
	fn.closure = f.isClosure(i) && fn.scopeOpener >= 0
	if !fn.closure {
		fn.use = -1
	}
	return fn, true
}

// useParens returns the parentheses of a closure's use clause, or -1, -1.
func (fn phpcsFn) useParens(f *slvFile) (open, closeIdx int) {
	if fn.use < 0 {
		return -1, -1
	}
	o := f.nextEffective(fn.use+1, len(f.toks))
	if !f.is(o, "(") || f.toks[o].closer < 0 {
		return -1, -1
	}
	return o, f.toks[o].closer
}

func (d PEARFunctionDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if fn, ok := phpcsFunctionAt(f, i); ok {
				d.process(f, fn)
			}
		}
	})
}

// phpcsSpacesAt is the sniffs' "newline" / whitespace length / 0 measure of
// the token at i, with -1 standing for "newline".
func phpcsSpacesAt(f *slvFile, i int) int {
	switch {
	case f.toks[i].content == f.eol:
		return -1
	case f.toks[i].kind == token.Whitespace:
		return f.toks[i].length
	}
	return 0
}

func (d PEARFunctionDeclaration) process(f *slvFile, fn phpcsFn) {
	stackPtr := fn.ptr
	if strings.ToLower(f.toks[stackPtr].orig) == "function" {
		if spaces := phpcsSpacesAt(f, stackPtr+1); spaces != 1 {
			if spaces == 0 {
				f.replaceToken(stackPtr, f.tokenContent(stackPtr)+" ")
			} else {
				f.replaceToken(stackPtr+1, " ")
			}
		}
	}

	if !fn.closure && f.prevEffective(fn.openParen-1) != stackPtr && !f.is(f.prevEffective(fn.openParen-1), "&") {
		if phpcsSpacesAt(f, fn.openParen-1) != 0 {
			f.replaceToken(fn.openParen-1, "")
		}
		if fn.scopeOpener < 0 {
			end := -1
			for j := fn.closeParen; j < len(f.toks); j++ {
				if f.is(j, ";") {
					end = j
					break
				}
			}
			if end >= 0 && phpcsSpacesAt(f, end-1) != 0 {
				f.replaceToken(end-1, "")
			}
		}
	}

	if fn.closure && fn.use >= 0 {
		use := fn.use
		if l := phpcsUseGap(f, use+1); l != 1 {
			if l == 0 {
				f.replaceToken(use, f.tokenContent(use)+" ")
			} else {
				f.replaceToken(use+1, " ")
			}
		}
		if l := phpcsUseGap(f, use-1); l != 1 {
			if l == 0 {
				f.replaceToken(use, " "+f.tokenContent(use))
			} else {
				f.replaceToken(use-1, " ")
			}
		}
	}

	if d.isMultiLine(f, fn) {
		d.processMultiLine(f, fn)
		return
	}
	if fn.scopeOpener < 0 {
		return
	}
	if fn.closure {
		openingFunctionBraceKernighanRitchie(f, fn.scopeOpener)
		return
	}
	openingFunctionBraceBsdAllman(f, stackPtr, fn.closeParen, fn.scopeOpener)
}

func phpcsUseGap(f *slvFile, i int) int {
	if f.toks[i].kind != token.Whitespace {
		return 0
	}
	return f.toks[i].length
}

func (PEARFunctionDeclaration) isMultiLine(f *slvFile, fn phpcsFn) bool {
	if f.toks[fn.openParen].line != f.toks[fn.closeParen].line {
		return true
	}
	if o, c := fn.useParens(f); o >= 0 && f.toks[o].line != f.toks[c].line {
		return true
	}
	return false
}

func (d PEARFunctionDeclaration) processMultiLine(f *slvFile, fn phpcsFn) {
	d.processArgumentList(f, fn)

	closeBracket := fn.closeParen
	if _, c := fn.useParens(f); c >= 0 {
		closeBracket = c
	}
	if fn.scopeOpener < 0 {
		return
	}
	opener := fn.scopeOpener
	prev := opener - 1
	for prev > closeBracket && f.isEmpty(prev) {
		prev--
	}
	if f.toks[opener].line != f.toks[closeBracket].line && f.toks[prev].line != f.toks[opener].line {
		f.beginChangeset()
		f.replaceToken(prev, f.tokenContent(prev)+" {")
		prev = opener - 1
		for prev > closeBracket && f.toks[prev].kind == token.Whitespace {
			prev--
		}
		next := opener + 1
		for next < len(f.toks) && f.toks[next].kind == token.Whitespace {
			next++
		}
		openerLine := f.toks[opener].line
		if f.toks[prev].line < openerLine && next < len(f.toks) && f.toks[next].line > openerLine {
			for i := prev + 1; i < next; i++ {
				if f.toks[i].line == openerLine {
					f.replaceToken(i, "")
				}
			}
		} else {
			f.replaceToken(opener, "")
			if next < len(f.toks) && f.toks[next].line == openerLine && opener+1 != next {
				f.replaceToken(opener+1, "")
			}
		}
		f.endChangeset()
		return
	}

	length := 0
	if f.toks[opener-1].kind == token.Whitespace {
		length = len(f.toks[opener-1].content)
	}
	if length != 1 {
		if length == 0 {
			f.replaceToken(opener, " "+f.tokenContent(opener))
		} else {
			f.replaceToken(opener-1, " ")
		}
	}
}

func (d PEARFunctionDeclaration) processArgumentList(f *slvFile, fn phpcsFn) {
	stackPtr := fn.ptr
	i := stackPtr - 1
	for i >= 0 && f.toks[i].line == f.toks[stackPtr].line {
		i--
	}
	i++
	functionIndent := 0
	if f.toks[i].kind == token.Whitespace {
		functionIndent = f.toks[i].length
	}

	closeBracket := fn.closeParen
	phpcsCloserOnNewLine(f, closeBracket)
	if fn.closure && fn.use >= 0 {
		if _, c := fn.useParens(f); c >= 0 {
			closeBracket = c
			phpcsCloserOnNewLine(f, closeBracket)
		}
	}

	lastLine := f.toks[fn.openParen].line
	for i := fn.openParen + 1; i < closeBracket; i++ {
		t := f.toks[i]
		if t.line != lastLine {
			expectedIndent := functionIndent + d.Indent
			if i == fn.closeParen || t.kind == token.Whitespace && (i+1 == closeBracket || i+1 == fn.closeParen) {
				expectedIndent = functionIndent
			}
			foundIndent := 0
			isComment := t.kind == token.Comment
			switch {
			case t.kind == token.Whitespace && t.line != f.toks[i+1].line:
				f.replaceToken(i, "")
				continue
			case t.kind == token.Whitespace:
				foundIndent = t.length
			case t.kind == token.DocComment && t.doc == slvDocWhitespace:
				foundIndent = t.length
				expectedIndent++
			case isComment && f.toks[i-1].kind == token.Comment && f.toks[i-1].line == lastLine:
				trimmed := strings.TrimLeft(t.content, " \t\n\r\x00\x0B")
				if trimmed == "" {
					lastLine = t.line
					continue
				}
				foundIndent = len(t.content) - len(trimmed)
				if trimmed[0] == '*' {
					expectedIndent++
				} else if foundIndent >= expectedIndent {
					lastLine = t.line
					continue
				}
			}
			if expectedIndent != foundIndent {
				spaces := strings.Repeat(" ", expectedIndent)
				switch {
				case foundIndent == 0:
					f.replaceToken(i, spaces+f.tokenContent(i))
				case isComment:
					f.replaceToken(i, spaces+strings.TrimLeft(t.content, " \t\n\r\x00\x0B"))
				default:
					f.replaceToken(i, spaces)
				}
			}
			lastLine = t.line
		}

		if f.is(i, "(") && t.closer >= 0 {
			if p := f.prevEffective(i - 1); !f.isKeyword(p, "use") {
				i = t.closer
				lastLine = f.toks[i].line
				continue
			}
		}
		if f.isShortArrayOpen(i) {
			i = t.closer
			lastLine = f.toks[i].line
			continue
		}
		if f.isKeyword(i, "array") {
			if o := f.nextEffective(i+1, len(f.toks)); f.is(o, "(") && f.toks[o].closer >= 0 {
				i = f.toks[o].closer
				lastLine = f.toks[i].line
				continue
			}
		}
		if t.kind == token.AttributeStart && t.closer >= 0 {
			i = t.closer
			lastLine = f.toks[i].line
			continue
		}
	}
}

// phpcsCloserOnNewLine moves the ")" at closeIdx of a multi-line list to its
// own line.
func phpcsCloserOnNewLine(f *slvFile, closeIdx int) {
	open := f.toks[closeIdx].closer
	prev := closeIdx - 1
	for prev >= 0 && f.toks[prev].kind == token.Whitespace {
		prev--
	}
	if f.toks[closeIdx].line != f.toks[open].line && f.toks[prev].line == f.toks[closeIdx].line {
		f.replaceToken(closeIdx, f.eol+f.tokenContent(closeIdx))
	}
}
