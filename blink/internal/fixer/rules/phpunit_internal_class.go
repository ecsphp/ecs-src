package rules

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitInternalClassFixer.php
//
// PhpUnitInternalClass marks PHPUnit test classes as @internal. The types option
// selects which classes to mark; the default (normal, final) skips abstract ones.
type PhpUnitInternalClass struct {
	types []string
}

var phpUnitInternalClassDefaultTypes = []string{"normal", "final"}

func (PhpUnitInternalClass) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitInternalClassFixer`
}

func (PhpUnitInternalClass) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitInternalClassFixer.php"
}

func (f PhpUnitInternalClass) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["types"]; ok {
		if list, ok := arrayNotationStringList(v); ok {
			f.types = append([]string{}, list...)
		}
	}
	return f
}

func (f PhpUnitInternalClass) Fix(s *tokens.Stream) bool {
	types := f.types
	if types == nil {
		types = phpUnitInternalClassDefaultTypes
	}
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		classIdx := puClassKeywordBefore(s, cls[0])
		if classIdx < 0 {
			continue
		}
		if p := sigPrev(s, classIdx); p >= 0 && kwIs(s.At(p), "new") {
			continue // skip anonymous classes
		}
		if !phpUnitInternalClassAllowed(s, classIdx, types) {
			continue
		}
		if puEnsureClassDocBlockAnnotation(s, classIdx, "internal", []string{"internal"}, nil) {
			changed = true
		}
	}
	return changed
}

func phpUnitInternalClassAllowed(s *tokens.Stream, classIdx int, types []string) bool {
	mods := puClassModifiers(s, classIdx)
	if mods["final"] {
		return slices.Contains(types, "final")
	}
	if mods["abstract"] {
		return slices.Contains(types, "abstract")
	}
	return slices.Contains(types, "normal")
}
