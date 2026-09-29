package rules

import (
	"regexp"
	"strings"

	"blink/internal/tokens"
)

// inheritDocRe matches an "@inheritdoc" tag token (any case), whether standalone
// ("@inheritdoc") or inline ("{@inheritdoc}"). The trailing \b keeps
// "@inheritdocs" untouched, mirroring the default ['inheritDoc'] tag set.
var inheritDocRe = regexp.MustCompile(`(?i)@inheritdoc\b`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagCasingFixer.php
//
// PhpdocTagCasing fixes the casing of the @inheritDoc tag (the default tag set),
// in both annotation and inline positions.
type PhpdocTagCasing struct{}

func (PhpdocTagCasing) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTagCasingFixer`
}

func (PhpdocTagCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagCasingFixer.php"
}

func (PhpdocTagCasing) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			fixed := inheritDocRe.ReplaceAllString(l.content, "@inheritDoc")
			if fixed != l.content {
				d.inner[i].content = fixed
				changed = true
			}
		}
		return changed
	})
}

// inlineTagRe ports PhpdocInlineTagNormalizerFixer's pattern: it matches an inline
// tag written as "@{tag}" or "{@tag}" (with stray braces or spaces) for a known
// inline tag, capturing the tag word (case preserved) and its inner text.
var inlineTagRe = regexp.MustCompile(`(?i)(?:@\{+|\{+[ \t]*@)[ \t]*(example|id|internal|inheritdoc|inheritdocs|link|source|toc|tutorial|see)\b([^}]*)\}+`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocInlineTagNormalizerFixer.php
//
// PhpdocInlineTagNormalizer normalizes inline tags: it moves "@" inside the
// braces, collapses repeated braces and trims inner spaces ("{ @see X }" ->
// "{@see X}"). The tag word's casing is left untouched (that is PhpdocTagCasing's
// job).
type PhpdocInlineTagNormalizer struct{}

func (PhpdocInlineTagNormalizer) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocInlineTagNormalizerFixer`
}

func (PhpdocInlineTagNormalizer) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocInlineTagNormalizerFixer.php"
}

func (PhpdocInlineTagNormalizer) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			fixed := inlineTagRe.ReplaceAllStringFunc(l.content, func(m string) string {
				sub := inlineTagRe.FindStringSubmatch(m)
				tag := sub[1]
				doc := strings.TrimSpace(sub[2])
				if doc == "" {
					return "{@" + tag + "}"
				}
				return "{@" + tag + " " + doc + "}"
			})
			if fixed != l.content {
				d.inner[i].content = fixed
				changed = true
			}
		}
		return changed
	})
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoDuplicateTypesFixer.php
//
// PhpdocNoDuplicateTypes removes duplicate members from a phpdoc type union
// ("int|int|string" -> "int|string"), comparing case-insensitively and keeping
// the first occurrence.
type PhpdocNoDuplicateTypes struct{}

func (PhpdocNoDuplicateTypes) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocNoDuplicateTypesFixer`
}

func (PhpdocNoDuplicateTypes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocNoDuplicateTypesFixer.php"
}

func (PhpdocNoDuplicateTypes) Fix(s *tokens.Stream) bool {
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			trimmed := strings.TrimLeft(l.content, " ")
			m := phpdocTypeTagRe.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			newType := dedupeUnionTypes(m[2])
			if newType == m[2] {
				continue
			}
			lead := l.content[:len(l.content)-len(trimmed)]
			d.inner[i].content = lead + m[1] + newType + m[3]
			changed = true
		}
		return changed
	})
}

// dedupeUnionTypes removes case-insensitive duplicate members from a "|" union,
// keeping the first spelling. Types with nested brackets (generics, DNF) are left
// untouched, since a "|" inside them is not a top-level separator.
func dedupeUnionTypes(typ string) string {
	nullable := strings.HasPrefix(typ, "?")
	body := strings.TrimPrefix(typ, "?")
	if strings.ContainsAny(body, "<>(){}[]") {
		return typ
	}
	parts := strings.Split(body, "|")
	if len(parts) < 2 {
		return typ
	}
	seen := make(map[string]bool, len(parts))
	kept := parts[:0:0]
	for _, p := range parts {
		key := strings.ToLower(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		kept = append(kept, p)
	}
	out := strings.Join(kept, "|")
	if nullable {
		out = "?" + out
	}
	return out
}
