package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Files/DeclareStatementSniff.php
//
// PSR12DeclareStatement strips the spaces inside "declare(ticks=1);",
// lowercases the directive and lays a declare block out like "declare(x=1) {".
type PSR12DeclareStatement struct{}

func (PSR12DeclareStatement) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Files\DeclareStatementSniff`
}

func (PSR12DeclareStatement) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Files/DeclareStatementSniff.php"
}

func (PSR12DeclareStatement) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.isKeyword(i, "declare") {
				psr12DeclareStatement(f, i)
			}
		}
	})
}

func psr12DeclareStatement(f *slvFile, ptr int) {
	n := len(f.toks)
	// PHP reads $tokens[false] as $tokens[0]
	at := func(i int) int { return max(i, 0) }
	isString := func(i int) bool { return i >= 0 && i < n && f.toks[i].kind == token.Ident }
	isLnumber := func(i int) bool {
		if i < 0 || i >= n || f.toks[i].kind != token.Number {
			return false
		}
		v := strings.ToLower(f.toks[i].orig)
		return strings.HasPrefix(v, "0x") || !strings.ContainsAny(v, ".e")
	}
	findNext := func(from int, match func(int) bool) int {
		for j := from; j < n; j++ {
			if match(j) {
				return j
			}
		}
		return -1
	}
	isPunct := func(v string) func(int) bool { return func(i int) bool { return f.is(i, v) } }
	nonWhitespace := func(from int) int {
		if from >= n {
			return -1
		}
		return f.nextNonWhitespace(from, n)
	}

	paren := ptr + 1
	if !f.is(ptr+1, "(") {
		paren = nonWhitespace(ptr + 1)
		if f.is(at(paren), "(") {
			f.replaceToken(ptr+1, "")
		} else {
			paren = findNext(at(paren)+1, isPunct("("))
		}
	}

	str := nonWhitespace(at(paren) + 1)
	if paren >= 0 && !isString(paren+1) {
		if isString(at(str)) {
			f.replaceToken(paren+1, "")
		} else {
			str = findNext(at(str)+1, isString)
		}
	}

	equals := nonWhitespace(at(str) + 1)
	if str >= 0 {
		if c := f.toks[str].content; c != strings.ToLower(c) {
			f.replaceToken(str, strings.ToLower(c))
		}
		if !f.is(str+1, "=") {
			if f.is(at(equals), "=") {
				f.replaceToken(str+1, "")
			} else {
				equals = findNext(at(equals)+1, isPunct("="))
			}
		}
	}

	value := nonWhitespace(at(equals) + 1)
	if value < 0 {
		return
	}
	if equals >= 0 && !isLnumber(equals+1) {
		if isLnumber(value) {
			f.replaceToken(equals+1, "")
		} else {
			value = findNext(value+1, isLnumber)
		}
	}

	paren = nonWhitespace(at(value) + 1)
	if value >= 0 && !f.is(value+1, ")") {
		if f.is(at(paren), ")") {
			f.replaceToken(value+1, "")
		} else {
			paren = findNext(at(paren)+1, isPunct(")"))
		}
	}

	curly := -1
	if !f.is(at(paren)+1, ";") {
		tok := at(nonWhitespace(at(paren) + 1))
		switch {
		case f.is(tok, "{"):
			curly = tok
		case f.is(tok, ";"):
			f.replaceToken(at(paren)+1, "")
		case f.toks[tok].kind == token.CloseTag:
			if f.toks[at(paren)].line != f.toks[tok].line {
				f.replaceToken(at(paren)+1, " ")
			}
		default:
			tok = findNext(tok+1, func(j int) bool {
				return f.toks[j].kind != token.Whitespace && (f.toks[j].kind != token.Comment || phpcsIsAnnotation(f, j))
			})
			if f.is(at(tok), "{") {
				curly = tok
			}
		}
	}
	if curly < 0 {
		return
	}
	psr12DeclareBlock(f, ptr, curly)
}

func psr12DeclareBlock(f *slvFile, ptr, curly int) {
	n := len(f.toks)
	prev := f.prevNonWhitespace(curly - 1)
	if f.is(prev, ")") && f.content(prev+1, curly-1) != " " {
		f.beginChangeset()
		f.replaceToken(prev+1, " ")
		for j := prev + 2; j < curly; j++ {
			f.replaceToken(j, "")
		}
		f.endChangeset()
	}

	closeCurly := f.toks[curly].closer
	if closeCurly < 0 {
		return
	}
	prev = f.prevNonWhitespace(closeCurly - 1)
	if f.toks[max(prev, 0)].line == f.toks[closeCurly].line {
		f.addContent(prev, f.eol)
	}

	if f.toks[ptr].column != f.toks[closeCurly].column {
		f.replaceToken(closeCurly-1, strings.Repeat(" ", f.toks[ptr].column-1))
	}

	next := max(f.nextEffective(curly+1, n), 0)
	if f.toks[curly].line == f.toks[next].line {
		f.beginChangeset()
		prev = f.prevNonWhitespace(next - 1)
		for j := prev + 1; j < next; j++ {
			f.replaceToken(j, "")
		}
		f.addContentBefore(next, f.eol)
		f.endChangeset()
	}
}
