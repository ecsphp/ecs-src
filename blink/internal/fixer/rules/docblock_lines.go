package rules

import (
	"regexp"
	"strings"
)

// A raw-line model of a docblock that mirrors PHP-CS-Fixer's DocBlock/Line:
// the content is split into lines, each keeping its trailing newline, and the
// opening "/**" and closing "*/" are lines too. Rebuilding concatenates them,
// so a removed line (set to "") disappears and addBlank appends a blank line.

var docLineSplitRe = regexp.MustCompile(`[^\n\r]+(?:\r\n|\r|\n)*`)

// splitDocLines mirrors DocBlock's Preg::split('/([^\n\r]+\R*)/', ...) so each
// line carries its own trailing newline(s).
func splitDocLines(content string) []string {
	return docLineSplitRe.FindAllString(content, -1)
}

var (
	docLineTagRe    = regexp.MustCompile(`\*\s*@`)
	docLineUsefulRe = regexp.MustCompile(`\*\s*\S+`)
	// the optional trailing newline mirrors PCRE's `$`, which also matches just
	// before a single newline at the end of the subject
	docLineAddBlankRe = regexp.MustCompile(`^([\t ]*\*)[^\r\n]*(\r?\n)(?:\r?\n)?$`)
)

func lineContainsTag(line string) bool {
	return docLineTagRe.MatchString(line)
}

func lineContainsUsefulContent(line string) bool {
	if !docLineUsefulRe.MatchString(line) {
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
