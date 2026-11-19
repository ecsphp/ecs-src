package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/EmbeddedPhpSniff.php
//
// EmbeddedPhp lays out PHP blocks embedded in HTML: a multi-line block gets
// its open and close tags on lines of their own, an inline one gets one space
// inside each tag and a closing semicolon, and an empty one is removed.
type EmbeddedPhp struct{}

func (EmbeddedPhp) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\EmbeddedPhpSniff`
}

func (EmbeddedPhp) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/EmbeddedPhpSniff.php"
}

const phpTrimChars = " \t\n\r\x00\x0B"

type embeddedPhp struct {
	f  *slvFile
	sc *phpcsScope
}

// phpcsOnePass: ECS stops on a fixer conflict between the open tag indent and
// the first content indent, which a second run would resolve differently.
func (EmbeddedPhp) phpcsOnePass() {}

func (EmbeddedPhp) Fix(s *tokens.Stream) bool {
	for i := range s.Len() {
		// ECS runs PHP with short_open_tag off, so "<?" is inline HTML there
		if t := s.At(i); t.Kind == token.OpenTag && t.Value == "<?" {
			return false
		}
	}
	return slvRun(s, func(f *slvFile) {
		e := embeddedPhp{f: f, sc: newPhpcsScope(f)}
		for i, t := range f.toks {
			if t.kind != token.OpenTag {
				continue
			}
			closeTag := e.next(token.CloseTag, i)
			if closeTag < 0 || f.toks[i].line != f.toks[closeTag].line {
				e.multiline(i, closeTag)
			} else {
				e.inline(i, closeTag)
			}
		}
	})
}

func (e embeddedPhp) next(kind token.Kind, from int) int {
	for i := from; i < len(e.f.toks); i++ {
		if e.f.toks[i].kind == kind {
			return i
		}
	}
	return -1
}

func (e embeddedPhp) line(i int) int { return e.f.toks[i].line }

// isCloser and isOpener tell a token that is its own scope_closer/scope_opener.
func (e embeddedPhp) isCloser(i int) bool { return e.sc.closer[i] == i }
func (e embeddedPhp) isOpener(i int) bool { return e.sc.opener[i] == i }

// prevNonWhitespace is findPrevious(T_WHITESPACE, start, end, true), with PHP
// reading the false it returns as token 0.
func (e embeddedPhp) prevNonWhitespace(start, end int) int {
	for i := start; i >= end; i-- {
		if e.f.toks[i].kind != token.Whitespace {
			return i
		}
	}
	return 0
}

func (e embeddedPhp) multiline(ptr, closingTag int) {
	f := e.f
	prevTag := -1
	for i := ptr - 1; i >= 0; i-- {
		if f.toks[i].kind == token.OpenTag {
			prevTag = i
			break
		}
	}
	if prevTag < 0 {
		return
	}
	firstContent := f.nextNonWhitespace(ptr+1, len(f.toks))
	if firstContent < 0 {
		return
	}
	if closingTag >= 0 {
		if f.nextNonWhitespace(closingTag+1, len(f.toks)) < 0 {
			return
		}
		if firstContent == closingTag {
			e.reportEmpty(ptr, closingTag)
			return
		}
	}

	indent, indentSet := 0, false
	if e.line(firstContent) == e.line(ptr) {
		padding := e.lineIndent(ptr)
		f.beginChangeset()
		f.replaceToken(ptr, strings.TrimRight(f.toks[ptr].content, phpTrimChars))
		f.addContent(ptr, f.eol)
		f.addContent(ptr, strings.Repeat(" ", padding))
		if f.toks[ptr+1].kind == token.Whitespace {
			f.replaceToken(ptr+1, "")
		}
		f.endChangeset()
	} else if !e.isCloser(firstContent) {
		if e.line(firstContent) > e.line(ptr)+1 {
			f.beginChangeset()
			for i := ptr + 1; i < firstContent; i++ {
				if e.line(i) == e.line(firstContent) || e.line(i) == e.line(ptr) {
					continue
				}
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}
		indent, indentSet = e.lineIndent(ptr), true
		if contentColumn := f.toks[firstContent].column - 1; contentColumn != indent {
			padding := strings.Repeat(" ", indent)
			if contentColumn == 0 {
				f.addContentBefore(firstContent, padding)
			} else {
				f.replaceToken(firstContent-1, padding)
			}
		}
	}

	lastBefore := e.prevNonWhitespace(ptr-1, 0)
	if e.line(lastBefore) == e.line(ptr) &&
		(f.toks[lastBefore].kind == token.InlineHTML && strings.Trim(f.toks[lastBefore].content, phpTrimChars) != "" ||
			lastBefore > 0 && f.toks[lastBefore-1].kind != token.InlineHTML && e.line(lastBefore-1) == e.line(ptr)) {
		padding := e.lineIndent(lastBefore)
		f.beginChangeset()
		f.addContentBefore(ptr, f.eol+strings.Repeat(" ", padding))
		if f.toks[ptr-1].kind == token.InlineHTML && strings.Trim(f.toks[ptr-1].content, phpTrimChars) == "" {
			f.replaceToken(ptr-1, "")
		}
		f.endChangeset()
	} else {
		first := lastBefore - 1
		for ; first > 0; first-- {
			if e.line(first) == e.line(ptr) {
				continue
			}
			if strings.Trim(f.toks[first].content, phpTrimChars) != "" {
				first = f.firstOnLine(first)
				break
			}
		}
		expected := e.lineIndent(max(first, 0)) + 4
		if found := f.toks[ptr].column - 1; found > expected {
			f.replaceToken(ptr-1, strings.Repeat(" ", expected))
		}
	}

	if closingTag < 0 {
		return
	}
	lastContent := e.prevNonWhitespace(closingTag-1, ptr+1)
	firstAfter := f.nextNonWhitespace(closingTag+1, len(f.toks))
	if e.line(lastContent) == e.line(closingTag) {
		closerIndent := 0
		if e.line(firstContent) == e.line(lastContent) && indentSet {
			closerIndent = indent
		} else {
			closerIndent = e.lineIndent(closingTag)
		}
		f.beginChangeset()
		if f.toks[closingTag-1].kind == token.Whitespace {
			f.replaceToken(closingTag-1, "")
		}
		f.addContentBefore(closingTag, strings.Repeat(" ", closerIndent))
		f.addContentBefore(closingTag, f.eol)
		f.endChangeset()
	} else if firstAfter >= 0 && e.line(firstAfter) == e.line(closingTag) {
		padding := e.lineIndent(closingTag)
		f.beginChangeset()
		f.addContent(closingTag, f.eol)
		f.addContent(closingTag, strings.Repeat(" ", padding))
		switch f.toks[firstAfter].kind {
		case token.InlineHTML:
			f.replaceToken(firstAfter, strings.TrimLeft(f.toks[firstAfter].content, phpTrimChars))
		case token.OpenTag:
			f.replaceToken(firstAfter, f.toks[firstAfter].content)
		}
		f.endChangeset()
	}

	if e.next(token.OpenTag, closingTag+1) < 0 {
		return
	}
	lastNonEmpty := 0
	for i := closingTag - 1; i >= ptr+1; i-- {
		if !f.isEmpty(i) {
			lastNonEmpty = i
			break
		}
	}
	if !e.isCloser(lastNonEmpty) && e.line(lastContent) < e.line(closingTag)-1 {
		f.beginChangeset()
		for i := lastContent + 1; i < closingTag; i++ {
			if e.line(i) == e.line(lastContent) || e.line(i) == e.line(closingTag) {
				continue
			}
			f.replaceToken(i, "")
		}
		f.endChangeset()
	}
}

func (e embeddedPhp) inline(ptr, closeTag int) {
	f := e.f
	if f.nextNonWhitespace(ptr+1, closeTag) < 0 {
		e.reportEmpty(ptr, closeTag)
		return
	}
	leadingSpace := 0
	if f.toks[ptr+1].kind == token.Whitespace {
		leadingSpace = f.toks[ptr+1].length
	}
	if leadingSpace != 1 {
		if f.toks[ptr+1].kind == token.Whitespace {
			f.replaceToken(ptr+1, " ")
		} else {
			f.addContent(ptr, " ")
		}
	}

	prev := 0
	for i := closeTag - 1; i >= ptr; i-- {
		if !f.isEmpty(i) {
			prev = i
			break
		}
	}
	if prev != ptr && !e.isOpener(prev) && !e.isCloser(prev) && !f.is(prev, ";") {
		f.addContent(prev, ";")
	}

	before := f.toks[closeTag-1]
	trailingSpace := 0
	isComment := before.kind == token.Comment
	if before.kind == token.Whitespace {
		trailingSpace = before.length
	} else if isComment && strings.HasSuffix(before.content, " ") {
		trailingSpace = len(before.content) - len(strings.TrimRight(before.content, phpTrimChars))
	}
	if trailingSpace != 1 {
		switch {
		case trailingSpace == 0:
			f.addContentBefore(closeTag, " ")
		case isComment:
			f.replaceToken(closeTag-1, strings.TrimRight(before.content, phpTrimChars)+" ")
		default:
			f.replaceToken(closeTag-1, " ")
		}
	}
}

func (e embeddedPhp) reportEmpty(ptr, closeTag int) {
	f := e.f
	f.beginChangeset()
	for i := ptr; i <= closeTag; i++ {
		f.replaceToken(i, "")
	}
	if closeTag+1 < len(f.toks) && e.line(closeTag+1) != e.line(closeTag) && ptr > 0 {
		if p := f.toks[ptr-1]; p.kind == token.InlineHTML && p.line == e.line(ptr) && p.column == 1 &&
			strings.Trim(p.content, phpTrimChars) == "" {
			f.replaceToken(ptr-1, "")
		}
	}
	f.endChangeset()
}

// lineIndent is the sniff's calculateLineIndent.
func (e embeddedPhp) lineIndent(ptr int) int {
	f := e.f
	first := ptr
	for first > 0 && f.toks[first].column != 1 {
		first--
	}
	for first > 0 && f.toks[first].kind == token.Comment && !phpcsIsAnnotation(f, first) &&
		f.toks[first].content != strings.TrimLeft(f.toks[first].content, phpTrimChars) {
		for first--; first > 0 && f.toks[first].column != 1; first-- {
		}
	}
	t := f.toks[first]
	leading := len(t.content) - len(strings.TrimLeft(t.content, phpTrimChars))
	switch {
	case t.kind == token.Whitespace:
		return f.toks[first+1].column - 1
	case t.kind == token.InlineHTML, t.kind == token.String && t.heredoc && !strings.HasSuffix(t.orig, "\n"):
		return leading
	case t.doc == slvDocWhitespace:
		return leading - 1
	}
	return 0
}
