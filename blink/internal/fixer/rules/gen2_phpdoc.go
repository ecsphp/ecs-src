package rules

import (
	"regexp"
	"sort"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocIndentFixer.php
//
// PhpdocIndent aligns a docblock's "*" continuation lines and closing "*/" to
// the indentation of the code the docblock documents, one space after the
// opening "/**"'s column.
type PhpdocIndent struct{}

func (PhpdocIndent) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocIndentFixer`
}

func (PhpdocIndent) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocIndentFixer.php"
}

func (PhpdocIndent) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		if !docblockAtLineStart(s, i) {
			continue
		}
		// align to the element the docblock documents (the next line), not the
		// docblock's own possibly-wrong indent
		indent, ok := docblockTargetIndent(s, i)
		if !ok {
			continue
		}
		d, ok := parseDoc(t.Value)
		if !ok || d.single {
			continue
		}
		if reindentDocblock(&d, indent) {
			s.SetValue(i, d.render())
			changed = true
		}
		// move the opening "/**" line to the same indent
		if s.At(i-1).Kind == token.Whitespace {
			v := s.At(i - 1).Value
			nl := strings.LastIndexByte(v, '\n')
			if nl >= 0 && v[nl+1:] != indent {
				s.SetValue(i-1, v[:nl+1]+indent)
				changed = true
			}
		}
	}
	return changed
}

// docblockAtLineStart reports whether the docblock at i begins its own line.
func docblockAtLineStart(s *tokens.Stream, i int) bool {
	if i == 0 {
		return false
	}
	prev := s.At(i - 1)
	return prev.Kind == token.Whitespace && strings.IndexByte(prev.Value, '\n') >= 0
}

// docblockTargetIndent returns the indentation of the structural element the
// docblock at i documents - the indent of the line holding the next significant
// token. ok is false when there is none.
func docblockTargetIndent(s *tokens.Stream, i int) (string, bool) {
	if i+1 >= s.Len() || s.At(i+1).Kind != token.Whitespace {
		return "", false
	}
	v := s.At(i + 1).Value
	nl := strings.LastIndexByte(v, '\n')
	if nl < 0 {
		return "", false
	}
	return v[nl+1:], true
}

// docblockLineIndent returns the indentation of the line the docblock at index i
// sits on, taken from the trailing whitespace of the preceding whitespace token.
// ok is false when the docblock is not at the start of its line.
func docblockLineIndent(s *tokens.Stream, i int) (string, bool) {
	if i == 0 {
		return "", false
	}
	prev := s.At(i - 1)
	if prev.Kind != token.Whitespace {
		return "", false
	}
	nl := strings.LastIndexByte(prev.Value, '\n')
	if nl < 0 {
		return "", false
	}
	return prev.Value[nl+1:], true
}

// reindentDocblock rewrites every interior line's "*" and the closing "*/" so
// the star sits one column past the indent. Reports whether anything changed.
func reindentDocblock(d *docblock, indent string) bool {
	changed := false
	for k, l := range d.inner {
		star := strings.IndexByte(l.prefix, '*')
		if star < 0 {
			continue
		}
		newPrefix := indent + " *" + l.prefix[star+1:]
		if newPrefix != l.prefix {
			d.inner[k].prefix = newPrefix
			changed = true
		}
	}
	if strings.TrimSpace(d.close) == "*/" {
		newClose := indent + " */"
		if newClose != d.close {
			d.close = newClose
			changed = true
		}
	}
	return changed
}

// coversRe matches a "@covers" annotation line, capturing the value after the
// tag. The value is required so "@coversNothing"/"@coversDefaultClass" and a
// bare "@covers" are left untouched.
var coversRe = regexp.MustCompile(`^@covers\s+(\S.*?)\s*$`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocOrderByValueFixer.php
//
// PhpdocOrderByValue sorts the annotations of the configured tag by their value.
// The default tag set is ["covers"]; a contiguous run of "@covers" lines is
// stable-sorted alphabetically (case-insensitively) by the value after the tag.
type PhpdocOrderByValue struct{}

func (PhpdocOrderByValue) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocOrderByValueFixer`
}

func (PhpdocOrderByValue) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocOrderByValueFixer.php"
}

func (PhpdocOrderByValue) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		n := len(d.inner)
		for start := 0; start < n; {
			if coversValue(d.inner[start]) == "" {
				start++
				continue
			}
			end := start + 1
			for end < n && coversValue(d.inner[end]) != "" {
				end++
			}
			if end-start > 1 && sortCoversRun(d.inner[start:end]) {
				changed = true
			}
			start = end
		}
		return changed
	})
}

// coversValue returns the lowercased value of a "@covers" annotation line, or ""
// when the line is not a "@covers" annotation with a value.
func coversValue(l docLine) string {
	m := coversRe.FindStringSubmatch(strings.TrimSpace(l.content))
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}

// sortCoversRun stable-sorts a run of "@covers" lines by their value and reports
// whether the order changed.
func sortCoversRun(run []docLine) bool {
	keys := make([]string, len(run))
	for i, l := range run {
		keys[i] = coversValue(l)
	}
	if sort.StringsAreSorted(keys) {
		return false
	}
	sort.SliceStable(run, func(i, j int) bool {
		return coversValue(run[i]) < coversValue(run[j])
	})
	return true
}
