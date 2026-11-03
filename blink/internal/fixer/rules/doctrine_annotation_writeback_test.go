package rules

import "testing"

// The PHP Doctrine annotation fixers write every eligible docblock back from its
// annotation tokens, even when they changed none: a string the Doctrine lexer
// reads differently from PHP (a backslash does not escape a quote) comes back
// rewritten. The expected outputs come from the PHP fixers.
func TestDoctrineAnnotationAlwaysWritesBack(t *testing.T) {
	t.Parallel()
	const in = "<?php\n/** @A(x=\"a\\\"b=c\", y=1) */\nclass C {}\n"
	const want = "<?php\n/** @A(x=\"a\\\"b=c\"\", y=1) */\nclass C {}\n"
	for name, f := range map[string]fixerRule{
		"spaces":                 DoctrineAnnotationSpaces{}.WithConfig(map[string]any{}).(fixerRule),
		"indentation":            DoctrineAnnotationIndentation{},
		"array assignment":       DoctrineAnnotationArrayAssignment{}.WithConfig(map[string]any{}).(fixerRule),
		"array assignment colon": DoctrineAnnotationArrayAssignment{}.WithConfig(map[string]any{"operator": ":"}).(fixerRule),
		"braces":                 DoctrineAnnotationBraces{},
		"braces with_braces":     DoctrineAnnotationBraces{}.WithConfig(map[string]any{"syntax": "with_braces"}).(fixerRule),
	} {
		if got, changed := apply(t, f, in); !changed || got != want {
			t.Errorf("%s: changed=%v got=%q", name, changed, got)
		}
		if got, changed := apply(t, f, want); changed || got != want {
			t.Errorf("%s: not idempotent: changed=%v got=%q", name, changed, got)
		}
	}
}
