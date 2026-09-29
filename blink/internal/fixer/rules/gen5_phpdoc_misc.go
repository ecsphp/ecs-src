package rules

import (
	"regexp"
	"strings"

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
type PhpdocReturnSelfReference struct{}

func (PhpdocReturnSelfReference) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocReturnSelfReferenceFixer`
}

func (PhpdocReturnSelfReference) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocReturnSelfReferenceFixer.php"
}

func (PhpdocReturnSelfReference) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			trimmed := strings.TrimLeft(l.content, " ")
			m := phpdocReturnSelfRefRe.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			newType, ok := normalizeReturnSelfType(m[2])
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
func normalizeReturnSelfType(typ string) (string, bool) {
	parts := strings.Split(typ, "|")
	changed := false
	for i, p := range parts {
		if repl, ok := phpdocReturnSelfRefMap[strings.ToLower(p)]; ok {
			parts[i] = repl
			changed = true
		}
	}
	if !changed {
		return "", false
	}
	return strings.Join(parts, "|"), true
}
