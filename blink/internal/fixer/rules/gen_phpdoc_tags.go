package rules

import (
	"regexp"
	"strings"

	"blink/internal/fixer"
	"blink/internal/tokens"
)

// inheritDocRe matches an "@inheritdoc" tag token (any case), whether standalone
// ("@inheritdoc") or inline ("{@inheritdoc}"). The trailing \b keeps
// "@inheritdocs" untouched, mirroring the default ['inheritDoc'] tag set.
var inheritDocRe = regexp.MustCompile(`(?i)@inheritdoc\b`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagCasingFixer.php
//
// PhpdocTagCasing fixes the casing of the configured tags (default ["inheritDoc"]),
// in both annotation and inline positions. Each tag is matched case-insensitively
// and rewritten to the configured spelling.
type PhpdocTagCasing struct {
	tags []string
}

func (PhpdocTagCasing) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocTagCasingFixer`
}

func (PhpdocTagCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocTagCasingFixer.php"
}

func (f PhpdocTagCasing) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := phpdocTagsStringList(config["tags"]); ok {
		f.tags = list
	}
	return f
}

func (f PhpdocTagCasing) Fix(s *tokens.Stream) bool {
	type replacement struct {
		re *regexp.Regexp
		to string
	}
	var reps []replacement
	if f.tags == nil {
		reps = []replacement{{inheritDocRe, "@inheritDoc"}}
	} else {
		for _, tag := range f.tags {
			reps = append(reps, replacement{
				regexp.MustCompile(`(?i)@` + regexp.QuoteMeta(tag) + `\b`),
				"@" + tag,
			})
		}
	}
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			fixed := l.content
			for _, r := range reps {
				fixed = r.re.ReplaceAllString(fixed, r.to)
			}
			if fixed != l.content {
				d.inner[i].content = fixed
				changed = true
			}
		}
		return changed
	})
}

// phpdocTagsStringList reads a config value as a list of strings.
func phpdocTagsStringList(v any) ([]string, bool) {
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

// inlineTagRe ports PhpdocInlineTagNormalizerFixer's pattern: it matches an inline
// tag written as "@{tag}" or "{@tag}" (with stray braces or spaces) for a known
// inline tag, capturing the tag word (case preserved) and its inner text.
var inlineTagRe = regexp.MustCompile(`(?i)(?:@\{+|\{+[ \t]*@)[ \t]*(example|id|internal|inheritdoc|inheritdocs|link|source|toc|tutorial|see)\b([^}]*)\}+`)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocInlineTagNormalizerFixer.php
//
// PhpdocInlineTagNormalizer normalizes inline tags: it moves "@" inside the
// braces, collapses repeated braces and trims inner spaces ("{ @see X }" ->
// "{@see X}"). The tag word's casing is left untouched (that is PhpdocTagCasing's
// job). Option `tags` selects which inline tags to normalize; a nil list keeps
// blink's default set.
type PhpdocInlineTagNormalizer struct {
	tags []string
}

func (PhpdocInlineTagNormalizer) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\PhpdocInlineTagNormalizerFixer`
}

func (PhpdocInlineTagNormalizer) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/PhpdocInlineTagNormalizerFixer.php"
}

func (f PhpdocInlineTagNormalizer) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := phpdocTagsStringList(config["tags"]); ok {
		f.tags = list
	}
	return f
}

func (f PhpdocInlineTagNormalizer) Fix(s *tokens.Stream) bool {
	re := inlineTagRe
	if f.tags != nil {
		if len(f.tags) == 0 {
			return false
		}
		re = phpdocInlineTagRe(f.tags)
	}
	return applyToDocblocks(s, func(d *docblock) bool {
		changed := false
		for i, l := range d.inner {
			fixed := re.ReplaceAllStringFunc(l.content, func(m string) string {
				sub := re.FindStringSubmatch(m)
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

// phpdocInlineTagRe builds the inline-tag pattern for a configured tag list,
// mirroring inlineTagRe's shape.
func phpdocInlineTagRe(tags []string) *regexp.Regexp {
	quoted := make([]string, len(tags))
	for i, tag := range tags {
		quoted[i] = regexp.QuoteMeta(tag)
	}
	return regexp.MustCompile(`(?i)(?:@\{+|\{+[ \t]*@)[ \t]*(` + strings.Join(quoted, "|") + `)\b([^}]*)\}+`)
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
