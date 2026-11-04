package rules

import (
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NumericLiteralSeparatorFixer.php
//
// NumericLiteralSeparator groups the digits of a numeric literal with "_"
// separators (the default use_separator strategy) or strips them (no_separator).
// Grouping is by base: binary in 8s and hex in 2s from the right, octal and
// decimal integer/exponent parts in 3s from the right, and the fractional part
// in 3s from the left.
type NumericLiteralSeparator struct {
	// strategy is "use_separator" (default, empty value) or "no_separator".
	strategy string
	// overrideExisting re-groups a literal that already has separators.
	overrideExisting bool
}

func (NumericLiteralSeparator) Name() string {
	return `PhpCsFixer\Fixer\Basic\NumericLiteralSeparatorFixer`
}

func (NumericLiteralSeparator) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/NumericLiteralSeparatorFixer.php"
}

func (f NumericLiteralSeparator) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["strategy"].(string); ok {
		f.strategy = v
	}
	if v, ok := config["override_existing"].(bool); ok {
		f.overrideExisting = v
	}
	return f
}

func (f NumericLiteralSeparator) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Number {
			continue
		}
		v := s.At(i).Value
		if nv := f.formatValue(v); nv != v {
			s.SetValue(i, nv)
			changed = true
		}
	}
	return changed
}

var numericFloatParts = regexp.MustCompile(`^([0-9_]*)(\.([0-9_]*))?([eE]([+-]?[0-9_]+))?$`)

func (f NumericLiteralSeparator) formatValue(v string) string {
	if f.strategy == "no_separator" {
		return strings.ReplaceAll(v, "_", "")
	}
	if strings.Contains(v, "_") {
		if !f.overrideExisting {
			return v
		}
		v = strings.ReplaceAll(v, "_", "")
	}

	lower := strings.ToLower(v)
	switch {
	case strings.HasPrefix(lower, "0b"):
		return insertEveryRight(v, 8, 2)
	case strings.HasPrefix(lower, "0x"):
		return insertEveryRight(v, 2, 2)
	case strings.HasPrefix(lower, "0o"):
		return insertEveryRight(v, 3, 2)
	case len(v) > 1 && v[0] == '0' && !strings.ContainsAny(v, ".eE"):
		return insertEveryRight(v, 3, 1) // legacy octal
	}

	m := numericFloatParts.FindStringSubmatch(v)
	if m == nil {
		return v
	}
	out := insertEveryRight(m[1], 3, 0)
	if m[2] != "" {
		out += "." + insertEveryLeft(m[3], 3)
	}
	if m[4] != "" {
		marker := m[4][:1]
		out += marker + insertEveryRightSigned(m[5], 3)
	}
	return out
}

// insertEveryRight groups digits in runs of size from the right, keeping the
// first offset characters (a base prefix) ungrouped.
func insertEveryRight(v string, size, offset int) string {
	runes := []byte(v)
	for i := len(runes) - size; i > offset; i -= size {
		runes = append(runes[:i], append([]byte{'_'}, runes[i:]...)...)
	}
	return string(runes)
}

// insertEveryRightSigned is insertEveryRight for an exponent that may carry a
// leading sign, which is kept ungrouped.
func insertEveryRightSigned(v string, size int) string {
	offset := 0
	if len(v) > 0 && (v[0] == '+' || v[0] == '-') {
		offset = 1
	}
	return insertEveryRight(v, size, offset)
}

// insertEveryLeft groups digits in runs of size from the left (for a fractional
// part), leaving the final partial group without a trailing separator.
func insertEveryLeft(v string, size int) string {
	runes := []byte(v)
	for i := size; i < len(runes); i += size + 1 {
		runes = append(runes[:i], append([]byte{'_'}, runes[i:]...)...)
	}
	return string(runes)
}
