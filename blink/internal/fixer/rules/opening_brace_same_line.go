package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Classes/OpeningBraceSameLineSniff.php
//
// OpeningBraceSameLine puts the opening brace of a class, interface, trait or
// enum on its declaration line, one space after it, as the last content there.
type OpeningBraceSameLine struct{}

func (OpeningBraceSameLine) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Classes\OpeningBraceSameLineSniff`
}

func (OpeningBraceSameLine) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Classes/OpeningBraceSameLineSniff.php"
}

func (OpeningBraceSameLine) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if open := phpcsClassyOpener(f, i); open >= 0 {
				openingBraceSameLine(f, open)
			}
		}
	})
}

func openingBraceSameLine(f *slvFile, open int) {
	last := f.prevNonWhitespace(open - 1)
	lineDifference := f.toks[open].line - f.toks[last].line
	if lineDifference > 0 {
		f.beginChangeset()
		f.replaceToken(last, f.tokenContent(last)+" {")
		f.replaceToken(open, "")
		f.endChangeset()
	}
	if next := f.nextNonWhitespace(open+1, len(f.toks)); next >= 0 && f.toks[next].line == f.toks[open].line {
		if next == f.toks[open].closer {
			return
		}
		f.replaceToken(open, f.tokenContent(open)+f.eol)
	}
	if lineDifference > 0 {
		return
	}
	switch {
	case f.toks[open-1].kind != token.Whitespace:
		f.replaceToken(open, " "+f.tokenContent(open))
	case f.toks[open-1].length != 1:
		f.replaceToken(open-1, " ")
	}
}

// phpcsClassyOpener returns the scope opener PHP_CodeSniffer gives the named
// class, interface, trait or enum declared at i, or -1.
func phpcsClassyOpener(f *slvFile, i int) int {
	t := f.toks[i]
	if t.kind != token.Keyword {
		return -1
	}
	next := f.nextEffective(i+1, len(f.toks))
	switch strings.ToLower(t.orig) {
	case "class", "interface", "trait":
		if f.is(next, ":") {
			return -1
		}
	case "enum":
		if next < 0 || f.toks[next].kind != token.Ident {
			return -1
		}
	default:
		return -1
	}
	prev := f.prevEffective(i - 1)
	for prev >= 0 && f.toks[prev].kind == token.AttributeEnd && f.toks[prev].closer >= 0 {
		prev = f.prevEffective(f.toks[prev].closer - 1)
	}
	if prev >= 0 && f.isKeyword(prev, "readonly") {
		prev = f.prevEffective(prev - 1)
	}
	if f.isKeyword(prev, "new") || f.isKeyword(prev, "function") || f.isKeyword(prev, "const") ||
		f.is(prev, "::") || f.is(prev, "->") || f.is(prev, "?->") {
		return -1
	}
	for j := i + 1; j < len(f.toks); j++ {
		switch {
		case f.is(j, "{"):
			if f.toks[j].closer < 0 {
				return -1
			}
			return j
		case f.is(j, ";") || f.is(j, "(") || f.is(j, "}"):
			return -1
		case f.toks[j].kind == token.Keyword && j != i:
			switch strings.ToLower(f.toks[j].orig) {
			case "class", "interface", "trait", "enum", "function":
				return -1
			}
		}
	}
	return -1
}
