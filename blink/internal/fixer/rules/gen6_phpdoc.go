package rules

import (
	"regexp"
	"strings"

	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocSummaryFixer.php
//
// PhpdocSummary ends a docblock's summary (short description) with a full stop.
// A summary already closed by sentence-final punctuation, an {@inheritdoc}, or a
// label-style first line ending in ":" is left alone.
type PhpdocSummary struct{}

func (PhpdocSummary) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocSummaryFixer`
}

func (PhpdocSummary) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocSummaryFixer.php"
}

// summaryPunct are the sentence-final characters PHP-CS-Fixer accepts.
var summaryPunct = []string{".", ":", "。", "!", "?", "¡", "¿", "！", "？"}

func summaryCorrectlyFormatted(content string) bool {
	if strings.Contains(strings.ToLower(content), "{@inheritdoc}") {
		return true
	}
	for _, p := range summaryPunct {
		if strings.HasSuffix(content, p) {
			return true
		}
	}
	return false
}

func (PhpdocSummary) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		first, last := summaryRange(d)
		if first < 0 {
			return false
		}
		content := strings.TrimRight(d.inner[last].content, " \t")
		if summaryCorrectlyFormatted(content) {
			return false
		}
		// a multi-line summary whose first line is a label ("Example:") is skipped
		if first != last && strings.HasSuffix(strings.TrimRight(d.inner[first].content, " \t"), ":") {
			return false
		}
		d.inner[last].content = content + "."
		return true
	})
}

// summaryRange returns the first and last inner-line index of the short
// description (leading text lines before the first blank line or "@" tag), or
// (-1,-1) when there is none.
func summaryRange(d *docblock) (int, int) {
	first, last := -1, -1
	for i, l := range d.inner {
		c := strings.TrimSpace(l.content)
		if c == "" {
			if first >= 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(c, "@") {
			break
		}
		if first < 0 {
			first = i
		}
		last = i
	}
	return first, last
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagTypeFixer.php
//
// PhpdocTagType normalizes a standalone inline "{@inheritDoc}" to its annotation
// form "@inheritDoc" (the default tags => ['inheritDoc' => 'annotation']). Inline
// occurrences inside descriptive text are left as-is.
type PhpdocTagType struct{}

func (PhpdocTagType) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTagTypeFixer`
}

func (PhpdocTagType) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagTypeFixer.php"
}

var inlineInheritDocRe = regexp.MustCompile(`^\{@([a-zA-Z]+)\}$`)

func (PhpdocTagType) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			trimmed := strings.TrimLeft(l.content, " ")
			m := inlineInheritDocRe.FindStringSubmatch(strings.TrimRight(trimmed, " "))
			if m == nil || !strings.EqualFold(m[1], "inheritDoc") {
				continue
			}
			lead := l.content[:len(l.content)-len(strings.TrimLeft(l.content, " "))]
			d.inner[i].content = lead + "@" + m[1]
			changed = true
		}
		return changed
	})
}
