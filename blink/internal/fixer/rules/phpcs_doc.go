package rules

import (
	"sort"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// phpcsDocTag is a docblock tag as PHP_CodeSniffer's comment tokenizer splits
// it, with the T_DOC_COMMENT_STRING that follows it on the same line.
type phpcsDocTag struct {
	Name   string
	Str    string
	StrPos int // byte offset of Str in the docblock, -1 when the line has none
}

// phpcsDocTags tokenizes doc like PHP_CodeSniffer's Comment tokenizer: the open
// tag is the leading "/*" run, the close tag the trailing one, and each line is
// padding, an optional "*" (not on the first line), padding, an optional tag
// with its padding, then the rest of the line as the string.
func phpcsDocTags(doc, eol string) []phpcsDocTag {
	open := len(doc) - len(strings.TrimLeft(doc, "/*"))
	if open == len(doc) {
		return nil
	}
	body := strings.TrimRight(doc[open:], "/*")
	lines := strings.Split(body, eol)
	var tags []phpcsDocTag
	pos := open
	for n, line := range lines {
		str := line
		if n < len(lines)-1 {
			str += eol
		}
		base := pos
		pos += len(str)
		c := skipDocPadding(str, 0)
		if c > 0 && c == len(str) {
			break
		}
		if str == "" {
			continue
		}
		if n > 0 && str[c] == '*' {
			c++
		}
		c = skipDocPadding(str, c)
		if c >= len(str) {
			continue
		}
		tag := phpcsDocTag{StrPos: -1}
		if str[c] == '@' {
			if name := phpcsDocTagName(str[c:]); name != "" && !strings.HasPrefix(strings.ToLower(name), "@phpcs:") {
				tag.Name = name
				c = skipDocPadding(str, c+len(name))
			}
		}
		end := len(str)
		if k := strings.Index(str[c:], eol); k >= 0 {
			end = c + k
		}
		if end > c {
			tag.Str = str[c:end]
			tag.StrPos = base + c
		}
		if tag.Name != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func skipDocPadding(str string, c int) int {
	for c < len(str) && (str[c] == ' ' || str[c] == '\t') {
		c++
	}
	return c
}

// phpcsDocTagName is the first "@" plus non-space run in v, as PHP_CodeSniffer's
// unanchored "/@[^\s]+/" finds it.
func phpcsDocTagName(v string) string {
	for k := 0; k < len(v); k++ {
		if v[k] != '@' {
			continue
		}
		end := k + 1
		for end < len(v) && !strings.ContainsRune(" \t\n\r\v\f", rune(v[end])) {
			end++
		}
		if end > k+1 {
			return v[k:end]
		}
	}
	return ""
}

// replaceDocStrings swaps the strings of tags for their replacements (keyed by
// StrPos) and returns the new docblock.
func replaceDocStrings(doc string, repl map[int][2]string) string {
	positions := make([]int, 0, len(repl))
	for p := range repl {
		positions = append(positions, p)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(positions)))
	for _, p := range positions {
		r := repl[p]
		doc = doc[:p] + r[1] + doc[p+len(r[0]):]
	}
	return doc
}

// phpcsLines is the 1-based line PHP_CodeSniffer gives each token: the count
// of eol in everything before it.
func phpcsLines(s *tokens.Stream, eol string) []int {
	lines := make([]int, s.Len())
	line := 1
	for i := range s.Len() {
		lines[i] = line
		line += strings.Count(s.At(i).Value, eol)
	}
	return lines
}

// phpcsDocEdit is one fix decided on an unchanged stream; edits run from the
// highest index down so earlier indices stay valid, as PHP_CodeSniffer applies
// every fix of a pass against the tokens it was decided on.
type phpcsDocEdit struct {
	at    int
	apply func(s *tokens.Stream)
}

func applyPhpcsEdits(s *tokens.Stream, edits []phpcsDocEdit) bool {
	sort.SliceStable(edits, func(a, b int) bool { return edits[a].at > edits[b].at })
	for _, e := range edits {
		e.apply(s)
	}
	return len(edits) > 0
}

// phpcsFixLoop re-runs pass until it changes nothing, at most 50 times like
// PHP_CodeSniffer's fixer.
func phpcsFixLoop(s *tokens.Stream, pass func(s *tokens.Stream) bool) bool {
	changed := false
	for range 50 {
		if !pass(s) {
			break
		}
		changed = true
	}
	return changed
}

// phpTrim is PHP's trim() with its default character list.
func phpTrim(v string) string {
	return strings.Trim(v, " \t\n\r\x00\x0B")
}
