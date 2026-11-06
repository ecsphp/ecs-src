package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitSizeClassFixer.php
//
// PhpUnitSizeClass adds a size annotation (@small by default) to non-abstract
// PHPUnit test classes that have no @small/@medium/@large annotation yet.
type PhpUnitSizeClass struct {
	group string
}

var phpUnitSizeGroups = []string{"small", "medium", "large"}

func (PhpUnitSizeClass) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitSizeClassFixer`
}

func (PhpUnitSizeClass) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitSizeClassFixer.php"
}

func (f PhpUnitSizeClass) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["group"].(string); ok {
		f.group = v
	}
	return f
}

func (f PhpUnitSizeClass) Fix(s *tokens.Stream) bool {
	group := f.group
	if group == "" {
		group = "small"
	}
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		classIdx := puClassKeywordBefore(s, cls[0])
		if classIdx < 0 {
			continue
		}
		if puClassModifiers(s, classIdx)["abstract"] {
			continue
		}
		if puEnsureClassDocBlockAnnotation(s, classIdx, group, phpUnitSizeGroups, phpUnitSizeGroups) {
			changed = true
		}
	}
	return changed
}
