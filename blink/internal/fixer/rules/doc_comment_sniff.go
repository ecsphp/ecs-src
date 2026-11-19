package rules

import (
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Commenting/DocCommentSniff.php
//
// GenericDocComment lays out docblocks: the open and close tags on their own
// lines, the short description on the first line, one blank line between the
// descriptions, before the tags and after each tag group, and tag values
// aligned within a group.
type GenericDocComment struct{}

func (GenericDocComment) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Commenting\DocCommentSniff`
}

func (GenericDocComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Commenting/DocCommentSniff.php"
}

func (GenericDocComment) Fix(s *tokens.Stream) bool {
	if !slices.ContainsFunc(s.Tokens(), slvIsDocBlock) {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if f.toks[i].doc == slvDocOpen {
				docCommentProcess(f, i)
			}
		}
	})
}

// phpcsDocAnnotation reports whether PHP_CodeSniffer retypes a docblock string as
// a phpcs: annotation token, which it does even with annotations off.
func phpcsDocAnnotation(v string) bool {
	text := strings.ToLower(strings.TrimRight(strings.TrimLeft(v, " \t/*#"), " */\t\r\n"))
	text = strings.TrimPrefix(text, "@")
	for _, p := range []string{"phpcs:set", "phpcs:ignorefile", "phpcs:disable", "phpcs:enable", "phpcs:ignore"} {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

func docCommentProcess(f *slvFile, start int) {
	end := start + 1
	for end < len(f.toks) && f.toks[end].doc != slvDocClose {
		end++
	}
	if end == len(f.toks) || f.toks[end].orig == "" && end == len(f.toks)-1 {
		return
	}
	line := func(i int) int { return f.toks[i].line }
	isEmpty := func(i int) bool {
		d := f.toks[i].doc
		return d == slvDocWhitespace || d == slvDocStar
	}
	isString := func(i int) bool {
		return f.toks[i].doc == slvDocString && !phpcsDocAnnotation(f.toks[i].content)
	}
	isTagOrAnnotation := func(i int) bool {
		return f.toks[i].doc == slvDocTag || f.toks[i].doc == slvDocString && phpcsDocAnnotation(f.toks[i].content)
	}
	nextNonEmpty := func(from, to int) int {
		for i := from; i < to; i++ {
			if !isEmpty(i) {
				return i
			}
		}
		return -1
	}
	prevNonEmpty := func(from int) int {
		for i := from; i >= start; i-- {
			if !isEmpty(i) {
				return i
			}
		}
		return -1
	}
	blank := func(from, to int, stop func(i int) bool) {
		for i := from; i < to; i++ {
			if stop(i) {
				break
			}
			f.replaceToken(i, "")
		}
	}
	indent := strings.Repeat(" ", f.toks[start].column)

	short := nextNonEmpty(start+1, end)
	if short < 0 {
		return
	}

	if line(short) == line(start) {
		f.beginChangeset()
		f.replaceToken(start, f.tokenContent(start)+f.eol)
		f.replaceToken(short, "* "+f.tokenContent(short))
		f.endChangeset()
	}

	prev := prevNonEmpty(end - 1)
	if line(prev) == line(end) {
		f.replaceToken(end, f.eol+f.tokenContent(end))
	}

	if line(prev) < line(end)-1 {
		f.beginChangeset()
		blank(prev+1, end, func(i int) bool { return line(i+1) == line(end) })
		f.endChangeset()
	}

	if isString(short) {
		if line(short) != line(start)+1 {
			f.beginChangeset()
			for i := start; i < short; i++ {
				if line(i) == line(start) {
					continue
				}
				if line(i) == line(short) {
					break
				}
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}

		shortEnd := short
		for i := short + 1; i < end; i++ {
			if isString(i) {
				if line(i) != line(shortEnd)+1 {
					break
				}
				shortEnd = i
			}
		}

		if long := nextNonEmpty(shortEnd+1, end-1); long >= 0 && isString(long) && line(long) != line(shortEnd)+2 {
			f.beginChangeset()
			for i := shortEnd + 1; i < long; i++ {
				if line(i) == line(shortEnd) {
					continue
				}
				if line(i) == line(long)-1 {
					break
				}
				f.replaceToken(i, "")
			}
			f.endChangeset()
		}
	}

	var tags []int
	for i := start + 1; i < end; i++ {
		if f.toks[i].doc == slvDocTag {
			tags = append(tags, i)
		}
	}
	if len(tags) == 0 {
		return
	}

	firstTag := tags[0]
	prev = prevNonEmpty(firstTag - 1)
	if line(firstTag) != line(prev)+2 && f.toks[prev].doc != slvDocOpen {
		f.beginChangeset()
		blank(prev+1, firstTag, func(i int) bool { return line(i) == line(firstTag) })
		f.replaceToken(prev, f.tokenContent(prev)+f.eol+indent+"*"+f.eol)
		f.endChangeset()
	}

	var groups [][]int
	for pos, tag := range tags {
		if pos > 0 {
			p := tags[pos-1]
			for i := tag - 1; i >= tags[pos-1]; i-- {
				if isString(i) {
					p = i
					break
				}
			}
			if line(p) != line(tag)-1 {
				groups = append(groups, nil)
			}
		}
		if len(groups) == 0 {
			groups = append(groups, nil)
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], tag)
	}

	for _, group := range groups {
		maxLength := 0
		var padded []int
		for _, tag := range group {
			maxLength = max(maxLength, f.toks[tag].length)
			for i := tag; i < end; i++ {
				if isString(i) {
					if line(i) == line(tag) {
						padded = append(padded, tag)
					}
					break
				}
			}
		}

		lastTag := group[len(group)-1]
		next := -1
		for i := lastTag + 3; i < end; i++ {
			if isTagOrAnnotation(i) {
				next = i
				break
			}
		}
		if next >= 0 {
			prev := -1
			for i := next - 1; i >= 0; i-- {
				if f.toks[i].doc == slvDocTag || isString(i) {
					prev = i
					break
				}
			}
			if prev >= 0 && line(next) != line(prev)+2 {
				f.beginChangeset()
				blank(prev+1, next, func(i int) bool { return line(i) == line(next) })
				f.replaceToken(prev, f.tokenContent(prev)+f.eol+indent+"*"+f.eol)
				f.endChangeset()
			}
		}

		for _, tag := range padded {
			if required := maxLength - f.toks[tag].length + 1; f.toks[tag+1].length != required {
				f.replaceToken(tag+1, strings.Repeat(" ", required))
			}
		}
	}
}
