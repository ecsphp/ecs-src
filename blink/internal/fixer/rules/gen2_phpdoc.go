package rules

import (
	"sort"
	"strings"

	"blink/internal/fixer"
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

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocOrderByValueFixer.php
//
// PhpdocOrderByValue sorts the annotations of the configured tags by their value.
// Each contiguous run of a tag's lines is stable-sorted alphabetically
// (case-insensitively) by its comparable value. Option `annotations` selects the
// tags; a nil list keeps the default ["covers"].
type PhpdocOrderByValue struct {
	annotations []string
}

func (PhpdocOrderByValue) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocOrderByValueFixer`
}

func (PhpdocOrderByValue) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocOrderByValueFixer.php"
}

func (f PhpdocOrderByValue) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := phpdocOrderByValueStringList(config["annotations"]); ok {
		f.annotations = list
	}
	return f
}

func (f PhpdocOrderByValue) Fix(s *tokens.Stream) bool {
	annotations := f.annotations
	if annotations == nil {
		annotations = []string{"covers"}
	}
	if len(annotations) == 0 {
		return false
	}
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for _, anno := range annotations {
			if phpdocOrderByValueSortType(d, anno) {
				changed = true
			}
		}
		return changed
	})
}

// phpdocOrderByValueSortType stable-sorts each contiguous run of the given
// annotation's lines by comparable value.
func phpdocOrderByValueSortType(d *docblock, anno string) bool {
	changed := false
	n := len(d.inner)
	for start := 0; start < n; {
		if _, ok := phpdocOrderByValueComparable(d.inner[start].content, anno); !ok {
			start++
			continue
		}
		end := start + 1
		for end < n {
			if _, ok := phpdocOrderByValueComparable(d.inner[end].content, anno); !ok {
				break
			}
			end++
		}
		if end-start > 1 && phpdocOrderByValueSortRun(d.inner[start:end], anno) {
			changed = true
		}
		start = end
	}
	return changed
}

// phpdocOrderByValueComparable returns the lowercased sort key for an annotation
// line, or ok=false when the line is not the given annotation with a value.
func phpdocOrderByValueComparable(content, anno string) (string, bool) {
	trimmed := strings.TrimSpace(content)
	prefix := "@" + anno
	if !strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(prefix)) {
		return "", false
	}
	rest := trimmed[len(prefix):]
	if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return "", false
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	switch strings.ToLower(anno) {
	case "property", "property-read", "property-write":
		if i := strings.IndexByte(rest, '$'); i >= 0 {
			rest = rest[i+1:]
		}
		if j := strings.IndexAny(rest, " \t"); j >= 0 {
			rest = rest[:j]
		}
	case "method":
		if i := strings.IndexByte(rest, '('); i >= 0 {
			rest = rest[:i]
		}
		if fields := strings.Fields(rest); len(fields) > 0 {
			rest = fields[len(fields)-1]
		}
	}
	return strings.ToLower(rest), true
}

// phpdocOrderByValueSortRun stable-sorts a run of annotation lines by comparable
// value and reports whether the order changed.
func phpdocOrderByValueSortRun(run []docLine, anno string) bool {
	value := func(l docLine) string {
		v, _ := phpdocOrderByValueComparable(l.content, anno)
		return v
	}
	keys := make([]string, len(run))
	for i, l := range run {
		keys[i] = value(l)
	}
	if sort.StringsAreSorted(keys) {
		return false
	}
	sort.SliceStable(run, func(i, j int) bool {
		return value(run[i]) < value(run[j])
	})
	return true
}

// phpdocOrderByValueStringList reads a config value as a list of strings.
func phpdocOrderByValueStringList(v any) ([]string, bool) {
	switch list := v.(type) {
	case []string:
		return append([]string{}, list...), true
	case []any:
		out := make([]string, 0, len(list))
		for _, e := range list {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out, true
	}
	return nil, false
}
