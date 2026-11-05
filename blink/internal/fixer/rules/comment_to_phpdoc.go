package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/CommentToPhpdocFixer.php
//
// CommentToPhpdoc turns a "/* */", "//" or "#" comment that documents a
// structural element and carries an "@tag" into a "/** */" docblock. The comment
// is left alone when it is the file header, does not precede a structural
// element, carries only ignored tags, or carries an "@...-ignore"/"@...-suppress"
// annotation. ignoredTags mirrors the "ignored_tags" option (lowercased).
type CommentToPhpdoc struct {
	ignoredTags []string
}

func (f CommentToPhpdoc) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["ignored_tags"]; ok {
		tags := singleElementConfigStrings(raw)
		f.ignoredTags = make([]string, len(tags))
		for i, tag := range tags {
			f.ignoredTags[i] = strings.ToLower(tag)
		}
	}
	return f
}

func (CommentToPhpdoc) Name() string {
	return `PhpCsFixer\Fixer\Comment\CommentToPhpdocFixer`
}

func (CommentToPhpdoc) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/CommentToPhpdocFixer.php"
}

func (CommentToPhpdoc) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasKind(token.Comment)
}

func (f CommentToPhpdoc) Fix(s *tokens.Stream) bool {
	changed := false
	for index := 0; index < s.Len(); index++ {
		if s.At(index).Kind != token.Comment {
			continue
		}
		if docIsHeaderComment(s, index) {
			continue
		}
		if !docIsBeforeStructuralElement(s, index) {
			continue
		}
		if hasIgnoreSuppressTag(s.At(index).Value) {
			continue
		}

		indices := commentBlockIndices(s, index)
		resume := indices[len(indices)-1]
		if f.isCommentCandidate(s, indices) {
			if len(indices) == 1 {
				if f.fixCommentSingleLine(s, indices[0]) {
					changed = true
				}
			} else if f.fixCommentMultiLine(s, indices) {
				changed = true
				resume = indices[0] // block collapsed to one token at its start
			}
		}
		index = resume
	}
	return changed
}

func (f CommentToPhpdoc) isCommentCandidate(s *tokens.Stream, indices []int) bool {
	for _, index := range indices {
		if tag, ok := firstCommentTag(s.At(index).Value); ok && !slices.Contains(f.ignoredTags, tag) {
			return true
		}
	}
	return false
}

func (CommentToPhpdoc) fixCommentSingleLine(s *tokens.Stream, index int) bool {
	message := commentMessage(s.At(index).Value)
	if firstByteNotSpace(message) {
		message = " " + message
	}
	if lastByteNotSpace(message) {
		message += " "
	}
	s.Set(index, token.Token{Kind: token.DocComment, Value: "/**" + message + "*/"})
	return true
}

func (CommentToPhpdoc) fixCommentMultiLine(s *tokens.Stream, indices []int) bool {
	start := indices[0]
	indent := ""
	if start > 0 && s.At(start-1).Kind == token.Whitespace {
		indent = trailingWhitespaceIndent(s.At(start - 1).Value)
	}

	const lineEnding = "\n"
	count := indices[len(indices)-1]

	var b strings.Builder
	b.WriteString("/**")
	b.WriteString(lineEnding)
	for index := start; index <= count; index++ {
		t := s.At(index)
		if !isComment(t) {
			continue
		}
		if strings.Contains(t.Value, "*/") {
			return false // a nested close would corrupt the docblock; leave as is
		}
		message := commentMessage(t.Value)
		if firstByteNotSpace(message) {
			message = " " + message
		}
		b.WriteString(indent)
		b.WriteString(" *")
		b.WriteString(message)
		b.WriteString(lineEnding)
	}
	b.WriteString(indent)
	b.WriteString(" */")

	s.ReplaceRange(start, count, []token.Token{{Kind: token.DocComment, Value: b.String()}})
	return true
}

// commentMessage strips the comment markers, mirroring the fixer's getMessage.
func commentMessage(content string) string {
	if strings.HasPrefix(content, "#") {
		return content[1:]
	}
	if strings.HasPrefix(content, "//") {
		return content[2:]
	}
	return strings.TrimRight(strings.TrimLeft(content, "/*"), "*/")
}

func firstByteNotSpace(s string) bool {
	return s != "" && strings.TrimSpace(s[:1]) != ""
}

func lastByteNotSpace(s string) bool {
	return s != "" && strings.TrimSpace(s[len(s)-1:]) != ""
}

// trailingWhitespaceIndent ports Utils::calculateTrailingWhitespaceIndent: the
// run of whitespace after the last line break of a whitespace token.
func trailingWhitespaceIndent(v string) string {
	v = strings.ReplaceAll(v, "\r\n", "\n")
	v = strings.ReplaceAll(v, "\r", "\n")
	if idx := strings.LastIndex(v, "\n"); idx >= 0 {
		return strings.TrimLeft(v[idx:], "\n")
	}
	return ""
}

const (
	commentTypeHash = iota
	commentTypeDoubleSlash
	commentTypeSlashAsterisk
)

func commentContentType(content string) int {
	if strings.HasPrefix(content, "#") {
		return commentTypeHash
	}
	if len(content) > 1 && content[1] == '*' {
		return commentTypeSlashAsterisk
	}
	return commentTypeDoubleSlash
}

// commentBlockIndices ports CommentsAnalyzer::getCommentBlockIndices: the indices
// of same-type "//" or "#" comments that form one consecutive block (a "/* */"
// comment is always a block of one).
func commentBlockIndices(s *tokens.Stream, index int) []int {
	ct := commentContentType(s.At(index).Value)
	indices := []int{index}
	if ct == commentTypeSlashAsterisk {
		return indices
	}
	for i := index + 1; i < s.Len(); i++ {
		t := s.At(i)
		if isComment(t) {
			if commentContentType(t.Value) == ct {
				indices = append(indices, i)
				continue
			}
			break
		}
		if t.Kind != token.Whitespace || whitespaceLineBreaks(t.Value) > 1 {
			break
		}
	}
	return indices
}

func whitespaceLineBreaks(v string) int {
	n := 0
	for i := 0; i < len(v); i++ {
		switch v[i] {
		case '\n':
			n++
		case '\r':
			n++
			if i+1 < len(v) && v[i+1] == '\n' {
				i++
			}
		}
	}
	return n
}

// commentTagRe matches a comment-marker prefix, optional whitespace and an "@tag"
// (the tag may hold a backslash or dash). The upstream trailing lookahead - the
// tag must be followed by whitespace, "(" or end - is verified separately.
var commentTagRe = regexp.MustCompile(`(?:#|//|/\*+|(?:\r\n|\n|\r)(?:\s*\*)?)\s*@([a-zA-Z0-9_\\-]+)`)

// ignoreSuppressTagRe mirrors the "@...-ignore"/"@...-suppress" lookahead.
var ignoreSuppressTagRe = regexp.MustCompile(`^[A-Za-z0-9_]+-(?:ignore|suppress)`)

// firstCommentTag returns the first boundary-valid "@tag" in content, lowercased.
func firstCommentTag(content string) (string, bool) {
	tags := commentTags(content)
	if len(tags) == 0 {
		return "", false
	}
	return strings.ToLower(tags[0]), true
}

// hasIgnoreSuppressTag reports whether content carries an "@...-ignore" or
// "@...-suppress" annotation.
func hasIgnoreSuppressTag(content string) bool {
	return slices.ContainsFunc(commentTags(content), ignoreSuppressTagRe.MatchString)
}

// commentTags collects the boundary-valid "@tag" names in content, left to right.
// A match whose tag is not followed by whitespace, "(" or end is rejected and the
// search resumes one byte past its start, as a backtracking engine would.
func commentTags(content string) []string {
	var tags []string
	for offset := 0; offset <= len(content); {
		loc := commentTagRe.FindStringSubmatchIndex(content[offset:])
		if loc == nil {
			break
		}
		tagStart, tagEnd := offset+loc[2], offset+loc[3]
		if commentTagBoundaryOK(content, tagEnd) {
			tags = append(tags, content[tagStart:tagEnd])
			offset = tagEnd
		} else {
			offset += loc[0] + 1
		}
	}
	return tags
}

func commentTagBoundaryOK(content string, end int) bool {
	if end >= len(content) {
		return true
	}
	switch content[end] {
	case '(', ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	}
	return false
}
