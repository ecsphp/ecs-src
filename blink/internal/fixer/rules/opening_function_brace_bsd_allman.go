package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Functions/OpeningFunctionBraceBsdAllmanSniff.php
//
// OpeningFunctionBraceBsdAllman puts a function's opening brace on the line
// after its declaration, indented like the declaration. Closures are left
// alone (checkClosures defaults to false).
type OpeningFunctionBraceBsdAllman struct{}

func (OpeningFunctionBraceBsdAllman) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Functions\OpeningFunctionBraceBsdAllmanSniff`
}

func (OpeningFunctionBraceBsdAllman) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Functions/OpeningFunctionBraceBsdAllmanSniff.php"
}

func (OpeningFunctionBraceBsdAllman) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if closeParen, open := phpcsNamedFunctionOpener(f, i); open >= 0 {
				openingFunctionBraceBsdAllman(f, i, closeParen, open)
			}
		}
	})
}

func openingFunctionBraceBsdAllman(f *slvFile, fn, closeParen, open int) {
	prev := open - 1
	for prev > closeParen && f.isEmpty(prev) {
		prev--
	}
	braceLine := f.toks[open].line
	lineDifference := braceLine - f.toks[prev].line
	switch {
	case lineDifference == 0:
		nextLine := open + 1
		trailingAnnotation := false
		for ; nextLine < len(f.toks) && f.toks[nextLine].line == braceLine; nextLine++ {
			if phpcsIsAnnotation(f, nextLine) {
				trailingAnnotation = true
			}
		}
		f.beginChangeset()
		if !trailingAnnotation || nextLine >= len(f.toks) {
			if f.toks[open-1].kind == token.Whitespace {
				f.replaceToken(open-1, "")
			}
			f.replaceToken(open, f.eol+f.tokenContent(open))
		} else {
			f.replaceToken(open, "")
			f.replaceToken(nextLine, f.eol+f.tokenContent(nextLine))
			f.replaceToken(nextLine, "{"+f.tokenContent(nextLine))
		}
		f.endChangeset()
	case lineDifference > 1:
		prevNonWs := open - 1
		for prevNonWs > closeParen && f.toks[prevNonWs].kind == token.Whitespace {
			prevNonWs--
		}
		if prevNonWs == prev {
			f.beginChangeset()
			for i := open; i > prev; i-- {
				if f.toks[i].line == braceLine {
					if f.toks[i].column == 1 {
						f.replaceToken(i, f.eol+f.tokenContent(i))
					}
					continue
				}
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}
	}
	phpcsContentAfterBrace(f, open, false)
	if lineDifference != 1 {
		return
	}
	lineStart := f.firstNonWhitespaceOnLine(fn)
	expected, found := f.toks[lineStart].column-1, f.toks[open].column-1
	if expected == found {
		return
	}
	indent := strings.Repeat(" ", expected)
	if found == 0 {
		f.replaceToken(open, indent+f.tokenContent(open))
	} else {
		f.replaceToken(open-1, indent)
	}
}

// phpcsContentAfterBrace moves code after a function's opening brace to the
// next line, unless the body is empty (or, with allowCloseTag, the code is "?>").
func phpcsContentAfterBrace(f *slvFile, open int, allowCloseTag bool) {
	next := open + 1
	for next < len(f.toks) && (f.toks[next].kind == token.Whitespace || phpcsIsAnnotation(f, next)) {
		next++
	}
	if next >= len(f.toks) || f.toks[next].line != f.toks[open].line || next == f.toks[open].closer {
		return
	}
	if allowCloseTag && f.toks[next].kind == token.CloseTag {
		return
	}
	f.replaceToken(open, f.tokenContent(open)+f.eol)
}

// phpcsIsAnnotation reports whether PHP_CodeSniffer tokenizes the comment at i
// as a phpcs: annotation, as it does even with annotations turned off.
func phpcsIsAnnotation(f *slvFile, i int) bool {
	if k := f.toks[i].kind; k != token.Comment && k != token.DocComment {
		return false
	}
	text := strings.ToLower(strings.TrimRight(strings.TrimLeft(f.toks[i].content, " \t/*#"), " */\t\r\n"))
	text = strings.TrimPrefix(text, "@")
	for _, p := range []string{"phpcs:set", "phpcs:ignorefile", "phpcs:disable", "phpcs:enable", "phpcs:ignore"} {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

// phpcsNamedFunctionOpener returns the closing parenthesis of the parameters
// and the scope opener of the named function declared at fn, or -1 when it is
// not one or has no body.
func phpcsNamedFunctionOpener(f *slvFile, fn int) (closeParen, open int) {
	if !f.isKeyword(fn, "function") || f.isClosure(fn) {
		return -1, -1
	}
	paren := f.nextEffective(fn+1, len(f.toks))
	if f.is(paren, "&") {
		paren = f.nextEffective(paren+1, len(f.toks))
	}
	if paren < 0 || f.toks[paren].kind == token.Punct {
		return -1, -1
	}
	paren = f.nextEffective(paren+1, len(f.toks))
	if !f.is(paren, "(") || f.toks[paren].closer < 0 {
		return -1, -1
	}
	closeParen = f.toks[paren].closer
	for j := closeParen + 1; j < len(f.toks); j++ {
		switch {
		case f.is(j, "(") && f.toks[j].closer >= 0:
			j = f.toks[j].closer
		case f.is(j, "{"):
			if f.toks[j].closer < 0 {
				return -1, -1
			}
			return closeParen, j
		case f.is(j, ";") || f.is(j, "}"):
			return -1, -1
		}
	}
	return -1, -1
}
