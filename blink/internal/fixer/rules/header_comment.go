package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/HeaderCommentFixer.php
//
// HeaderComment adds, replaces or normalizes a file header comment. It is a
// no-op unless a non-empty header is configured, so it never changes a file on
// its default configuration. With a header it inserts the comment after the
// opening tag or after declare(strict_types=1), honouring the comment_type,
// location and separate options.
type HeaderComment struct {
	Header      string
	CommentType string
	Location    string
	Separate    string
}

func (HeaderComment) Name() string {
	return `PhpCsFixer\Fixer\Comment\HeaderCommentFixer`
}

func (HeaderComment) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Comment/HeaderCommentFixer.php"
}

func (f HeaderComment) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["header"].(string); ok {
		f.Header = v
	}
	if v, ok := config["comment_type"].(string); ok {
		f.CommentType = v
	}
	if v, ok := config["location"].(string); ok {
		f.Location = v
	}
	if v, ok := config["separate"].(string); ok {
		f.Separate = v
	}
	return f
}

const (
	hcTypeComment = "comment"
	hcTypePHPDoc  = "PHPDoc"
)

var (
	hcBlankLineRe = regexp.MustCompile(`^[ \t]*(?:\r\n|\n|\r)[ \t]*$`)
	hcTrailRe     = regexp.MustCompile(`(?:\r\n|\n|\r)?[ \t]*$`)
)

func (f HeaderComment) Fix(s *tokens.Stream) bool {
	header := f.Header
	if strings.TrimSpace(header) == "" || strings.Contains(header, "*/") {
		return false
	}
	if !isMonolithicPHPFile(s) {
		return false
	}
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind == token.OpenTag && s.At(i).Value == "<?=" {
			return false
		}
	}

	commentType := f.CommentType
	if commentType == "" {
		commentType = hcTypeComment
	}
	location := f.Location
	if location == "" {
		location = "after_declare_strict"
	}
	separate := f.Separate
	if separate == "" {
		separate = "both"
	}

	version, before := s.Version(), s.Render()
	headerAsComment := hcHeaderAsComment(header, commentType)

	// dedup the two candidate insertion slots the way the upstream fixer does,
	// preferring the configured location when both resolve to the same slot
	slotPos := map[int]int{}
	var ordered []string
	for _, possible := range []string{"after_open", "after_declare_strict"} {
		slot := hcInsertionIndex(s, possible)
		if pos, ok := slotPos[slot]; ok {
			if possible == location {
				ordered[pos] = possible
			}
			continue
		}
		slotPos[slot] = len(ordered)
		ordered = append(ordered, possible)
	}

	for _, possible := range ordered {
		headerNewIndex := hcInsertionIndex(s, possible)
		headerCurrentIndex := hcCurrentIndex(s, headerAsComment, headerNewIndex-1, separate, commentType)

		if headerCurrentIndex < 0 {
			if possible != location {
				continue
			}
			hcInsertHeader(s, headerAsComment, headerNewIndex, commentType, separate)
			continue
		}

		currentComment := s.At(headerCurrentIndex).Value
		sameComment := headerAsComment == currentComment
		expectedLocation := possible == location

		if !sameComment || !expectedLocation {
			if expectedLocation != sameComment {
				hcRemoveHeader(s, headerCurrentIndex)
			}
			if possible == location {
				hcInsertHeader(s, headerAsComment, headerNewIndex, commentType, separate)
			}
			continue
		}

		hcFixWhitespace(s, headerCurrentIndex, separate)
	}

	s.ClearEmptyTokens()
	return s.Version() != version && s.Render() != before
}

// hcHeaderAsComment encloses the header text in a comment block.
func hcHeaderAsComment(header, commentType string) string {
	open := "/*"
	if commentType == hcTypePHPDoc {
		open = "/**"
	}
	var b strings.Builder
	b.WriteString(open)
	b.WriteByte('\n')
	for line := range strings.SplitSeq(strings.ReplaceAll(header, "\r", ""), "\n") {
		b.WriteString(strings.TrimRight(" * "+line, " \t\n\r\x00\x0b"))
		b.WriteByte('\n')
	}
	b.WriteString(" */")
	return b.String()
}

// hcInsertionIndex finds the index where the header comment must be inserted.
func hcInsertionIndex(s *tokens.Stream, location string) int {
	openTagIndex := 0
	if s.Len() > 0 && s.At(0).Kind == token.InlineHTML {
		openTagIndex = 1
	}
	if location == "after_open" {
		return openTagIndex + 1
	}

	index := hcMeaningfulSibling(s, openTagIndex, 1)
	if !hcWordAt(s, index, "declare") {
		return openTagIndex + 1
	}
	next := hcMeaningfulSibling(s, index, 1)
	if !hcPunctAt(s, next, "(") {
		return openTagIndex + 1
	}
	next = hcMeaningfulSibling(s, next, 1)
	if !hcWordAt(s, next, "strict_types") {
		return openTagIndex + 1
	}
	next = hcMeaningfulSibling(s, next, 1)
	if !hcPunctAt(s, next, "=") {
		return openTagIndex + 1
	}
	next = hcMeaningfulSibling(s, next, 1)
	if next < 0 || s.At(next).Kind != token.Number {
		return openTagIndex + 1
	}
	next = hcMeaningfulSibling(s, next, 1)
	if !hcPunctAt(s, next, ")") {
		return openTagIndex + 1
	}
	next = hcMeaningfulSibling(s, next, 1)
	if !hcPunctAt(s, next, ";") {
		return openTagIndex + 1
	}
	return next + 1
}

// hcCurrentIndex finds an existing header comment at start's location, or -1.
func hcCurrentIndex(s *tokens.Stream, headerAsComment string, start int, separate, commentType string) int {
	index := hcNextNonWhitespace(s, start)
	if index < 0 || !hcIsComment(s.At(index)) {
		return -1
	}
	next := index + 1
	if next >= s.Len() || separate == "top" || separate == "none" || s.At(index).Kind != token.DocComment {
		return index
	}
	if hcIsWhitespace(s.At(next)) {
		if !hcBlankLineRe.MatchString(s.At(next).Value) {
			return index
		}
		next++
	}
	if next >= s.Len() || (!hcIsClassy(s, next) && !hcWordAt(s, next, "function")) {
		return index
	}
	if headerAsComment == s.At(index).Value {
		return index
	}
	return -1
}

func hcInsertHeader(s *tokens.Stream, headerAsComment string, index int, commentType, separate string) {
	kind := token.Comment
	if commentType == hcTypePHPDoc {
		kind = token.DocComment
	}
	s.InsertAt(index, token.Token{Kind: kind, Value: headerAsComment})
	hcFixWhitespace(s, index, separate)
}

func hcRemoveHeader(s *tokens.Stream, index int) {
	newlineRemoved := false
	if prev := index - 1; prev >= 0 && hcIsWhitespace(s.At(prev)) {
		content := s.At(prev).Value
		if strings.ContainsAny(content, "\r\n") {
			newlineRemoved = true
		}
		s.SetValue(prev, hcTrailRe.ReplaceAllString(content, ""))
	}
	if next := index + 1; !newlineRemoved && next < s.Len() && hcIsWhitespace(s.At(next)) {
		s.SetValue(next, hcStripLeadingNewline(s.At(next).Value))
	}
	hcClearAndMerge(s, index)
}

// hcClearAndMerge mirrors Tokens::clearTokenAndMergeSurroundingWhitespace: it
// empties the token at index and folds the whitespace on either side into one.
func hcClearAndMerge(s *tokens.Stream, index int) {
	count := s.Len()
	s.SetValue(index, "")
	if index == count-1 {
		return
	}
	nextIndex := hcNonEmptySibling(s, index, 1)
	if nextIndex < 0 || !hcIsWhitespace(s.At(nextIndex)) {
		return
	}
	prevIndex := hcNonEmptySibling(s, index, -1)
	if prevIndex >= 0 && hcIsWhitespace(s.At(prevIndex)) {
		s.SetValue(prevIndex, s.At(prevIndex).Value+s.At(nextIndex).Value)
	} else if prevIndex+1 < s.Len() && s.At(prevIndex+1).Value == "" {
		s.SetValue(prevIndex+1, s.At(nextIndex).Value)
	}
	s.SetValue(nextIndex, "")
}

func hcFixWhitespace(s *tokens.Stream, headerIndex int, separate string) {
	const lineEnding = "\n"

	expectedAfter := 1
	if (separate == "both" || separate == "bottom") && hcMeaningfulSibling(s, headerIndex, 1) >= 0 {
		expectedAfter = 2
	}
	if headerIndex == s.Len()-1 {
		s.InsertAt(headerIndex+1, token.Token{Kind: token.Whitespace, Value: strings.Repeat(lineEnding, expectedAfter)})
	} else {
		count := hcLineBreakCount(s, headerIndex, 1)
		switch {
		case count < expectedAfter:
			missing := strings.Repeat(lineEnding, expectedAfter-count)
			if headerIndex+1 < s.Len() && hcIsWhitespace(s.At(headerIndex+1)) {
				s.SetValue(headerIndex+1, missing+s.At(headerIndex+1).Value)
			} else {
				s.InsertAt(headerIndex+1, token.Token{Kind: token.Whitespace, Value: missing})
			}
		case count > expectedAfter && headerIndex+1 < s.Len() && hcIsWhitespace(s.At(headerIndex+1)):
			s.SetValue(headerIndex+1, hcStripLeadingNewlines(s.At(headerIndex+1).Value, count-expectedAfter))
		}
	}

	expectedBefore := 1
	if separate == "both" || separate == "top" {
		expectedBefore = 2
	}
	// the upstream T_OPEN_TAG trailing-space fixup does not apply: blink lexes the
	// open tag as "<?php" alone, with its following whitespace as its own token.
	if count := hcLineBreakCount(s, headerIndex, -1); count < expectedBefore {
		s.InsertAt(headerIndex, token.Token{Kind: token.Whitespace, Value: strings.Repeat(lineEnding, expectedBefore-count)})
	}
}

func hcLineBreakCount(s *tokens.Stream, index, direction int) int {
	var ws strings.Builder
	for i := index + direction; i >= 0 && i < s.Len(); i += direction {
		t := s.At(i)
		if hcIsWhitespace(t) {
			ws.WriteString(t.Value)
			continue
		}
		if direction == -1 && t.Kind == token.OpenTag {
			ws.WriteString(t.Value)
		}
		if t.Value != "" {
			break
		}
	}
	return strings.Count(ws.String(), "\n")
}

// hcStripLeadingNewline drops a single leading line break, like /^\R/.
func hcStripLeadingNewline(v string) string {
	switch {
	case strings.HasPrefix(v, "\r\n"):
		return v[2:]
	case strings.HasPrefix(v, "\n"), strings.HasPrefix(v, "\r"):
		return v[1:]
	}
	return v
}

// hcStripLeadingNewlines drops up to n leading line breaks, like /^\R{n}/.
func hcStripLeadingNewlines(v string, n int) string {
	for ; n > 0; n-- {
		stripped := hcStripLeadingNewline(v)
		if stripped == v {
			break
		}
		v = stripped
	}
	return v
}

func hcIsWhitespace(t token.Token) bool {
	return t.Value != "" && t.Kind == token.Whitespace
}

func hcIsComment(t token.Token) bool {
	return t.Value != "" && (t.Kind == token.Comment || t.Kind == token.DocComment)
}

func hcIsClassy(s *tokens.Stream, i int) bool {
	return hcWordAt(s, i, "class") || hcWordAt(s, i, "interface") || hcWordAt(s, i, "trait") || hcWordAt(s, i, "enum")
}

func hcWordAt(s *tokens.Stream, i int, word string) bool {
	if i < 0 || i >= s.Len() {
		return false
	}
	t := s.At(i)
	return t.Value != "" && (t.Kind == token.Keyword || t.Kind == token.Ident) && strings.EqualFold(t.Value, word)
}

func hcPunctAt(s *tokens.Stream, i int, v string) bool {
	return i >= 0 && i < s.Len() && s.At(i).Kind == token.Punct && s.At(i).Value == v
}

// hcNextNonWhitespace returns the first token after i that is not whitespace,
// stopping at an emptied token as Tokens::getNextNonWhitespace does.
func hcNextNonWhitespace(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		if !hcIsWhitespace(s.At(j)) {
			return j
		}
	}
	return -1
}

// hcMeaningfulSibling returns the closest non-empty token in direction that is
// not whitespace or a comment, like Tokens::getMeaningfulTokenSibling.
func hcMeaningfulSibling(s *tokens.Stream, index, direction int) int {
	for i := index + direction; i >= 0 && i < s.Len(); i += direction {
		t := s.At(i)
		if t.Value == "" || t.Kind == token.Whitespace || t.Kind == token.Comment || t.Kind == token.DocComment {
			continue
		}
		return i
	}
	return -1
}

func hcNonEmptySibling(s *tokens.Stream, index, direction int) int {
	for i := index + direction; i >= 0 && i < s.Len(); i += direction {
		if s.At(i).Value != "" {
			return i
		}
	}
	return -1
}
