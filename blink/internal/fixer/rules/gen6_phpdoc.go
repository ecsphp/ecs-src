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

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocOrderFixer.php
//
// PhpdocOrder orders @param, @throws and @return tags into that sequence (the
// build's default). Only those tag blocks are permuted; summary, blank lines and
// other tags keep their positions.
type PhpdocOrder struct{}

func (PhpdocOrder) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocOrderFixer`
}

func (PhpdocOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocOrderFixer.php"
}

var phpdocTagNameRe = regexp.MustCompile(`^@([a-zA-Z][a-zA-Z0-9_-]*)`)

// phpdocOrderRank returns the sort rank of a tag and whether it is one of the
// reordered tags. phpstan-/psalm- prefixed variants share their base rank.
func phpdocOrderRank(tag string) (int, bool) {
	base := strings.ToLower(tag)
	base = strings.TrimPrefix(base, "phpstan-")
	base = strings.TrimPrefix(base, "psalm-")
	switch base {
	case "param":
		return 0, true
	case "throws":
		return 1, true
	case "return":
		return 2, true
	}
	return 0, false
}

type phpdocBlock struct {
	lines  []docLine
	rank   int
	target bool
}

func (PhpdocOrder) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		if d.single {
			return false
		}
		blocks := splitPhpdocBlocks(d.inner)

		var targetPos []int
		var targetBlocks []phpdocBlock
		for i, b := range blocks {
			if b.target {
				targetPos = append(targetPos, i)
				targetBlocks = append(targetBlocks, b)
			}
		}
		if len(targetBlocks) < 2 {
			return false
		}
		sorted := make([]phpdocBlock, len(targetBlocks))
		copy(sorted, targetBlocks)
		stableSortByRank(sorted)
		if sameBlockOrder(targetBlocks, sorted) {
			return false
		}
		for k, pos := range targetPos {
			blocks[pos] = sorted[k]
		}
		var inner []docLine
		for _, b := range blocks {
			inner = append(inner, b.lines...)
		}
		d.inner = inner
		return true
	})
}

// splitPhpdocBlocks groups inner lines: each "@tag" line plus its continuation
// lines forms one block; every other line is its own block.
func splitPhpdocBlocks(inner []docLine) []phpdocBlock {
	var blocks []phpdocBlock
	for i := 0; i < len(inner); {
		c := strings.TrimSpace(inner[i].content)
		if m := phpdocTagNameRe.FindStringSubmatch(c); m != nil {
			rank, target := phpdocOrderRank(m[1])
			j := i + 1
			for j < len(inner) {
				cj := strings.TrimSpace(inner[j].content)
				if cj == "" || strings.HasPrefix(cj, "@") {
					break
				}
				j++
			}
			blocks = append(blocks, phpdocBlock{lines: inner[i:j], rank: rank, target: target})
			i = j
			continue
		}
		blocks = append(blocks, phpdocBlock{lines: inner[i : i+1]})
		i++
	}
	return blocks
}

// stableSortByRank orders blocks by rank, keeping equal-rank blocks in place.
func stableSortByRank(blocks []phpdocBlock) {
	for i := 1; i < len(blocks); i++ {
		for j := i; j > 0 && blocks[j-1].rank > blocks[j].rank; j-- {
			blocks[j-1], blocks[j] = blocks[j], blocks[j-1]
		}
	}
}

func sameBlockOrder(a, b []phpdocBlock) bool {
	for i := range a {
		if len(a[i].lines) != len(b[i].lines) {
			return false
		}
		for k := range a[i].lines {
			if a[i].lines[k] != b[i].lines[k] {
				return false
			}
		}
	}
	return true
}
