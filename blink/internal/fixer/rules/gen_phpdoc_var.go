package rules

import (
	"regexp"
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// nextCodeIndex returns the first non-whitespace, non-comment token after i, or
// -1. It mirrors PHP-CS-Fixer's getNextMeaningfulToken.
func nextCodeIndex(s *tokens.Stream, i int) int {
	for j := i + 1; j < s.Len(); j++ {
		switch s.At(j).Kind {
		case token.Whitespace, token.Comment, token.DocComment:
			continue
		}
		return j
	}
	return -1
}

func isKeywordValue(t token.Token, vals ...string) bool {
	if t.Kind != token.Keyword {
		return false
	}
	return slices.Contains(vals, strings.ToLower(t.Value))
}

// isPropertyModifier reports whether t is an access/property modifier keyword,
// i.e. the docblock in front of it documents a class property.
func isPropertyModifier(t token.Token) bool {
	return isKeywordValue(t, "private", "protected", "public", "var", "readonly")
}

var (
	varWithoutNameTagRe = regexp.MustCompile(`(?i)^@(?:var|type)(?:\s|$)`)
	varNameRe           = regexp.MustCompile(` \$[A-Za-z_\x{0080}-\x{00FF}][A-Za-z0-9_\x{0080}-\x{00FF}]*`)
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocVarWithoutNameFixer.php
//
// PhpdocVarWithoutName removes the variable name from @var/@type tags in a docblock
// documenting a class property, e.g. "@var int $bar" -> "@var int". Conservative:
// docblocks containing array-shape braces are left untouched.
type PhpdocVarWithoutName struct{}

func (PhpdocVarWithoutName) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocVarWithoutNameFixer`
}

func (PhpdocVarWithoutName) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocVarWithoutNameFixer.php"
}

func (PhpdocVarWithoutName) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		next := nextCodeIndex(s, i)
		if next < 0 {
			continue
		}
		// "static public $foo" - skip the leading static.
		if isKeywordValue(s.At(next), "static") {
			next = nextCodeIndex(s, next)
			if next < 0 {
				continue
			}
		}
		if !isPropertyModifier(s.At(next)) {
			continue
		}
		d, ok := parseDoc(s.At(i).Value)
		if !ok {
			continue
		}
		if docblockHasBraces(d) {
			continue
		}
		if fixVarNames(&d) {
			s.SetValue(i, d.render())
			changed = true
		}
	}
	return changed
}

func docblockHasBraces(d docblock) bool {
	for _, l := range d.inner {
		if strings.ContainsAny(l.content, "{}") {
			return true
		}
	}
	return false
}

// fixVarNames strips " $name" (but never " $this") from @var/@type lines.
func fixVarNames(d *docblock) bool {
	changed := false
	for i, l := range d.inner {
		if !varWithoutNameTagRe.MatchString(strings.TrimSpace(l.content)) {
			continue
		}
		newContent := varNameRe.ReplaceAllStringFunc(l.content, func(m string) string {
			if m == " $this" {
				return m
			}
			return ""
		})
		if newContent != l.content {
			d.inner[i].content = newContent
			changed = true
		}
	}
	return changed
}
