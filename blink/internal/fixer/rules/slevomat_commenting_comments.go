package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// cmdIsComment tells PHP_CodeSniffer's T_COMMENT, which a "phpcs:" annotation
// is not.
func (f *slvFile) cmdIsComment(i int) bool {
	return f.cmdIsInlineComment(i) && !slvIsPhpcsAnnotation(f.toks[i].content)
}

// cmdIsInlineComment tells TokenHelper::INLINE_COMMENT_TOKEN_CODES outside a
// docblock.
func (f *slvFile) cmdIsInlineComment(i int) bool {
	return f.toks[i].kind == token.Comment && f.toks[i].doc == slvDocNone
}

// cmdIsDocAnnotation tells a "phpcs:" annotation inside a docblock, which
// PHP_CodeSniffer gives the comment_closer of its docblock.
func (f *slvFile) cmdIsDocAnnotation(i int) bool {
	return f.toks[i].doc == slvDocString && slvIsPhpcsAnnotation(f.toks[i].content)
}

var cmdLineComment = regexp.MustCompile(`^(?://|#)(.*)`)

// cmdIsLineComment is CommentHelper::isLineComment.
func (f *slvFile) cmdIsLineComment(i int) bool {
	return cmdLineComment.MatchString(f.toks[i].content)
}

// cmdCommentEnd is CommentHelper::getCommentEndPointer, -1 for null.
func (f *slvFile) cmdCommentEnd(start int) int {
	if f.isDocOpen(start) || f.cmdIsDocAnnotation(start) {
		return f.docCloser(start)
	}
	if f.cmdIsLineComment(start) {
		return start
	}
	if !strings.HasPrefix(f.toks[start].content, "/*") {
		return -1
	}
	end := start
	for j := start + 1; j < len(f.toks) && f.cmdIsInlineComment(j); j++ {
		end = j
	}
	return end
}

// cmdTrimTrailing removes the characters of set at the end of s like a
// preg_replace of "~[set]+$~", whose "$" also matches before a final newline;
// it reports whether anything was removed.
func cmdTrimTrailing(s, set string) (string, bool) {
	body, nl := s, ""
	if strings.HasSuffix(s, "\n") && !strings.ContainsRune(set, '\n') {
		body, nl = s[:len(s)-1], "\n"
	}
	trimmed := strings.TrimRight(body, set)
	if trimmed == body {
		return s, false
	}
	return trimmed + nl, true
}

const cmdPhpTrimSet = " \t\n\r\x00\x0b"

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/EmptyCommentSniff.php
//
// SlevomatEmptyComment removes empty comments.
type SlevomatEmptyComment struct{}

func (SlevomatEmptyComment) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\EmptyCommentSniff`
}

func (SlevomatEmptyComment) SourceURL() string {
	return fixer.SourceURLFor(SlevomatEmptyComment{}.Name())
}

func (r SlevomatEmptyComment) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.isDocOpen(i) || f.cmdIsComment(i) }, func(i int) int {
			r.process(f, i)
			return 0
		})
	})
}

var (
	cmdEmptyLine  = regexp.MustCompile(`^[\t\n\v\f\r ]*$`)
	cmdEmptyBlock = regexp.MustCompile(`^[\t\n\v\f\r *]*$`)
)

func (SlevomatEmptyComment) isEmpty(comment string, line bool) bool {
	if line {
		return cmdEmptyLine.MatchString(comment)
	}
	return cmdEmptyBlock.MatchString(comment)
}

func (SlevomatEmptyComment) commentContent(f *slvFile, start, end int) string {
	if f.isDocOpen(start) {
		return f.content(start+1, end-1)
	}
	if m := cmdLineComment.FindStringSubmatch(f.toks[start].content); m != nil {
		return m[1]
	}
	c := f.content(start, end)
	if len(c) < 4 {
		return ""
	}
	return c[2 : len(c)-2]
}

func (r SlevomatEmptyComment) process(f *slvFile, start int) {
	end := f.cmdCommentEnd(start)
	if end < 0 {
		return
	}
	line := f.cmdIsLineComment(start)
	if !r.isEmpty(r.commentContent(f, start, end), line) {
		return
	}
	if line && r.isNonEmptyLineCommentBefore(f, start) && r.isNonEmptyLineCommentAfter(f, end) {
		return
	}

	before := f.prevNonWhitespace(start - 1)
	whitespace := ""
	if before != start-1 {
		whitespace = f.content(before+1, start-1)
	}
	fixedWhitespace, _ := cmdTrimTrailing(whitespace, " \t")

	f.beginChangeset()
	for j := before + 1; j < start; j++ {
		f.replaceToken(j, "")
	}
	f.add(before, fixedWhitespace)
	for j := start; j <= end; j++ {
		f.replaceToken(j, "")
	}
	after := end + 1
	if f.toks[before].line == f.toks[start].line {
		if strings.HasSuffix(f.toks[end].content, f.eol) {
			f.replaceToken(end, f.tokenContent(end)+f.eol)
		}
	} else if after < len(f.toks) && f.toks[after].kind == token.Whitespace {
		content := f.toks[after].content
		ws := strings.TrimLeft(content, " \t")
		if strings.HasPrefix(ws, f.eol) {
			content = ws[len(f.eol):]
		}
		f.replace(after, content)
	}
	f.endChangeset()
}

func (r SlevomatEmptyComment) isNonEmptyLineCommentBefore(f *slvFile, start int) bool {
	before := f.prevNonWhitespace(start - 1)
	if before < 0 || !f.cmdIsComment(before) || !f.cmdIsLineComment(before) {
		return false
	}
	if f.toks[before].line+1 != f.toks[start].line {
		return false
	}
	if !r.isEmpty(r.commentContent(f, before, f.cmdCommentEnd(before)), true) {
		return true
	}
	return r.isNonEmptyLineCommentBefore(f, before)
}

func (r SlevomatEmptyComment) isNonEmptyLineCommentAfter(f *slvFile, end int) bool {
	after := f.nextNonWhitespace(end+1, len(f.toks))
	if after < 0 || !f.cmdIsComment(after) || !f.cmdIsLineComment(after) {
		return false
	}
	if f.toks[end].line+1 != f.toks[after].line {
		return false
	}
	afterEnd := f.cmdCommentEnd(after)
	if !r.isEmpty(r.commentContent(f, after, afterEnd), true) {
		return true
	}
	return r.isNonEmptyLineCommentAfter(f, afterEnd)
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/DisallowCommentAfterCodeSniff.php
//
// SlevomatDisallowCommentAfterCode moves a comment after code to its own line
// before the code.
type SlevomatDisallowCommentAfterCode struct{}

func (SlevomatDisallowCommentAfterCode) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\DisallowCommentAfterCodeSniff`
}

func (SlevomatDisallowCommentAfterCode) SourceURL() string {
	return fixer.SourceURLFor(SlevomatDisallowCommentAfterCode{}.Name())
}

func (r SlevomatDisallowCommentAfterCode) Fix(s *tokens.Stream) bool {
	return slvRun(s, func(f *slvFile) {
		f.eachToken(func(i int) bool { return f.cmdIsInlineComment(i) || f.isDocOpen(i) || f.cmdIsDocAnnotation(i) }, func(i int) int {
			r.process(f, i)
			return 0
		})
	})
}

// cmdIsStringToken tells Tokens::STRING_TOKENS.
func (f *slvFile) cmdIsStringToken(i int) bool {
	return f.toks[i].kind == token.String && !f.toks[i].heredoc
}

func (r SlevomatDisallowCommentAfterCode) process(f *slvFile, comment int) {
	if f.toks[comment].column == 1 {
		return
	}
	firstOnLine := f.nsuFirstNonWhitespaceOnLine(comment)
	if firstOnLine == comment {
		return
	}
	if f.isDocOpen(firstOnLine) && f.docCloser(firstOnLine) > comment {
		return
	}
	end := f.cmdCommentEnd(comment)
	if end < 0 {
		// part of a block comment, which always starts a line
		return
	}
	if next := f.nextNonWhitespace(end+1, len(f.toks)); next >= 0 && f.toks[next].line == f.toks[end].line {
		return
	}

	content := f.content(comment, end)
	hasNewLine := strings.HasSuffix(content, f.eol)
	if !hasNewLine {
		content += f.eol
	}
	before := f.prevNonWhitespace(comment - 1)
	newLineAfter := end
	if !hasNewLine {
		newLineAfter = f.lastOnLine(end)
	}
	indentation := f.indentation(firstOnLine)
	firstPointerOnLine := f.firstOnLine(firstOnLine)

	f.beginChangeset()
	switch {
	case f.is(before, "{") && r.opensElseOrClosure(f, before):
		f.add(before, f.eol+indentation+strings.Repeat(" ", slvTabWidth)+content)
	case f.is(firstOnLine, "}"):
		f.add(before, f.eol+indentation+content)
	case f.cmdIsStringToken(firstPointerOnLine):
		prev := firstPointerOnLine - 1
		for prev >= 0 && (f.toks[prev].kind == token.Whitespace || f.cmdIsStringToken(prev)) {
			prev--
		}
		prevIndentation := f.indentation(f.nsuFirstNonWhitespaceOnLine(prev))
		f.addBefore(f.firstOnLine(prev), prevIndentation+content)
		f.replaceToken(before, f.tokenContent(before)+f.eol)
	default:
		f.addBefore(firstPointerOnLine, indentation+content)
		f.replaceToken(before, f.tokenContent(before)+f.eol)
	}
	for j := before + 1; j <= newLineAfter; j++ {
		f.replaceToken(j, "")
	}
	f.endChangeset()
}

func (SlevomatDisallowCommentAfterCode) opensElseOrClosure(f *slvFile, open int) bool {
	if f.toks[open].closer < 0 {
		return false
	}
	owner := f.scopeCondition(open)
	if owner < 0 {
		return false
	}
	return f.isKeyword(owner, "elseif") || f.isKeyword(owner, "else") ||
		f.isKeyword(owner, "function") && f.isClosure(owner)
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/ForbiddenAnnotationsSniff.php
//
// SlevomatForbiddenAnnotations removes the configured annotations from doc
// comments.
type SlevomatForbiddenAnnotations struct {
	forbidden []string
}

func (SlevomatForbiddenAnnotations) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\ForbiddenAnnotationsSniff`
}

func (SlevomatForbiddenAnnotations) SourceURL() string {
	return fixer.SourceURLFor(SlevomatForbiddenAnnotations{}.Name())
}

func (r SlevomatForbiddenAnnotations) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := slvStringList(config["forbiddenAnnotations"]); ok {
		r.forbidden = list
	}
	return r
}

func (r SlevomatForbiddenAnnotations) Fix(s *tokens.Stream) bool {
	if len(r.forbidden) == 0 {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		d := newSlvDocs(f)
		f.eachToken(f.isDocOpen, func(i int) int {
			r.process(f, d, i)
			return 0
		})
	})
}

func (f *slvFile) cmdIsDoc(i int, kind slvDocKind) bool {
	return f.toks[i].kind == token.DocComment && f.toks[i].doc == kind
}

func (r SlevomatForbiddenAnnotations) process(f *slvFile, d *slvDocs, open int) {
	p := d.parse(open)
	if p == nil {
		return
	}
	closeIdx := f.docCloser(open)
	for _, a := range p.annotations {
		if !slices.Contains(r.forbidden, a.name) {
			continue
		}
		annStart, annEnd := f.thAnnotationPointers(p, a.node)

		star := -1
		for j := annStart - 1; j >= open; j-- {
			if f.cmdIsDoc(j, slvDocStar) {
				star = j
				break
			}
		}
		start := annStart
		if star >= 0 {
			start = star
		}

		next := -1
		for j := annEnd + 1; j < len(f.toks); j++ {
			if f.cmdIsDoc(j, slvDocTag) || f.cmdIsDoc(j, slvDocClose) {
				next = j
				break
			}
		}
		if next < 0 {
			continue
		}
		if f.cmdIsDoc(next, slvDocTag) {
			for j := next - 1; j >= 0; j-- {
				if f.cmdIsDoc(j, slvDocStar) {
					next = j
					break
				}
			}
		}
		end := next - 1

		if f.cmdIsDoc(next, slvDocClose) && star >= 0 {
			before := start - 1
			for before >= 0 && (f.cmdIsDoc(before, slvDocWhitespace) || f.cmdIsDoc(before, slvDocStar)) {
				before--
			}
			for j := before + 1; j < len(f.toks); j++ {
				if f.cmdIsDoc(j, slvDocStar) {
					start = j
					break
				}
			}
		}

		f.beginChangeset()
		for j := start; j <= end; j++ {
			f.replaceToken(j, "")
		}
		useful := false
		for j := open + 1; j < closeIdx; j++ {
			if c := strings.Trim(f.tokenContent(j), cmdPhpTrimSet); c != "" && c != "*" {
				useful = true
				break
			}
		}
		if !useful {
			after := f.nextEffective(closeIdx+1, len(f.toks))
			for j := open; j <= after-1; j++ {
				f.replaceToken(j, "")
			}
		}
		f.endChangeset()
	}
}

// Slevomat: https://github.com/slevomat/coding-standard/blob/master/SlevomatCodingStandard/Sniffs/Commenting/ForbiddenCommentsSniff.php
//
// SlevomatForbiddenComments removes the parts of doc comment descriptions the
// configured patterns match.
type SlevomatForbiddenComments struct {
	patterns []*regexp.Regexp
	invalid  bool
}

func (SlevomatForbiddenComments) Name() string {
	return `SlevomatCodingStandard\Sniffs\Commenting\ForbiddenCommentsSniff`
}

func (SlevomatForbiddenComments) SourceURL() string {
	return fixer.SourceURLFor(SlevomatForbiddenComments{}.Name())
}

func (r SlevomatForbiddenComments) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := slvStringList(config["forbiddenCommentPatterns"]); ok {
		r.patterns, r.invalid = nil, false
		for _, p := range list {
			re := slvlCompilePattern(p)
			if re == nil {
				// upstream throws on an invalid pattern, so ECS fixes nothing
				r.invalid = true
				continue
			}
			r.patterns = append(r.patterns, re)
		}
	}
	return r
}

func (r SlevomatForbiddenComments) Fix(s *tokens.Stream) bool {
	if r.invalid || len(r.patterns) == 0 {
		return false
	}
	return slvRun(s, func(f *slvFile) {
		f.eachToken(f.isDocOpen, func(i int) int {
			r.process(f, i)
			return 0
		})
	})
}

type cmdComment struct {
	pointer int
	content string
}

// cmdDescription is DocCommentHelper::getDocCommentDescription of a doc
// comment opener.
func (f *slvFile) cmdDescription(open int) []cmdComment {
	closeIdx := f.docCloser(open)
	start := -1
	for j := open + 1; j < closeIdx; j++ {
		if !f.cmdIsDoc(j, slvDocWhitespace) && !f.cmdIsDoc(j, slvDocStar) {
			start = j
			break
		}
	}
	if start < 0 || !f.cmdIsDoc(start, slvDocString) {
		return nil
	}
	after := closeIdx + 1
	for j := start + 1; j < closeIdx+1; j++ {
		if f.cmdIsDoc(j, slvDocTag) || f.cmdIsDoc(j, slvDocClose) {
			after = j
			break
		}
	}
	var out []cmdComment
	for j := start; j < after; j++ {
		if f.cmdIsDoc(j, slvDocString) {
			out = append(out, cmdComment{j, strings.Trim(f.toks[j].content, cmdPhpTrimSet)})
		}
	}
	return out
}

var cmdLeadingNewlines = regexp.MustCompile(`^[\r\n]+`)

func (r SlevomatForbiddenComments) process(f *slvFile, open int) {
	comments := f.cmdDescription(open)
	if comments == nil {
		return
	}
	closeIdx := f.docCloser(open)
	for _, re := range r.patterns {
		for _, c := range comments {
			if !re.MatchString(c.content) {
				continue
			}
			f.beginChangeset()
			f.replace(c.pointer, re.ReplaceAllLiteralString(c.content, ""))
			for j := c.pointer - 1; j > open; j-- {
				v, replaced := cmdTrimTrailing(f.toks[j].content, " ")
				if !replaced {
					break
				}
				f.replace(j, v)
			}

			var b strings.Builder
			for j := open + 1; j < closeIdx; j++ {
				b.WriteString(f.tokenContent(j))
			}
			if cmdEmptyBlock.MatchString(b.String()) {
				if before := open - 1; before >= 0 {
					if v, replaced := cmdTrimTrailing(f.toks[before].content, " \t"); replaced {
						f.replace(before, v)
					}
				}
				for j := open; j <= closeIdx; j++ {
					f.replaceToken(j, "")
				}
				if after := closeIdx + 1; after < len(f.toks) {
					if v := cmdLeadingNewlines.ReplaceAllString(f.toks[after].content, ""); v != f.toks[after].content {
						f.replace(after, v)
					}
				}
			}
			f.endChangeset()
		}
	}
}
