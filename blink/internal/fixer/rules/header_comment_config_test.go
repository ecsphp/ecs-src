package rules

import "testing"

// With no header configured the fixer does nothing, so its default configuration
// never touches a file. This keeps it parity-neutral: no prepared set enables it.
func TestHeaderCommentNoOpWithoutHeader(t *testing.T) {
	t.Parallel()
	const src = "<?php\n\n/**\n * new\n */\n$a;\n"
	for name, f := range map[string]fixerRule{
		"zero value":   HeaderComment{},
		"empty config": HeaderComment{}.WithConfig(map[string]any{}).(fixerRule),
		"empty header": HeaderComment{}.WithConfig(map[string]any{"header": ""}).(fixerRule),
		"blank header": HeaderComment{}.WithConfig(map[string]any{"header": "   "}).(fixerRule),
	} {
		if got, changed := apply(t, f, src); changed || got != src {
			t.Errorf("%s: changed=%v got=%q", name, changed, got)
		}
	}
}

// A header containing the comment terminator cannot be enclosed safely, so the
// fixer leaves the file unchanged (matching the upstream validation).
func TestHeaderCommentRejectsTerminator(t *testing.T) {
	t.Parallel()
	const src = "<?php\ndeclare(strict_types=1);\n\nnamespace A\\B;\n"
	f := HeaderComment{}.WithConfig(map[string]any{"header": "bad */ header"}).(fixerRule)
	if got, changed := apply(t, f, src); changed || got != src {
		t.Errorf("changed=%v got=%q", changed, got)
	}
}

// The options come through WithConfig the way the ECS loader supplies them.
func TestHeaderCommentWithConfig(t *testing.T) {
	t.Parallel()
	f := HeaderComment{}.WithConfig(map[string]any{
		"header":       "Made with love.",
		"comment_type": "PHPDoc",
		"location":     "after_open",
		"separate":     "bottom",
	}).(fixerRule)
	const src = "<?php\n\nnamespace A\\B;\n\necho 1;\n"
	const want = "<?php\n/**\n * Made with love.\n */\n\nnamespace A\\B;\n\necho 1;\n"
	got, changed := apply(t, f, src)
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}
