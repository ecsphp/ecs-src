package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Files/FileHeaderSniff.php
//
// PSR12FileHeader separates the file header blocks (open tag, file docblock,
// declares, namespace, use imports) by one blank line and removes the blank
// lines inside each block.
type PSR12FileHeader struct{}

func (PSR12FileHeader) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Files\FileHeaderSniff`
}

func (PSR12FileHeader) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Files/FileHeaderSniff.php"
}

func (PSR12FileHeader) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if phpcsIsOpenTag(f, i) {
				psr12FileHeader(f, i)
				return
			}
		}
	})
}

type psr12HeaderLine struct {
	typ        string
	start, end int
}

func psr12FileHeader(f *slvFile, ptr int) {
	possible := map[int][]psr12HeaderLine{}
	openTag := ptr
	for openTag >= 0 {
		lines := psr12HeaderLines(f, openTag)
		if len(lines) == 0 && openTag == ptr {
			return
		}
		possible[openTag] = lines
		if len(lines) > 1 {
			break
		}
		next := -1
		for j := openTag + 1; j < len(f.toks); j++ {
			if phpcsIsOpenTag(f, j) || phpcsIsOOKeyword(f, j) {
				next = j
				break
			}
		}
		if phpcsIsOOKeyword(f, max(next, 0)) {
			break
		}
		openTag = next
	}
	if openTag < 0 {
		if len(possible) > 1 || ptr > 0 {
			return
		}
		openTag = ptr
	}
	psr12ProcessHeaderLines(f, possible[openTag])
}

func psr12HeaderLines(f *slvFile, ptr int) []psr12HeaderLine {
	n := len(f.toks)
	var sc *phpcsScope
	endOfStatement := func(i int) int {
		if sc == nil {
			sc = newPhpcsScope(f)
		}
		return sc.endOfStatement(i)
	}
	nonWhitespace := func(from int) int {
		if from >= n {
			return -1
		}
		return f.nextNonWhitespace(from, n)
	}
	next := nonWhitespace(ptr + 1)
	if next < 0 {
		return nil
	}
	lines := []psr12HeaderLine{{"tag", ptr, ptr}}
	foundDocblock := false
loop:
	for next >= 0 {
		t := f.toks[next]
		switch {
		case t.doc == slvDocOpen:
			if foundDocblock {
				break loop
			}
			end := next
			for end < n-1 && f.toks[end].doc != slvDocClose {
				end++
			}
			doc := end + 1
			for ; doc < n; doc++ {
				if f.isEmpty(doc) {
					continue
				}
				if f.toks[doc].kind == token.AttributeStart && f.toks[doc].closer >= 0 {
					doc = f.toks[doc].closer
					continue
				}
				break
			}
			if doc == n {
				doc--
			}
			if !psr12HeaderCommentOpener(f, doc) {
				annotation := false
				for i := next; i < end; i++ {
					if f.toks[i].doc == slvDocTag && strings.ToLower(f.toks[i].content) == "@var" {
						annotation = true
						break
					}
				}
				if !annotation {
					foundDocblock = true
					lines = append(lines, psr12HeaderLine{"docblock", next, end})
				}
			}
			next = end
		case f.isKeyword(next, "declare") || f.isKeyword(next, "namespace"):
			if psr12HasScopeOpener(f, next) {
				break loop
			}
			end := endOfStatement(next)
			lines = append(lines, psr12HeaderLine{strings.ToLower(t.orig), next, end})
			next = end
		case phpcsIsUse(f, next):
			typ := "use"
			if useType := f.nextEffective(next+1, n); useType >= 0 {
				if c := strings.ToLower(f.toks[useType].content); c == "function" || c == "const" {
					typ += " " + c
				}
			}
			end := endOfStatement(next)
			lines = append(lines, psr12HeaderLine{typ, next, end})
			next = end
		case t.kind == token.Comment || t.kind == token.DocComment:
			nonComment := -1
			for j := next + 1; j < n; j++ {
				if k := f.toks[j].kind; k != token.Comment && k != token.DocComment {
					nonComment = j
					break
				}
			}
			if nonComment < 0 {
				break loop
			}
			next = nonComment - 1
		default:
			break loop
		}
		next = nonWhitespace(next + 1)
	}
	return lines
}

// psr12HeaderCommentOpener reports whether the docblock before i documents
// code (a class, function, switch, ... or a modifier) rather than the file.
func psr12HeaderCommentOpener(f *slvFile, i int) bool {
	t := f.toks[i]
	if t.kind != token.Keyword {
		return false
	}
	switch strings.ToLower(t.orig) {
	case "class", "interface", "trait", "enum", "function", "switch", "case", "default",
		"else", "elseif", "catch", "finally", "match",
		"private", "public", "protected", "abstract", "static", "final", "readonly":
		return true
	}
	return false
}

// psr12HasScopeOpener reports whether the declare or namespace at i has a
// body ("declare(ticks=1) {", "namespace Foo {").
func psr12HasScopeOpener(f *slvFile, i int) bool {
	for j := i + 1; j < len(f.toks); j++ {
		switch {
		case f.is(j, "(") && f.toks[j].closer >= 0:
			j = f.toks[j].closer
		case f.is(j, "{"):
			return true
		case f.is(j, ":") && f.isKeyword(i, "declare"):
			return true
		case f.is(j, ";") || f.toks[j].kind == token.CloseTag:
			return false
		}
	}
	return false
}

func psr12ProcessHeaderLines(f *slvFile, lines []psr12HeaderLine) {
	n := len(f.toks)
	for i, line := range lines {
		endLine := f.toks[line.end].line
		next := -1
		if line.end+1 < n {
			next = f.nextNonWhitespace(line.end+1, n)
		}
		if i+1 == len(lines) || lines[i+1].typ != line.typ {
			if next < 0 || f.toks[next].line == endLine+2 {
				continue
			}
			switch f.toks[next].line {
			case endLine:
				f.addContentBefore(next, f.eol+f.eol)
			case endLine + 1:
				f.addContent(line.end, f.eol)
			default:
				f.beginChangeset()
				for j := line.end + 1; j < next; j++ {
					if f.toks[j].line == endLine+2 {
						break
					}
					f.replaceToken(j, "")
				}
				f.endChangeset()
			}
			continue
		}
		next = max(next, 0)
		if f.toks[next].line <= endLine+1 {
			continue
		}
		f.beginChangeset()
		for j := line.end + 1; j < next; j++ {
			if f.toks[j].line == endLine {
				continue
			}
			if f.toks[j].line == f.toks[next].line {
				break
			}
			f.replaceToken(j, "")
		}
		f.endChangeset()
	}
}

// phpcsIsOpenTag tells T_OPEN_TAG from "<?=" (T_OPEN_TAG_WITH_ECHO).
func phpcsIsOpenTag(f *slvFile, i int) bool {
	return f.toks[i].kind == token.OpenTag && !strings.HasPrefix(f.toks[i].orig, "<?=")
}

// phpcsIsOOKeyword reports whether i is a token of Tokens::OO_SCOPE_TOKENS.
func phpcsIsOOKeyword(f *slvFile, i int) bool {
	if f.toks[i].kind != token.Keyword {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "class", "interface", "trait":
		p := f.prevEffective(i - 1)
		return !f.is(p, "::") && !f.is(p, "->") && !f.is(p, "?->")
	case "enum":
		next := f.nextEffective(i+1, len(f.toks))
		return next >= 0 && f.toks[next].kind == token.Ident
	}
	return false
}
