package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Traits/UseDeclarationSniff.php
//
// TraitUseDeclaration lays out trait imports: the first right after the class
// opening brace, one trait per "use", one space after "use", a blank line after
// the last import, and one conflict resolution statement per line.
type TraitUseDeclaration struct{}

func (TraitUseDeclaration) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Traits\UseDeclarationSniff`
}

func (TraitUseDeclaration) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Traits/UseDeclarationSniff.php"
}

func (TraitUseDeclaration) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.isKeyword(i, "use") }, func(i int) int {
			return traitUseDeclaration(f, i)
		})
	})
}

func (f *slvFile) isWS(i int) bool { return f.toks[i].kind == token.Whitespace }

func (f *slvFile) isCommentToken(i int) bool {
	k := f.toks[i].kind
	return k == token.Comment || k == token.DocComment
}

// docOpener steps from a docblock close tag back to its open tag.
func (f *slvFile) docOpener(i int) int {
	if f.toks[i].kind != token.DocComment || f.toks[i].doc != slvDocClose {
		return i
	}
	for j := i; j >= 0; j-- {
		if f.toks[j].doc == slvDocOpen {
			return j
		}
	}
	return i
}

// prevNot is File::findPrevious with exclude=true: the first token from start
// down to after end that skip rejects, or -1.
func (f *slvFile) prevNot(skip func(int) bool, start, end int) int {
	for i := start; i > end && i >= 0; i-- {
		if !skip(i) {
			return i
		}
	}
	return -1
}

// nextNot is File::findNext with exclude=true, end exclusive.
func (f *slvFile) nextNot(skip func(int) bool, start, end int) int {
	end = min(end, len(f.toks))
	for i := start; i < end; i++ {
		if !skip(i) {
			return i
		}
	}
	return -1
}

func (f *slvFile) addContent(i int, v string) bool {
	return f.replaceToken(i, f.tokenContent(i)+v)
}

func (f *slvFile) addContentBefore(i int, v string) bool {
	return f.replaceToken(i, v+f.tokenContent(i))
}

// traitUseScope returns the "{" and "}" of a trait use block, or -1, -1.
func traitUseScope(f *slvFile, use int) (int, int) {
	for j := use + 1; j < len(f.toks); j++ {
		if f.is(j, ";") {
			return -1, -1
		}
		if f.is(j, "{") {
			if f.toks[j].closer < 0 {
				return -1, -1
			}
			return j, f.toks[j].closer
		}
	}
	return -1, -1
}

func traitUseDeclaration(f *slvFile, ptr int) int {
	opener := f.enclosingScope(ptr)
	if opener < 0 {
		return 0
	}
	ooToken := f.braceOwner(opener)
	if ooToken < 0 {
		return 0
	}
	switch strings.ToLower(f.toks[ooToken].orig) {
	case "class", "interface", "trait", "enum":
	default:
		return 0
	}
	ooCloser := f.toks[opener].closer
	line := func(i int) int { return f.toks[i].line }
	all := len(f.toks)
	notWS := func(i int) bool { return f.isWS(i) }

	useTokens := []int{ptr}
	for i := ptr + 1; i < ooCloser; i++ {
		if f.isKeyword(i, "use") {
			useTokens = append(useTokens, i)
		}
		if f.is(i, "{") && f.toks[i].closer >= 0 {
			i = f.toks[i].closer
		}
	}

	for usePos, useToken := range useTokens {
		if usePos == 0 {
			lastValidContent := useToken
			for i := useToken - 1; i > opener; i-- {
				if f.isWS(i) && (line(i-1) == line(i) || line(i+1) == line(i)) {
					continue
				}
				if f.isCommentToken(i) {
					i = f.docOpener(i)
					lastValidContent = i
					continue
				}
				break
			}
			if line(lastValidContent) != line(opener)+1 {
				prev := f.prevNot(f.isEmpty, useToken-1, opener-1)
				if line(prev) == line(opener) {
					f.beginChangeset()
					for i := useToken - 1; i > opener; i-- {
						if line(i) == line(opener) {
							break
						}
						if line(i) == line(useToken) {
							continue
						}
						if f.isWS(i) && line(i-1) != line(i) && line(i+1) != line(i) {
							f.replaceToken(i, "")
						}
						if f.isCommentToken(i) {
							i = f.docOpener(i)
						}
					}
					f.endChangeset()
				}
			}
		} else {
			prev := f.prevNot(f.isEmpty, useToken-1, -1)
			if prev >= 0 && line(prev) == line(useToken) {
				prevNonWs := f.prevNot(notWS, useToken-1, -1)
				if prevNonWs == prev {
					f.beginChangeset()
					for x := useToken - 1; x > prev; x-- {
						if line(x) == line(useToken) {
							continue
						}
						f.replaceToken(x, "")
					}
					f.addContent(prev, f.eol)
					if line(prev) == line(useToken) {
						if f.isWS(useToken - 1) {
							f.replaceToken(useToken-1, "")
						}
						f.addContent(prev, strings.Repeat(" ", f.toks[useTokens[0]].column-1))
					}
					f.endChangeset()
				}
			}
		}

		phpcsOneSpaceAfter(f, useToken, all)

		var end int
		if useOpen, useClose := traitUseScope(f, useToken); useOpen >= 0 {
			traitUseGroup(f, useToken, useOpen, useClose)
			end = useClose
		} else {
			traitUseStatement(f, useToken)
			end = -1
			for j := useToken + 1; j < all; j++ {
				if f.is(j, ";") {
					end = j
					break
				}
			}
			if end < 0 {
				return 0
			}
		}

		if usePos != len(useTokens)-1 {
			continue
		}
		next := f.nextNot(notWS, end+1, all)
		if next == ooCloser {
			if line(ooCloser) > line(end)+1 {
				f.beginChangeset()
				for i := end + 1; i < ooCloser; i++ {
					if line(i) == line(end) {
						continue
					}
					if line(i) == line(ooCloser) {
						break
					}
					f.replaceToken(i, "")
				}
				f.endChangeset()
			}
		} else if next >= 0 && !f.isKeyword(next, "use") {
			for next = end + 1; next < ooCloser; next++ {
				if f.isWS(next) {
					continue
				}
				if f.isCommentToken(next) && line(next) == line(end) {
					continue
				}
				break
			}
			if line(next) <= line(end)+1 {
				if line(next) == line(useToken) {
					f.addContentBefore(next, f.eol+f.eol)
				} else {
					i := next - 1
					for ; i > end; i-- {
						if line(i) != line(next) {
							break
						}
					}
					f.addContentBefore(i+1, f.eol)
				}
			}
		}
	}
	return ooCloser
}

// phpcsOneSpaceAfter is the "1 space after" check UseDeclaration runs after
// "use", ",", "insteadof" and "as"; next whitespace is searched before end.
func phpcsOneSpaceAfter(f *slvFile, i, end int) {
	if !f.isWS(i + 1) {
		f.addContent(i, " ")
		return
	}
	if f.toks[i+1].content == " " {
		return
	}
	next := f.nextNot(f.isWS, i+1, end)
	if next >= 0 && f.toks[next].line != f.toks[i].line {
		f.beginChangeset()
		for x := i + 1; x < next; x++ {
			f.replaceToken(x, "")
		}
		f.addContent(i, " ")
		f.endChangeset()
		return
	}
	f.replaceToken(i+1, " ")
}

// traitUseKeywordSpacing checks one space around "insteadof" or "as" at i.
func traitUseKeywordSpacing(f *slvFile, i, opener, closer int) {
	if !f.isWS(i - 1) {
		f.addContentBefore(i, " ")
	} else if f.toks[i-1].content != " " {
		prev := f.prevNot(f.isWS, i-1, opener)
		newline := f.toks[prev].line != f.toks[i].line
		if f.prevNot(f.isEmpty, i-1, opener) == prev {
			if newline {
				f.beginChangeset()
				for x := i - 1; x > prev; x-- {
					f.replaceToken(x, "")
				}
				f.addContentBefore(i, " ")
				f.endChangeset()
			} else {
				f.replaceToken(i-1, " ")
			}
		}
	}

	if !f.isWS(i + 1) {
		f.addContent(i, " ")
	} else if f.toks[i+1].content != " " {
		next := f.nextNot(f.isWS, i+1, closer)
		newline := f.toks[next].line != f.toks[i].line
		if f.nextNot(f.isEmpty, i+1, closer) == next {
			if newline {
				f.beginChangeset()
				for x := i + 1; x < next; x++ {
					f.replaceToken(x, "")
				}
				f.addContent(i, " ")
				f.endChangeset()
			} else {
				f.replaceToken(i+1, " ")
			}
		}
	}
}

func traitUseGroup(f *slvFile, ptr, opener, closer int) {
	line := func(i int) int { return f.toks[i].line }
	notWS := func(i int) bool { return f.isWS(i) }

	if line(opener) != line(ptr) {
		canFix := true
		for i := ptr + 1; i < opener; i++ {
			if line(i) != line(i+1) && !f.isWS(i) {
				canFix = false
				break
			}
		}
		if canFix {
			f.beginChangeset()
			for i := ptr + 1; i < opener; i++ {
				if line(i) != line(i+1) {
					f.replaceToken(i, " ")
				}
			}
			f.endChangeset()
		}
	}

	if !f.isWS(opener - 1) {
		f.addContentBefore(opener, " ")
	} else if f.toks[opener-1].content != " " {
		prev := f.prevNot(notWS, opener-1, -1)
		if line(prev) != line(opener) {
			f.beginChangeset()
			for x := opener - 1; x > prev; x-- {
				f.replaceToken(x, "")
			}
			f.addContentBefore(opener, " ")
			f.endChangeset()
		} else {
			f.replaceToken(opener-1, " ")
		}
	}

	if next := f.nextNot(f.isEmpty, opener+1, closer-1); next >= 0 && line(next) != line(opener)+1 {
		if f.nextNot(notWS, opener+1, closer-1) == next {
			f.beginChangeset()
			for x := opener + 1; x < next; x++ {
				if line(x) == line(next) {
					break
				}
				f.replaceToken(x, "")
			}
			f.addContent(opener, f.eol)
			f.endChangeset()
		}
	}

	for i := ptr + 1; i < opener; i++ {
		if !f.is(i, ",") {
			continue
		}
		if f.isWS(i - 1) {
			f.replaceToken(i-1, "")
		}
		phpcsOneSpaceAfter(f, i, opener)
	}

	for i := opener + 1; i < closer; i++ {
		if f.isKeyword(i, "insteadof") || f.isKeyword(i, "as") {
			traitUseKeywordSpacing(f, i, opener, closer)
		}
		if f.is(i, ";") {
			if f.isWS(i - 1) {
				f.replaceToken(i-1, "")
			}
			next := f.nextNot(f.isEmpty, i+1, closer-1)
			if next >= 0 && line(next) == line(i) && f.nextNot(notWS, i+1, closer-1) == next {
				f.beginChangeset()
				if f.isWS(i + 1) {
					f.replaceToken(i+1, "")
				}
				f.addContent(i, f.eol)
				f.endChangeset()
			}
		}
	}

	if prev := f.prevNot(f.isEmpty, closer-1, opener+1); prev >= 0 && line(prev) != line(closer)-1 {
		if f.prevNot(notWS, closer-1, opener+1) == prev {
			f.beginChangeset()
			for x := closer - 1; x > prev; x-- {
				if line(x) == line(closer) {
					continue
				}
				f.replaceToken(x, "")
			}
			f.addContent(prev, f.eol)
			f.endChangeset()
		}
	}
}

func traitUseStatement(f *slvFile, ptr int) {
	for j := ptr + 1; j < len(f.toks); j++ {
		if f.is(j, ";") {
			return
		}
		if f.is(j, ",") {
			f.replaceToken(j, ";"+f.eol+strings.Repeat(" ", f.toks[ptr].column-1)+"use ")
			return
		}
	}
}
