package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocAlignFixer.php
//
// PhpdocAlign here follows the ECS laravel config: align="left", spacing
// {param:2,_default:1}, default tags. Left alignment collapses the spacing
// between tag/hint/var/desc to the configured spacing and re-indents wrapped
// description lines under the description column.
type PhpdocAlign struct{}

// default tags split by kind
var (
	phpAlignNameTags   = []string{"param", "property", "property-read", "property-write", "phpstan-param", "phpstan-property", "phpstan-property-read", "phpstan-property-write", "phpstan-assert", "phpstan-assert-if-true", "phpstan-assert-if-false", "psalm-param", "psalm-param-out", "psalm-property", "psalm-property-read", "psalm-property-write", "psalm-assert", "psalm-assert-if-true", "psalm-assert-if-false"}
	phpAlignMethodTags = []string{"method", "phpstan-method", "psalm-method"}
	// DEFAULT_TAGS = method,param,property,return,throws,type,var
	phpAlignNoNameTags = []string{"return", "throws", "type", "var"}
)

func (PhpdocAlign) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocAlignFixer`
}

func (PhpdocAlign) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocAlignFixer.php"
}

func phpAlignSpacingFor(tag string) int {
	if tag == "param" {
		return 2
	}
	return 1
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

func (PhpdocAlign) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		v := s.At(i).Value
		rebuilt := phpAlignDocBlock(v)
		if rebuilt != v {
			s.SetValue(i, rebuilt)
			changed = true
		}
	}
	return changed
}

func phpAlignDocBlock(content string) string {
	lines := splitDocLines(content)
	l := len(lines)
	for i := 0; i < l; i++ {
		m := phpAlignGetMatches(lines[i], false)
		if m == nil {
			continue
		}
		current := i
		items := []alignMatch{*m}
		for {
			i++
			if i >= l {
				// break both loops
				phpAlignRewrite(lines, current, items)
				return strings.Join(lines, "")
			}
			m2 := phpAlignGetMatches(lines[i], true)
			if m2 == nil {
				break
			}
			items = append(items, *m2)
		}
		phpAlignRewrite(lines, current, items)
		i-- // reconsider the line that broke the group (it may start a new group)
	}
	return strings.Join(lines, "")
}

// phpAlignRewrite reformats the grouped item lines (left align).
func phpAlignRewrite(lines []string, current int, items []alignMatch) {
	// group-wide flag: any tag line carries a `static` keyword
	hasStatic := false
	for j := range items {
		if items[j].isTag && items[j].static != "" {
			hasStatic = true
			break
		}
	}
	var itemOpening *alignMatch
	for j := range items {
		item := items[j]
		var line string
		if !item.isTag {
			if len(item.desc) > 0 && item.desc[0] == '@' {
				line = item.indent + " * " + item.desc
				lines[current+j] = line + "\n"
				continue
			}
			leftIndent := phpAlignLeftDescIndent(items, j)
			hintPad := ""
			if itemOpening != nil && itemOpening.hint != "" {
				hintPad = " "
			}
			line = item.indent + " * " + hintPad + strings.Repeat(" ", leftIndent) + item.desc
			lines[current+j] = line + "\n"
			continue
		}
		itemOpening = &items[j]
		spacing := phpAlignSpacingFor(item.tag)
		line = item.indent + " * @" + item.tag
		// static: only emitted when the group has any static keyword
		if hasStatic {
			if item.static != "" {
				line += strings.Repeat(" ", spacing) + item.static
			}
			// non-static line adds no padding under left align
		}
		// hint
		if item.hint != "" {
			line += strings.Repeat(" ", spacing) + item.hint
		}
		// var
		if item.varName != "" {
			line += strings.Repeat(" ", spacing) + item.varName
			if item.desc != "" {
				line += strings.Repeat(" ", spacing) + item.desc
			}
		} else if item.desc != "" {
			line += strings.Repeat(" ", spacing) + item.desc
		}
		lines[current+j] = line + "\n"
	}
}

// phpAlignLeftDescIndent mirrors getLeftAlignedDescriptionIndent for align=left.
func phpAlignLeftDescIndent(items []alignMatch, index int) int {
	var item *alignMatch
	for ; index >= 0; index-- {
		item = &items[index]
		if item.isTag {
			break
		}
	}
	if item == nil || !item.isTag {
		return 0
	}
	spacing := phpAlignSpacingFor(item.tag)
	sent := func(sfrag string, hasVal bool) int {
		if !hasVal {
			return 0
		}
		n := len(sfrag)
		if n == 0 {
			return 0
		}
		return n + spacing
	}
	// static, tag, hint, var
	return sent(item.static, item.static != "") +
		sent(item.tag, true) +
		sent(item.hint, item.hint != "") +
		sent(item.varName, item.varName != "")
}
