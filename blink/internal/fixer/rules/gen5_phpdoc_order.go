package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTypesOrderFixer.php
//
// PhpdocTypesOrder moves a null member to the end of a phpdoc type union,
// matching the @Symfony/@PhpCsFixer preset config (null_adjustment=always_last,
// sort_algorithm=none) that ECS uses: only null is relocated, the other members
// keep their order. Unions containing generics or DNF (`<`, `(`, `{`) are skipped.
type PhpdocTypesOrder struct{}

func (PhpdocTypesOrder) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTypesOrderFixer`
}

func (PhpdocTypesOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTypesOrderFixer.php"
}

func (PhpdocTypesOrder) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			trimmed := strings.TrimLeft(l.content, " ")
			m := phpdocTypeTagRe.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			sorted, ok := sortPhpdocUnion(m[2])
			if !ok || sorted == m[2] {
				continue
			}
			lead := l.content[:len(l.content)-len(trimmed)]
			d.inner[i].content = lead + m[1] + sorted + m[3]
			changed = true
		}
		return changed
	})
}

// sortPhpdocUnion moves any null member of a `|`-separated union to the end,
// preserving the order of the rest. ok is false when nothing moves or the type
// is skipped (generics/DNF, or a single member).
func sortPhpdocUnion(typ string) (string, bool) {
	if strings.ContainsAny(typ, "<({") {
		return typ, false
	}
	members := splitTopLevelUnion(typ)
	if len(members) < 2 {
		return typ, false
	}
	var nonNull, nulls []string
	for _, m := range members {
		if strings.EqualFold(normalizePhpdocCompare(m), "null") {
			nulls = append(nulls, m)
		} else {
			nonNull = append(nonNull, m)
		}
	}
	if len(nulls) == 0 {
		return typ, false
	}
	return strings.Join(append(nonNull, nulls...), "|"), true
}

// splitTopLevelUnion splits on `|` at bracket depth 0 only, so a `|` inside
// generics/array-shapes never splits a member.
func splitTopLevelUnion(typ string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(typ); i++ {
		switch typ[i] {
		case '<', '(', '[', '{':
			depth++
		case '>', ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '|':
			if depth == 0 {
				parts = append(parts, typ[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, typ[start:])
}

// normalizePhpdocCompare mirrors the fixer's /^\(*\??\\\?/ normalization used
// only to detect a null member; the member's original spelling is preserved.
func normalizePhpdocCompare(t string) string {
	t = strings.TrimLeft(t, "(")
	t = strings.TrimPrefix(t, "?")
	return strings.TrimPrefix(t, "\\")
}

// Swaps a `$var`-before-type @var/@type into type-before-`$var`, ported from the
// fixer's Preg replace (\h -> [ \t] for RE2).
var phpdocVarOrderRe = regexp.MustCompile(`(?i)(@(?:type|var)\s*)(\$\S+)([ \t]+)([^$](?:[^<\s]|<[^>]*>)*)(\s|\*)`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocVarAnnotationCorrectOrderFixer.php
//
// PhpdocVarAnnotationCorrectOrder fixes swapped @var/@type order, rewriting
// "@var $x int" to "@var int $x". Already-correct "@var Type $x" is a no-op.
type PhpdocVarAnnotationCorrectOrder struct{}

func (PhpdocVarAnnotationCorrectOrder) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocVarAnnotationCorrectOrderFixer`
}

func (PhpdocVarAnnotationCorrectOrder) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocVarAnnotationCorrectOrderFixer.php"
}

func (PhpdocVarAnnotationCorrectOrder) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.DocComment {
			continue
		}
		lower := strings.ToLower(t.Value)
		if !strings.Contains(lower, "@var") && !strings.Contains(lower, "@type") {
			continue
		}
		newVal := phpdocVarOrderRe.ReplaceAllString(t.Value, "${1}${4}${3}${2}${5}")
		if newVal != t.Value {
			s.SetValue(i, newVal)
			changed = true
		}
	}
	return changed
}
