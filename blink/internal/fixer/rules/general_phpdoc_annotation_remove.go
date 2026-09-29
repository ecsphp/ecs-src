package rules

import (
	"strings"

	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/GeneralPhpdocAnnotationRemoveFixer.php
//
// GeneralPhpdocAnnotationRemove removes the configured phpdoc annotations. ECS's
// psr12+common (via SetList::DOCBLOCK) configures it to strip author, package,
// group and category, so those are the default set here.
type GeneralPhpdocAnnotationRemove struct{}

func (GeneralPhpdocAnnotationRemove) Name() string {
	return `PhpCsFixer\Fixer\Phpdoc\GeneralPhpdocAnnotationRemoveFixer`
}

func (GeneralPhpdocAnnotationRemove) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Phpdoc/GeneralPhpdocAnnotationRemoveFixer.php"
}

// generalPhpdocAnnotationsToRemove is the configured annotation set, matching
// ECS's psr12+common (SetList::DOCBLOCK).
var generalPhpdocAnnotationsToRemove = []string{"author", "package", "group", "category"}

func (GeneralPhpdocAnnotationRemove) Fix(s *tokens.Stream) bool {
	if len(generalPhpdocAnnotationsToRemove) == 0 {
		return false
	}
	return applyToDocblocks(s, func(d *docblock) bool {
		kept := d.inner[:0:0]
		removed := false
		for _, l := range d.inner {
			if generalPhpdocLineHasTag(l.content, generalPhpdocAnnotationsToRemove) {
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

func generalPhpdocLineHasTag(content string, tags []string) bool {
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
		if strings.EqualFold(name, t) {
			return true
		}
	}
	return false
}
