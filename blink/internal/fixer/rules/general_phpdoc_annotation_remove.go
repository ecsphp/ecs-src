package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/GeneralPhpdocAnnotationRemoveFixer.php
//
// GeneralPhpdocAnnotationRemove removes the configured phpdoc annotations. The
// zero value strips author, package, group and category (ECS's common set); the
// registry uses PHP-CS-Fixer's default, an empty list.
//
// Option `annotations` overrides the annotation set (an empty list removes
// nothing). Option `case_sensitive` switches to exact-case matching; the zero
// value keeps blink's case-insensitive match.
type GeneralPhpdocAnnotationRemove struct {
	annotations   []string
	caseSensitive bool
}

func (GeneralPhpdocAnnotationRemove) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\GeneralPhpdocAnnotationRemoveFixer`
}

func (GeneralPhpdocAnnotationRemove) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/GeneralPhpdocAnnotationRemoveFixer.php"
}

// generalPhpdocAnnotationsToRemove is the configured annotation set, matching
// ECS's psr12+common (SetList::DOCBLOCK).
var generalPhpdocAnnotationsToRemove = []string{"author", "package", "group", "category"}

func (f GeneralPhpdocAnnotationRemove) WithConfig(config map[string]any) fixer.Fixer {
	if list, ok := generalPhpdocAnnotationStringList(config["annotations"]); ok {
		f.annotations = list
	}
	if cs, ok := config["case_sensitive"].(bool); ok {
		f.caseSensitive = cs
	}
	return f
}

func (f GeneralPhpdocAnnotationRemove) Fix(s *tokens.Stream) bool {
	tags := f.annotations
	if tags == nil {
		tags = generalPhpdocAnnotationsToRemove
	}
	if len(tags) == 0 {
		return false
	}
	return applyToDocblocks(s, func(d *docblock) bool {
		kept := d.inner[:0:0]
		removed := false
		for _, l := range d.inner {
			if generalPhpdocLineHasTag(l.content, tags, f.caseSensitive) {
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

func generalPhpdocLineHasTag(content string, tags []string, caseSensitive bool) bool {
	trimmed := strings.TrimLeft(content, " ")
	name := ""
	if strings.HasPrefix(trimmed, "@") {
		j := 1
		for j < len(trimmed) && (isIdentByte(trimmed[j]) || trimmed[j] == '-') {
			j++
		}
		name = trimmed[1:j]
	}
	if name == "" {
		return false
	}
	for _, t := range tags {
		if caseSensitive {
			if name == t {
				return true
			}
		} else if strings.EqualFold(name, t) {
			return true
		}
	}
	return false
}

// generalPhpdocAnnotationStringList reads a config value as a list of strings.
func generalPhpdocAnnotationStringList(v any) ([]string, bool) {
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
