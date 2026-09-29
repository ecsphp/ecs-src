package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoTrailingWhitespaceFixer.php
//
// NoTrailingWhitespace removes spaces and tabs at the end of a line.
type NoTrailingWhitespace struct{}

func (NoTrailingWhitespace) Name() string {
	return `PhpCsFixer\Fixer\Whitespace\NoTrailingWhitespaceFixer`
}

func (NoTrailingWhitespace) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Whitespace/NoTrailingWhitespaceFixer.php"
}

func (NoTrailingWhitespace) Fix(s *tokens.Stream) bool {
	changed := false
	last := s.Len() - 1
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Whitespace {
			continue
		}
		v := stripTrailingWS(t.Value, i == last)
		if v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}

// stripTrailingWS drops a run of spaces/tabs that sits immediately before a
// (\r?\n) newline, and - only in the file's final token (isLast) - a run at the
// very end. It returns the input unchanged, with no allocation, when there is
// nothing to strip, matching the previous regexes `[ \t]+(\r?\n)` and `[ \t]+$`.
func stripTrailingWS(val string, isLast bool) string {
	for i := 0; i < len(val); {
		if val[i] == ' ' || val[i] == '\t' {
			start := i
			for i < len(val) && (val[i] == ' ' || val[i] == '\t') {
				i++
			}
			if trailing(val, i, isLast) {
				return stripTrailingBuild(val, start, isLast)
			}
			_ = start
		} else {
			i++
		}
	}
	return val
}

// stripTrailingBuild builds the stripped string once a first run to drop is
// found at firstDrop, copying the untouched prefix and continuing the scan.
func stripTrailingBuild(val string, firstDrop int, isLast bool) string {
	out := make([]byte, 0, len(val))
	out = append(out, val[:firstDrop]...)
	for i := firstDrop; i < len(val); {
		if val[i] == ' ' || val[i] == '\t' {
			start := i
			for i < len(val) && (val[i] == ' ' || val[i] == '\t') {
				i++
			}
			if !trailing(val, i, isLast) {
				out = append(out, val[start:i]...)
			}
		} else {
			out = append(out, val[i])
			i++
		}
	}
	return string(out)
}

// trailing reports whether a space/tab run ending at end should be dropped: it
// is directly before a newline, or at the file's end in the final token.
func trailing(val string, end int, isLast bool) bool {
	if end < len(val) && val[end] == '\n' {
		return true
	}
	if end+1 < len(val) && val[end] == '\r' && val[end+1] == '\n' {
		return true
	}
	return end == len(val) && isLast
}
