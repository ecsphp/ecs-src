package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Commenting/FunctionCommentSniff.php
//
// PEARFunctionComment removes blank lines between a function's docblock and
// its declaration and aligns the types, names and comments of its @param tags.
type PEARFunctionComment struct{}

func (PEARFunctionComment) Name() string {
	return `PHP_CodeSniffer\Standards\PEAR\Sniffs\Commenting\FunctionCommentSniff`
}

func (PEARFunctionComment) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PEAR/Sniffs/Commenting/FunctionCommentSniff.php"
}

func (PEARFunctionComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		for i := range f.toks {
			if phpcsIsNamedFunction(f, i) {
				pearFunctionComment(f, i)
			}
		}
	})
}

// phpcsIsNamedFunction tells T_FUNCTION from a closure (T_CLOSURE) and the
// "function" of a "use function" import (T_STRING).
func phpcsIsNamedFunction(f *slvFile, i int) bool {
	if !f.isKeyword(i, "function") || f.isClosure(i) {
		return false
	}
	if phpcsIsUse(f, f.prevEffective(i-1)) {
		return false
	}
	e := f.enclosing(i)
	return e < 0 || !phpcsIsUseGroupOpen(f, e)
}

func pearIsMethodModifier(f *slvFile, i int) bool {
	if f.toks[i].kind != token.Keyword {
		return false
	}
	switch strings.ToLower(f.toks[i].orig) {
	case "private", "public", "protected", "abstract", "static", "final":
		return true
	}
	return false
}

func pearFunctionComment(f *slvFile, ptr int) {
	ignored := func(i int) bool { return f.toks[i].kind == token.Whitespace || pearIsMethodModifier(f, i) }
	commentEnd := ptr - 1
	for ; commentEnd >= 0; commentEnd-- {
		if ignored(commentEnd) {
			continue
		}
		if t := f.toks[commentEnd]; t.kind == token.AttributeEnd && t.closer >= 0 {
			commentEnd = t.closer
			continue
		}
		break
	}
	if commentEnd < 0 {
		return
	}
	isComment := func(i int) bool { return f.toks[i].kind == token.Comment && !phpcsIsAnnotation(f, i) }
	if isComment(commentEnd) {
		prev := commentEnd - 1
		for prev >= 0 && ignored(prev) {
			prev--
		}
		if prev >= 0 && f.toks[prev].line == f.toks[commentEnd].line {
			commentEnd = prev
		}
	}
	if f.toks[commentEnd].doc != slvDocClose {
		return
	}

	if f.toks[commentEnd].line != f.toks[ptr].line-1 {
		for i := commentEnd + 1; i < ptr; i++ {
			if c := phpcsAttributeCloser(f, i); c >= 0 {
				i = c
				continue
			}
			if f.toks[i].column != 1 || f.toks[i].kind != token.Whitespace || f.toks[i].line == f.toks[i+1].line ||
				f.toks[i-1].kind == token.Comment && phpcsIsAnnotation(f, i-1) {
				continue
			}
			next := f.nextNonWhitespace(i+1, len(f.toks))
			f.beginChangeset()
			for j := i; j < next; j++ {
				if f.toks[j].line == f.toks[next].line {
					break
				}
				f.replaceToken(j, "")
			}
			f.endChangeset()
			i = next
		}
	}

	commentStart := commentEnd
	for commentStart > 0 && f.toks[commentStart].doc != slvDocOpen {
		commentStart--
	}
	var tags []int
	for i := commentStart + 1; i < commentEnd; i++ {
		if f.toks[i].doc == slvDocTag {
			tags = append(tags, i)
		}
	}
	pearProcessParams(f, commentEnd, tags)
}

// phpcsAttributeCloser is the attribute_closer PHP_CodeSniffer gives every
// token of an attribute, or -1 outside one.
func phpcsAttributeCloser(f *slvFile, i int) int {
	switch f.toks[i].kind {
	case token.AttributeStart:
		return f.toks[i].closer
	case token.AttributeEnd:
		if f.toks[i].closer >= 0 {
			return i
		}
	}
	for e := f.enclosing(i); e >= 0; e = f.enclosing(e) {
		if f.toks[e].kind == token.AttributeStart {
			return f.toks[e].closer
		}
	}
	return -1
}

type pearParam struct {
	tag                 int
	typ, varName        string
	comment             string
	commentEnd          int
	commentTokens       []int
	typeSpace, varSpace int
}

func pearProcessParams(f *slvFile, commentCloser int, tags []int) {
	isString := func(i int) bool { return i < len(f.toks) && f.toks[i].doc == slvDocString }
	var params []pearParam
	maxType, maxVar := 0, 0
	for pos, tag := range tags {
		if f.toks[tag].content != "@param" {
			continue
		}
		p := pearParam{tag: tag}
		if isString(tag + 2) {
			m := pearParseParam(f.toks[tag+2].content)
			typeLen := len(m.typ)
			p.typ = strings.Trim(m.typ, " \t\n\r\x00\x0B")
			p.typeSpace = typeLen - len(p.typ)
			maxType = max(maxType, len(p.typ))
			if m.hasVar {
				p.varName = m.varName
				maxVar = max(maxVar, len(p.varName))
				if m.hasComment {
					p.varSpace = len(m.space)
					p.comment = m.comment
					end := commentCloser
					if pos+1 < len(tags) {
						end = tags[pos+1]
					}
					for i := tag + 3; i < end; i++ {
						if isString(i) {
							p.comment += " " + f.toks[i].content
							p.commentEnd = i
							p.commentTokens = append(p.commentTokens, i)
						}
					}
				}
			}
		}
		params = append(params, p)
	}

	for _, p := range params {
		if p.varName == "" {
			continue
		}
		if strings.Trim(p.typ, " \t\n\r\x00\x0B") != "" {
			if spaces := maxType - len(p.typ) + 1; p.typeSpace != spaces {
				pearRewriteParam(f, p, p.typ+strings.Repeat(" ", spaces)+p.varName+strings.Repeat(" ", p.varSpace))
			}
		}
		if p.comment == "" {
			continue
		}
		if spaces := maxVar - len(p.varName) + 1; p.varSpace != spaces {
			pearRewriteParam(f, p, p.typ+strings.Repeat(" ", p.typeSpace)+p.varName+strings.Repeat(" ", spaces))
		}
		if p.tag == p.commentEnd {
			continue
		}
		ct := f.toks[p.tag+2]
		wrapLength := ct.length - p.typeSpace - p.varSpace - len(p.typ) - len(p.varName)
		startColumn := ct.column + ct.length - wrapLength
		star := pearPrevStar(f, p.tag)
		padding := strings.Repeat(" ", max(startColumn-f.toks[star].column-1, 0))
		for _, c := range p.commentTokens {
			if f.toks[c].column == startColumn {
				continue
			}
			if f.toks[c-1].doc == slvDocWhitespace {
				f.replaceToken(c-1, padding)
			} else {
				f.addContentBefore(c, padding)
			}
		}
	}
}

func pearRewriteParam(f *slvFile, p pearParam, content string) {
	ct := p.tag + 2
	wrapLength := f.toks[ct].length - p.typeSpace - p.varSpace - len(p.typ) - len(p.varName)
	star := pearPrevStar(f, p.tag)
	spaceLength := len(content) + f.toks[ct-1].length + f.toks[ct-2].length
	padding := strings.Repeat(" ", f.toks[star].column-1) + "* " + strings.Repeat(" ", spaceLength)
	f.replaceToken(ct, content+phpWordwrap(p.comment, wrapLength, f.eol+padding))
	for i := ct + 1; i <= p.commentEnd; i++ {
		f.replaceToken(i, "")
	}
}

// pearPrevStar is findPrevious(T_DOC_COMMENT_STAR, tag), with PHP reading
// $tokens[false] as $tokens[0].
func pearPrevStar(f *slvFile, tag int) int {
	for i := tag; i >= 0; i-- {
		if f.toks[i].doc == slvDocStar {
			return i
		}
	}
	return 0
}

type pearParamMatch struct {
	typ, varName, space, comment string
	hasVar, hasComment           bool
}

// pearParseParam matches the sniff's
// /((?:(?![$.]|&(?=\$)).)*)(?:((?:\.\.\.)?(?:\$|&)[^\s]+)(?:(\s+)(.*))?)?/.
func pearParseParam(s string) pearParamMatch {
	var m pearParamMatch
	isSpace := func(c byte) bool { return strings.IndexByte(" \t\n\v\f\r", c) >= 0 }
	i := 0
	for i < len(s) && s[i] != '\n' && s[i] != '$' && s[i] != '.' && (s[i] != '&' || i+1 >= len(s) || s[i+1] != '$') {
		i++
	}
	m.typ = s[:i]
	j := i
	if strings.HasPrefix(s[j:], "...") && j+3 < len(s) && (s[j+3] == '$' || s[j+3] == '&') {
		j += 3
	}
	if j >= len(s) || s[j] != '$' && s[j] != '&' {
		return m
	}
	k := j + 1
	for k < len(s) && !isSpace(s[k]) {
		k++
	}
	if k == j+1 {
		return m
	}
	m.hasVar, m.varName = true, s[i:k]
	e := k
	for e < len(s) && isSpace(s[e]) {
		e++
	}
	if e == k {
		return m
	}
	rest := s[e:]
	if n := strings.IndexByte(rest, '\n'); n >= 0 {
		rest = rest[:n]
	}
	m.hasComment, m.space, m.comment = true, s[k:e], rest
	return m
}

// phpWordwrap is PHP's wordwrap($text, $width, $break) without $cut.
func phpWordwrap(text string, width int, brk string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	laststart, lastspace := 0, 0
	current := 0
	for ; current < len(text); current++ {
		switch {
		case text[current] == brk[0] && current+len(brk) < len(text) && strings.HasPrefix(text[current:], brk):
			b.WriteString(text[laststart : current+len(brk)])
			current += len(brk) - 1
			laststart = current + 1
			lastspace = laststart
		case text[current] == ' ':
			if current-laststart >= width {
				b.WriteString(text[laststart:current])
				b.WriteString(brk)
				laststart = current + 1
			}
			lastspace = current
		case current-laststart >= width && laststart < lastspace:
			b.WriteString(text[laststart:lastspace])
			b.WriteString(brk)
			laststart = lastspace + 1
			lastspace = laststart
		}
	}
	if laststart != current {
		b.WriteString(text[laststart:])
	}
	return b.String()
}
