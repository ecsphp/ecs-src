package rules

import (
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitTestClassRequiresCoversFixer.php
//
// PhpUnitTestClassRequiresCovers adds a default @coversNothing annotation to
// non-abstract PHPUnit test classes that have no @covers* annotation.
type PhpUnitTestClassRequiresCovers struct{}

var phpUnitCoversAttributeShortNames = []string{
	"coversclass", "coversnothing", "coversmethod", "coversfunction", "coverstrait",
}

func (PhpUnitTestClassRequiresCovers) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitTestClassRequiresCoversFixer`
}

func (PhpUnitTestClassRequiresCovers) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitTestClassRequiresCoversFixer.php"
}

func (PhpUnitTestClassRequiresCovers) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range puClassesLastFirst(s) {
		classIdx := puClassKeywordBefore(s, cls[0])
		if classIdx < 0 {
			continue
		}
		if taIsAnonymousClass(s, classIdx) {
			continue
		}
		if puClassModifiers(s, classIdx)["abstract"] {
			continue
		}
		if puEnsureClassDocBlockAnnotation(s, classIdx, "coversNothing",
			[]string{"covers", "coversDefaultClass", "coversNothing"},
			phpUnitCoversAttributeShortNames) {
			changed = true
		}
	}
	return changed
}
