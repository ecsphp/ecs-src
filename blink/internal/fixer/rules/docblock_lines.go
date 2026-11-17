package rules

import (
	"regexp"
	"strings"
)

// A raw-line model of a docblock that mirrors PHP-CS-Fixer's DocBlock/Line:
// the content is split into lines, each keeping its trailing newline, and the
// opening "/**" and closing "*/" are lines too. Rebuilding concatenates them,
// so a removed line (set to "") disappears and addBlank appends a blank line.

// splitDocLines mirrors DocBlock's Preg::split('/([^\n\r]+\R*)/', ...) so each
// line carries its own trailing newline(s): every run of other characters plus
// the \r and \n after it. Newlines before the first such run are dropped.
func splitDocLines(content string) []string {
	var lines []string
	i := skipLineBreaks(content, 0)
	for i < len(content) {
		start := i
		for i < len(content) && content[i] != '\n' && content[i] != '\r' {
			i++
		}
		i = skipLineBreaks(content, i)
		lines = append(lines, content[start:i])
	}
	return lines
}

func skipLineBreaks(s string, i int) int {
	for i < len(s) && (s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i
}

var (
	// the optional trailing newline mirrors PCRE's `$`, which also matches just
	// before a single newline at the end of the subject
	docLineAddBlankRe = regexp.MustCompile(`^([\t ]*\*)[^\r\n]*(\r?\n)(?:\r?\n)?$`)
)

// isRESpace is Go regexp's ASCII \s.
func isRESpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
}

// starThenSpaces reports whether match holds for the offset after some "*" and
// the \s run that follows it.
func starThenSpaces(line string, match func(j int) bool) bool {
	for i := 0; i < len(line); i++ {
		if line[i] != '*' {
			continue
		}
		j := i + 1
		for j < len(line) && isRESpace(line[j]) {
			j++
		}
		if match(j) {
			return true
		}
	}
	return false
}

// lineContainsTag is /\*\s*@/.
func lineContainsTag(line string) bool {
	return starThenSpaces(line, func(j int) bool { return j < len(line) && line[j] == '@' })
}

func lineContainsUsefulContent(line string) bool {
	// /\*\s*\S+/: a "*" followed, after spaces, by anything else
	if !starThenSpaces(line, func(j int) bool { return j < len(line) }) {
		return false
	}
	replaced := make([]rune, 0, len(line))
	for _, r := range line {
		if r == '/' || r == '*' {
			replaced = append(replaced, ' ')
		} else {
			replaced = append(replaced, r)
		}
	}
	for _, r := range replaced {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' && r != '\v' && r != '\f' {
			return true
		}
	}
	return false
}

// lineAddBlank appends a blank doc line ("<indent>*<newline>") after line,
// mirroring Line::addBlank.
func lineAddBlank(line string) string {
	m := docLineAddBlankRe.FindStringSubmatch(line)
	if m == nil {
		return line
	}
	return line + m[1] + m[2]
}

// docAnnotation is a span of lines beginning at a tag line.
type docAnnotation struct {
	start int
	end   int
	name  string
}

// docAnnotations mirrors DocBlock::getAnnotations - each annotation runs from
// its "@tag" line through its continuation lines.
func docAnnotations(lines []string) []docAnnotation {
	var anns []docAnnotation
	total := len(lines)
	for index := 0; index < total; index++ {
		if !lineContainsTag(lines[index]) {
			continue
		}
		length := findAnnotationLength(lines, index)
		anns = append(anns, docAnnotation{
			start: index,
			end:   index + length - 1,
			name:  annotationTagName(lines[index : index+length]),
		})
		index += length - 1
	}
	return anns
}

// findAnnotationLength mirrors DocBlock::findAnnotationLength.
func findAnnotationLength(lines []string, start int) int {
	index := start
	for {
		index++
		if index >= len(lines) {
			break
		}
		line := lines[index]
		if lineContainsTag(line) {
			break
		}
		if !lineContainsUsefulContent(line) {
			if index+1 >= len(lines) || !lineContainsUsefulContent(lines[index+1]) || lineContainsTag(lines[index+1]) {
				break
			}
		}
	}
	return index - start
}

// tagNameRe matches the @tag name; the trailing boundary (whitespace, end or
// "(") is checked separately since Go's regexp has no lookahead.
var tagNameRe = regexp.MustCompile(`@([a-zA-Z0-9_\-]+)`)

func annotationTagName(lines []string) string {
	content := strings.Join(lines, "")
	loc := tagNameRe.FindStringSubmatchIndex(content)
	if loc == nil {
		return ""
	}
	name := content[loc[2]:loc[3]]
	after := loc[3]
	if after >= len(content) {
		return name
	}
	c := content[after]
	if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f' || c == '(' {
		return name
	}
	return ""
}
