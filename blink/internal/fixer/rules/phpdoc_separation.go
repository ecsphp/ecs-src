package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocSeparationFixer.php
//
// PhpdocSeparation groups annotations of the same type together and separates
// annotations of a different type by a single blank line, and separates the
// description from the annotations.
type PhpdocSeparation struct{}

// phpdocSeparationGroups is the default `groups` option: tags in the same group
// stay together, everything else is separated (skip_unlisted_annotations is
// false under the v3 defaults).
var phpdocSeparationGroups = [][]string{
	{"author", "copyright", "license"},
	{"category", "package", "subpackage"},
	{"property", "property-read", "property-write"},
	{"deprecated", "link", "see", "since"},
}

func (PhpdocSeparation) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocSeparationFixer`
}

func (PhpdocSeparation) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocSeparationFixer.php"
}

func (PhpdocSeparation) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		lines := splitDocLines(s.At(i).Value)
		if len(lines) < 2 {
			continue
		}
		fixSeparationDescription(lines)
		fixSeparationAnnotations(lines)
		rebuilt := strings.Join(lines, "")
		if rebuilt != s.At(i).Value {
			s.SetValue(i, rebuilt)
			changed = true
		}
	}
	return changed
}

// fixSeparationDescription separates the description from the first annotation.
func fixSeparationDescription(lines []string) {
	for i := range lines {
		if lineContainsTag(lines[i]) {
			break
		}
		if lineContainsUsefulContent(lines[i]) {
			if i+1 < len(lines) && lineContainsTag(lines[i+1]) {
				lines[i] = lineAddBlank(lines[i])
				break
			}
		}
	}
}

func fixSeparationAnnotations(lines []string) {
	anns := docAnnotations(lines)
	for idx := range anns {
		if idx+1 >= len(anns) {
			break
		}
		first, next := anns[idx], anns[idx+1]
		if separationShouldBeTogether(first.name, next.name) {
			ensureAnnotationsTogether(lines, first, next)
		} else {
			ensureAnnotationsSeparate(lines, first, next)
		}
	}
}

// ensureAnnotationsTogether removes any lines between two annotations.
func ensureAnnotationsTogether(lines []string, first, second docAnnotation) {
	for pos := first.end + 1; pos < second.start; pos++ {
		lines[pos] = ""
	}
}

// ensureAnnotationsSeparate leaves exactly one blank line between two annotations.
func ensureAnnotationsSeparate(lines []string, first, second docAnnotation) {
	pos := first.end
	final := second.start - 1
	if pos == final {
		lines[pos] = lineAddBlank(lines[pos])
		return
	}
	for p := pos + 1; p < final; p++ {
		lines[p] = ""
	}
}

// separationShouldBeTogether reports whether two adjacent tags must stay
// together (true only for the DocBlock's "true" case; null and false both mean
// separate under the default skip_unlisted_annotations=false).
func separationShouldBeTogether(first, second string) bool {
	if first == "" || second == "" {
		return false
	}
	if first == second {
		return true
	}
	for _, group := range phpdocSeparationGroups {
		firstIn := tagInGroup(first, group)
		secondIn := tagInGroup(second, group)
		if firstIn {
			return secondIn
		}
		if secondIn {
			return false
		}
	}
	return false
}

func tagInGroup(tag string, group []string) bool {
	for _, g := range group {
		if !strings.Contains(g, "*") {
			if g == tag {
				return true
			}
			continue
		}
		if wildcardTagMatch(g, tag) {
			return true
		}
	}
	return false
}

// wildcardTagMatch matches a group entry containing "*" (any characters) against
// a tag, anchored at both ends.
func wildcardTagMatch(pattern, tag string) bool {
	parts := strings.Split(pattern, "*")
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			if !strings.HasPrefix(tag[pos:], part) {
				return false
			}
			pos += len(part)
			continue
		}
		idx := strings.Index(tag[pos:], part)
		if idx < 0 {
			return false
		}
		pos += idx + len(part)
	}
	if parts[len(parts)-1] != "" {
		return strings.HasSuffix(tag, parts[len(parts)-1])
	}
	return true
}
