package rules

import (
	"regexp"
	"strings"

	"blink/internal/fixer"
	"blink/internal/tokens"
)

// phpdocReturnSelfRefMap is the fixer's default "replacements" map, verbatim from
// source. Keys are matched case-insensitively (the fixer lowercases the type).
var phpdocReturnSelfRefMap = map[string]string{
	"this":    "$this",
	"@this":   "$this",
	"$self":   "self",
	"@self":   "self",
	"$static": "static",
	"@static": "static",
}

var phpdocReturnSelfRefRe = regexp.MustCompile(`(?i)^(@return\s+)(\S+)(.*)$`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocReturnSelfReferenceFixer.php
//
// PhpdocReturnSelfReference normalizes self-reference return types in "@return"
// (this -> $this, @self -> self, $static -> static, ...). Each union member that
// exactly matches an alias (case-insensitive) is replaced; the description after
// the type is preserved.
//
// Option `replacements` maps a self-reference alias (lowercased) to its target
// type; a nil map keeps the default set above.
type PhpdocReturnSelfReference struct {
	replacements map[string]string
}

func (PhpdocReturnSelfReference) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocReturnSelfReferenceFixer`
}

func (PhpdocReturnSelfReference) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocReturnSelfReferenceFixer.php"
}

func (f PhpdocReturnSelfReference) WithConfig(config map[string]any) fixer.Fixer {
	if raw, ok := config["replacements"].(map[string]any); ok {
		m := make(map[string]string, len(raw))
		for k, v := range raw {
			if s, ok := v.(string); ok {
				m[strings.ToLower(k)] = s
			}
		}
		f.replacements = m
	}
	return f
}

func (f PhpdocReturnSelfReference) Fix(s *tokens.Stream) bool {
	repl := f.replacements
	if repl == nil {
		repl = phpdocReturnSelfRefMap
	}
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			trimmed := strings.TrimLeft(l.content, " ")
			m := phpdocReturnSelfRefRe.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			newType, ok := normalizeReturnSelfType(m[2], repl)
			if !ok {
				continue
			}
			lead := l.content[:len(l.content)-len(trimmed)]
			d.inner[i].content = lead + m[1] + newType + m[3]
			changed = true
		}
		return changed
	})
}

// normalizeReturnSelfType replaces each union member equal (case-insensitive) to
// a configured alias. Rejoining with "|" is lossless, so non-matching or complex
// types round-trip unchanged; ok is false when nothing was replaced.
func normalizeReturnSelfType(typ string, replacements map[string]string) (string, bool) {
	parts := strings.Split(typ, "|")
	changed := false
	for i, p := range parts {
		if repl, ok := replacements[strings.ToLower(p)]; ok {
			parts[i] = repl
			changed = true
		}
	}
	if !changed {
		return "", false
	}
	return strings.Join(parts, "|"), true
}
