package rules

import (
	"regexp"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// applyToDocblocks runs fn over each doc comment, replacing it when fn reports a
// change.
func applyToDocblocks(s *tokens.Stream, fn func(*docblock) bool) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		d, ok := parseDoc(t.Value)
		if !ok {
			continue
		}
		if fn(&d) {
			s.SetValue(i, d.render())
			changed = true
		}
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTrimFixer.php
//
// PhpdocTrim removes blank lines at the start and end of a docblock.
type PhpdocTrim struct{}

func (PhpdocTrim) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTrimFixer`
}

func (PhpdocTrim) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTrimFixer.php"
}

func (PhpdocTrim) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		// php-cs-fixer trims empty lines around the content; a docblock with no
		// content at all is left untouched (e.g. "/**\n *\n */" after other fixers
		// stripped its description and tags), not collapsed to "/**\n */"
		hasContent := false
		for _, l := range d.inner {
			if strings.TrimSpace(l.content) != "" {
				hasContent = true
				break
			}
		}
		if !hasContent {
			return false
		}
		before := len(d.inner)
		for len(d.inner) > 0 && strings.TrimSpace(d.inner[0].content) == "" {
			d.inner = d.inner[1:]
		}
		for len(d.inner) > 0 && strings.TrimSpace(d.inner[len(d.inner)-1].content) == "" {
			d.inner = d.inner[:len(d.inner)-1]
		}
		return len(d.inner) != before
	})
}

// only when void/null is the whole return type - not part of a union like
// "null|string", which is a real nullable type
var emptyReturnRe = regexp.MustCompile(`(?i)^@return\s+(void|null)($|\s)`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoEmptyReturnFixer.php
//
// PhpdocNoEmptyReturn removes an "@return void" or "@return null" tag.
type PhpdocNoEmptyReturn struct{}

func (PhpdocNoEmptyReturn) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocNoEmptyReturnFixer`
}

func (PhpdocNoEmptyReturn) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoEmptyReturnFixer.php"
}

func (PhpdocNoEmptyReturn) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		kept := d.inner[:0:0]
		removed := false
		for _, l := range d.inner {
			if emptyReturnRe.MatchString(strings.TrimSpace(l.content)) {
				removed = true
				continue
			}
			kept = append(kept, l)
		}
		if removed {
			d.inner = kept
		}
		return removed
	})
}

var phpdocScalarMap = map[string]string{
	"boolean": "bool", "integer": "int", "double": "float",
	"real": "float", "str": "string", "callback": "callable",
}

var phpdocTypeTagRe = regexp.MustCompile(`(?i)^(@(?:param|return|var|throws|property|property-read|property-write|method)\s+)(\S+)(.*)$`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocScalarFixer.php
//
// PhpdocScalar normalizes scalar type aliases in phpdoc tags (integer -> int,
// boolean -> bool, double/real -> float, str -> string, callback -> callable).
// Option `types` restricts which aliases are converted; a nil set converts all of
// them (blink's default).
type PhpdocScalar struct {
	types map[string]bool
}

func (PhpdocScalar) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocScalarFixer`
}

func (PhpdocScalar) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocScalarFixer.php"
}

func (f PhpdocScalar) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := phpdocScalarStringList(config["types"]); ok {
		set := make(map[string]bool, len(list))
		for _, t := range list {
			set[t] = true
		}
		f.types = set
	}
	return f
}

func (f PhpdocScalar) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			m := phpdocTypeTagRe.FindStringSubmatch(strings.TrimLeft(l.content, " "))
			if m == nil {
				continue
			}
			newType := normalizeScalarType(m[2], f.types)
			if newType == m[2] {
				continue
			}
			lead := l.content[:len(l.content)-len(strings.TrimLeft(l.content, " "))]
			d.inner[i].content = lead + m[1] + newType + m[3]
			changed = true
		}
		return changed
	})
}

// normalizeScalarType replaces scalar aliases in a phpdoc type, handling unions,
// nullables and array suffixes ("integer[]|null" -> "int[]|null"). The lookup is
// case-sensitive so a class named like an alias (e.g. Laravel's "Str") is safe.
func normalizeScalarType(typ string, enabled map[string]bool) string {
	nullable := strings.HasPrefix(typ, "?")
	body := strings.TrimPrefix(typ, "?")
	parts := strings.Split(body, "|")
	for i, p := range parts {
		base, suffix := p, ""
		for strings.HasSuffix(base, "[]") {
			base = base[:len(base)-2]
			suffix = "[]" + suffix
		}
		if repl, ok := phpdocScalarMap[base]; ok && (enabled == nil || enabled[base]) {
			parts[i] = repl + suffix
		}
	}
	out := strings.Join(parts, "|")
	if nullable {
		out = "?" + out
	}
	return out
}

// phpdocScalarStringList reads a config value as a list of strings.
func phpdocScalarStringList(v any) ([]string, bool) {
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
