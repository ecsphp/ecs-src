package rules

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocAlignFixer.php
//
// PhpdocAlign aligns tag, hint, variable and description. By default it aligns them
// vertically with spacing 1, like PHP-CS-Fixer. With Left it collapses the spacing
// between them to the configured spacing and re-indents wrapped description lines
// under the description column.
type PhpdocAlign struct {
	Left bool
	// Spacing holds the spacing per tag, "_default" for the other tags; nil means 1.
	Spacing map[string]int
	// Tags holds the aligned tags; nil means phpAlignDefaultTags.
	Tags []string
}

// WithConfig reads the options of PHP-CS-Fixer's fixer: align, spacing (an int, or a
// map of tag to int) and tags. An option that is not set keeps its value.
func (f PhpdocAlign) WithConfig(config map[string]any) fixer.Fixer {
	if align, ok := config["align"].(string); ok {
		f.Left = align == "left"
	}
	switch spacing := config["spacing"].(type) {
	case map[string]any:
		f.Spacing = make(map[string]int, len(spacing))
		for tag := range spacing {
			f.Spacing[tag] = alignSpacing(spacing, tag)
		}
	case nil:
	default:
		f.Spacing = map[string]int{"_default": alignSpacing(config, "spacing")}
	}
	if tags, ok := phpdocTagsStringList(config["tags"]); ok {
		f.Tags = tags
	}
	// a PHP list with gaps, as array_diff gives, is dumped as an object
	if tags, ok := config["tags"].(map[string]any); ok {
		f.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			if name, ok := tag.(string); ok {
				f.Tags = append(f.Tags, name)
			}
		}
	}
	return f
}

// alignSpacing keeps the default for a value PHP-CS-Fixer rejects: zero or less.
func alignSpacing(config map[string]any, key string) int {
	return max(configInt(config, key, 1), 1)
}

var (
	phpAlignDefaultTags = []string{"method", "param", "property", "return", "throws", "type", "var"}

	phpAlignNameTags   = []string{"param", "property", "property-read", "property-write", "phpstan-param", "phpstan-property", "phpstan-property-read", "phpstan-property-write", "phpstan-assert", "phpstan-assert-if-true", "phpstan-assert-if-false", "psalm-param", "psalm-param-out", "psalm-property", "psalm-property-read", "psalm-property-write", "psalm-assert", "psalm-assert-if-true", "psalm-assert-if-false"}
	phpAlignMethodTags = []string{"method", "phpstan-method", "psalm-method"}
)

func (PhpdocAlign) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocAlignFixer`
}

func (PhpdocAlign) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocAlignFixer.php"
}

func (f PhpdocAlign) spacingFor(tag string) int {
	if spacing, ok := f.Spacing[tag]; ok {
		return spacing
	}
	if spacing, ok := f.Spacing["_default"]; ok {
		return spacing
	}
	return 1
}

func (f PhpdocAlign) indent(vertical, left int) string {
	if f.Left {
		return strings.Repeat(" ", left)
	}
	return strings.Repeat(" ", vertical)
}

type alignMatch struct {
	indent  string
	tag     string // "" for a comment-only continuation line
	hint    string
	varName string
	static  string
	desc    string
	isTag   bool // matched the tag regex
}

func (f PhpdocAlign) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		v := s.At(i).Value
		rebuilt := f.alignDocBlock(v)
		if rebuilt != v {
			s.SetValue(i, rebuilt)
			changed = true
		}
	}
	return changed
}

func (f PhpdocAlign) alignDocBlock(content string) string {
	lines := splitDocLines(content)
	l := len(lines)
	for i := 0; i < l; i++ {
		m := f.getMatches(lines[i], false)
		if m == nil {
			continue
		}
		current := i
		items := []alignMatch{*m}
		for {
			i++
			if i >= l {
				// the group runs to the end of the block: like the PHP fixer, leave it as it is
				return strings.Join(lines, "")
			}
			m2 := f.getMatches(lines[i], true)
			if m2 == nil {
				break
			}
			items = append(items, *m2)
		}
		f.rewrite(lines, current, items)
	}
	return strings.Join(lines, "")
}

// rewrite reformats the grouped item lines, like PhpdocAlignFixer::fixDocBlock.
func (f PhpdocAlign) rewrite(lines []string, current int, items []alignMatch) {
	hasStatic := false
	tagMax, hintMax, varMax := 0, 0, 0
	for j := range items {
		if !items[j].isTag {
			continue
		}
		hasStatic = hasStatic || items[j].static != ""
		tagMax = max(tagMax, len(items[j].tag))
		hintMax = max(hintMax, len(items[j].hint))
		varMax = max(varMax, len(items[j].varName))
	}

	var opening alignMatch
	var spacing int
	for j := range items {
		item := items[j]
		if !item.isTag {
			lines[current+j] = f.continuationLine(item, opening, spacing, hasStatic, tagMax+hintMax, varMax) + "\n"
			continue
		}
		opening = item
		spacing = f.spacingFor(item.tag)
		line := item.indent + " * @" + item.tag

		hintIndent := tagMax - len(item.tag) + spacing
		if hasStatic {
			if item.static != "" {
				line += f.indent(hintIndent, spacing) + item.static
			} else {
				line += f.indent(hintIndent, 0) + f.indent(6, 0)
			}
			hintIndent = spacing
		}
		hintLeft := 0
		if item.hint != "" {
			hintLeft = spacing
		}
		line += f.indent(hintIndent, hintLeft) + item.hint

		switch {
		case item.varName != "":
			column := hintMax
			if column == 0 {
				column = -1
			}
			line += f.indent(column-utf8.RuneCountInString(item.hint)+spacing, spacing) + item.varName
			if item.desc != "" {
				line += f.indent(varMax-len(item.varName)+spacing, spacing) + item.desc
			}
		case item.desc != "":
			line += f.indent(hintMax-len(item.hint)+spacing, spacing) + item.desc
		}
		lines[current+j] = line + "\n"
	}
}

func (f PhpdocAlign) continuationLine(item, opening alignMatch, spacing int, hasStatic bool, tagAndHint, varMax int) string {
	if item.desc[0] == '@' {
		return item.indent + " * " + item.desc
	}
	extra := 2 * spacing
	if slices.Contains(phpAlignNameTags, opening.tag) || slices.Contains(phpAlignMethodTags, opening.tag) {
		extra += varMax + spacing
	}
	if hasStatic {
		extra += 7 // len("static ")
	}
	hintPad := ""
	if opening.hint != "" {
		hintPad = " "
	}
	return item.indent + " * " + hintPad + f.indent(tagAndHint+extra, leftDescIndent(opening, spacing)) + item.desc
}

// leftDescIndent is the width of the opening tag line up to its description.
func leftDescIndent(opening alignMatch, spacing int) int {
	indent := 0
	for _, part := range []string{opening.static, opening.tag, opening.hint, opening.varName} {
		if part != "" {
			indent += len(part) + spacing
		}
	}
	return indent
}
