package rules

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocSeparationFixer.php
//
// PhpdocSeparation groups annotations of the same type together and separates
// annotations of a different type by a single blank line, and separates the
// description from the annotations.
//
// It reads groups (the sets of tags kept together, `*` matching any characters;
// the tags are taken as written, so "@param" never matches) and
// skip_unlisted_annotations (false by default: true leaves alone the spacing
// between two annotations when no group decides, as when a tag is unlisted).
// PHP rejects a tag listed in two groups; blink takes such a config as given.
type PhpdocSeparation struct {
	groups       [][]string // nil keeps phpdocSeparationGroups
	skipUnlisted bool
}

// phpdocSeparationGroups is the default `groups` option.
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

func (f PhpdocSeparation) WithConfig(config map[string]any) fixer.Fixer {
	if groups, ok := phpdocSeparationGroupsOption(config["groups"]); ok {
		f.groups = groups
	}
	if v, ok := config["skip_unlisted_annotations"].(bool); ok {
		f.skipUnlisted = v
	}
	return f
}

// phpdocSeparationGroupsOption reads a list of string lists; a PHP list with
// gaps arrives as an object, read in key order.
func phpdocSeparationGroupsOption(v any) ([][]string, bool) {
	outer, ok := phpListValues(v)
	if !ok {
		return nil, false
	}
	groups := make([][]string, 0, len(outer))
	for _, g := range outer {
		inner, ok := phpListValues(g)
		if !ok {
			return nil, false
		}
		group := make([]string, 0, len(inner))
		for _, tag := range inner {
			if name, ok := tag.(string); ok {
				group = append(group, name)
			}
		}
		groups = append(groups, group)
	}
	return groups, true
}

// phpListValues returns the values of a JSON-decoded PHP list: a []any, or an
// object with integer keys (a list with gaps), in key order.
func phpListValues(v any) ([]any, bool) {
	switch list := v.(type) {
	case []any:
		return list, true
	case []string:
		out := make([]any, len(list))
		for i, s := range list {
			out[i] = s
		}
		return out, true
	case map[string]any:
		keys := make([]string, 0, len(list))
		for k := range list {
			keys = append(keys, k)
		}
		slices.SortFunc(keys, func(a, b string) int {
			ai, _ := strconv.Atoi(a)
			bi, _ := strconv.Atoi(b)
			return ai - bi
		})
		out := make([]any, len(keys))
		for i, k := range keys {
			out[i] = list[k]
		}
		return out, true
	}
	return nil, false
}

func (f PhpdocSeparation) Fix(s *tokens.Stream) bool {
	groups := f.groups
	if groups == nil {
		groups = phpdocSeparationGroups
	}
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
		fixSeparationAnnotations(lines, groups, f.skipUnlisted)
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

func fixSeparationAnnotations(lines []string, groups [][]string, skipUnlisted bool) {
	anns := docAnnotations(lines)
	names := make([]string, len(anns))
	for idx, a := range anns {
		names[idx] = separationTagName(strings.Join(lines[a.start:a.end+1], ""))
	}
	for idx := range anns {
		if idx+1 >= len(anns) {
			break
		}
		first, next := anns[idx], anns[idx+1]
		together, decided := separationShouldBeTogether(names[idx], names[idx+1], groups)
		if together {
			ensureAnnotationsTogether(lines, first, next)
		} else if decided || !skipUnlisted {
			ensureAnnotationsSeparate(lines, first, next)
		}
	}
}

// separationTagRe is PHP's `/@([a-zA-Z0-9_\\\-]+(?=\s|$|\()/`; the lookahead is
// checked in separationTagName.
var separationTagRe = regexp.MustCompile(`@([a-zA-Z0-9_\\\-]+)`)

// separationTagName mirrors PhpdocSeparationFixer::tagName: the first "@name"
// of the annotation followed by whitespace, "(" or the end ("" if none). Unlike
// the shared annotationTagName, the name may hold a backslash (`@ORM\Id`), and
// a later "@name" is tried when the first one is not followed by a boundary.
func separationTagName(content string) string {
	for _, loc := range separationTagRe.FindAllStringSubmatchIndex(content, -1) {
		after := loc[3]
		if after >= len(content) {
			return content[loc[2]:loc[3]]
		}
		switch content[after] {
		case ' ', '\t', '\n', '\r', '\v', '\f', '(':
			return content[loc[2]:loc[3]]
		}
	}
	return ""
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

// separationShouldBeTogether mirrors PHP's shouldBeTogether, which returns
// true, false or null: decided is false for null (a tag could not be read, or
// no group lists either tag).
func separationShouldBeTogether(first, second string, groups [][]string) (together, decided bool) {
	if first == "" || second == "" {
		return false, false
	}
	if first == second {
		return true, true
	}
	for _, group := range groups {
		firstIn := tagInGroup(first, group)
		secondIn := tagInGroup(second, group)
		if firstIn {
			return secondIn, true
		}
		if secondIn {
			return false, true
		}
	}
	return false, false
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
