package rules

import (
	"regexp"
	"slices"
	"strings"
)

// A port of PHP-CS-Fixer's DocBlock, Line, Annotation and Tag classes
// (src/DocBlock/). Lines keep their own line ending, as in PHP.

// phpDocLine is DocBlock\Line: one line of a docblock, with its line ending.
type phpDocLine struct {
	content string
}

// splitPHPDocLines ports DocBlock's Preg::split('/([^\n\r]+\R*)/', ...): each
// line is a run of characters other than \n and \r plus the \R characters
// after it.
func splitPHPDocLines(content string) []*phpDocLine {
	var lines []*phpDocLine
	for i := 0; i < len(content); {
		start := i
		for i < len(content) && content[i] != '\n' && content[i] != '\r' {
			i++
		}
		i = pcreSkip(content, i, pcreIsVSpace)
		lines = append(lines, &phpDocLine{content: content[start:i]})
	}
	return lines
}

// pcreStarThenSpaces reports whether match holds for the offset after some
// "*" and the UCP \s run that follows it.
func pcreStarThenSpaces(line string, match func(j int) bool) bool {
	for i := 0; i < len(line); i++ {
		if line[i] == '*' && match(pcreSkip(line, i+1, pcreIsSpace)) {
			return true
		}
	}
	return false
}

// containsUsefulContent ports Line::containsUsefulContent.
func (l *phpDocLine) containsUsefulContent() bool {
	// /\*\s*\S+/
	if !pcreStarThenSpaces(l.content, func(j int) bool { return j < len(l.content) }) {
		return false
	}
	return strings.Trim(strings.NewReplacer("/", " ", "*", " ").Replace(l.content), " \t\n\r\x00\x0B") != ""
}

// containsATag ports Line::containsATag, /\*\s*@/.
func (l *phpDocLine) containsATag() bool {
	return pcreStarThenSpaces(l.content, func(j int) bool { return j < len(l.content) && l.content[j] == '@' })
}

func (l *phpDocLine) isTheStart() bool { return strings.Contains(l.content, "/**") }
func (l *phpDocLine) isTheEnd() bool   { return strings.Contains(l.content, "*/") }

// phpDocTagName ports Tag::getName, /@[a-zA-Z0-9_-]+(?=\s|$)/: the first
// "@name" on the line that is followed by whitespace or the end, or "other".
func phpDocTagName(line string) string {
	for i := 0; i < len(line); i++ {
		if line[i] != '@' {
			continue
		}
		end := i + 1
		for end < len(line) && isTagNameByte(line[end]) {
			end++
		}
		if end == i+1 {
			continue
		}
		if r, _ := pcreRuneAt(line, end); end == len(line) || pcreIsSpace(r) {
			return line[i+1 : end]
		}
		i = end - 1
	}
	return "other"
}

func isTagNameByte(c byte) bool {
	return c == '_' || c == '-' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')
}

// phpDocAnnotation is DocBlock\Annotation: the lines [start, end] of one tag.
type phpDocAnnotation struct {
	start, end int
	lines      []*phpDocLine
	tag        string
}

func (a *phpDocAnnotation) content() string {
	var b strings.Builder
	for _, l := range a.lines {
		b.WriteString(l.content)
	}
	return b.String()
}

var (
	// #(\s*/\*\*).*# and #(\s*)\S.*(\*/.*)#
	phpDocRemoveStartRe = regexp.MustCompile(`([` + pcreSpaceClass + `]*/\*\*).*`)
	phpDocRemoveEndRe   = regexp.MustCompile(`([` + pcreSpaceClass + `]*)[^` + pcreSpaceClass + `].*(\*/.*)`)
)

// remove ports Annotation::remove: blank the annotation's lines, keeping the
// docblock opener/closer when the annotation shares a line with it.
func (a *phpDocAnnotation) remove() {
	for _, l := range a.lines {
		switch start, end := l.isTheStart(), l.isTheEnd(); {
		case start && end:
			l.content = ""
		case start:
			l.content = phpDocRemoveStartRe.ReplaceAllString(l.content, "$1")
		case end:
			l.content = phpDocRemoveEndRe.ReplaceAllString(l.content, "$1$2")
		default:
			l.content = ""
		}
	}
}

// phpDocBlock is DocBlock\DocBlock, parsed once (PHP caches the annotations
// too, so mutations during a pass do not re-split the block).
type phpDocBlock struct {
	lines       []*phpDocLine
	annotations []*phpDocAnnotation
}

func newPHPDocBlock(content string) *phpDocBlock {
	d := &phpDocBlock{lines: splitPHPDocLines(content)}
	for i := 0; i < len(d.lines); i++ {
		if !d.lines[i].containsATag() {
			continue
		}
		end := i + d.annotationLength(i) - 1
		d.annotations = append(d.annotations, &phpDocAnnotation{
			start: i,
			end:   end,
			lines: d.lines[i : end+1],
			tag:   phpDocTagName(d.lines[i].content),
		})
		i = end
	}
	return d
}

// annotationLength ports DocBlock::findAnnotationLength.
func (d *phpDocBlock) annotationLength(start int) int {
	index := start
	for {
		index++
		if index >= len(d.lines) {
			break
		}
		line := d.lines[index]
		if line.containsATag() {
			break
		}
		if !line.containsUsefulContent() {
			if index+1 >= len(d.lines) {
				break
			}
			next := d.lines[index+1]
			if !next.containsUsefulContent() || next.containsATag() {
				break
			}
		}
	}
	return index - start
}

// annotationsOfType ports DocBlock::getAnnotationsOfType (exact tag names).
func (d *phpDocBlock) annotationsOfType(types []string) []*phpDocAnnotation {
	var out []*phpDocAnnotation
	for _, a := range d.annotations {
		if slices.Contains(types, a.tag) {
			out = append(out, a)
		}
	}
	return out
}

func (d *phpDocBlock) content() string {
	var b strings.Builder
	for _, l := range d.lines {
		b.WriteString(l.content)
	}
	return b.String()
}
