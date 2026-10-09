package rules

import "testing"

func TestVisibilityRequiredElements(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass C\n{\n    function f() {}\n    const A = 1;\n}\n"

	// default: visibility added to methods and constants alike
	got, changed := apply(t, VisibilityRequired{}, src)
	if want := "<?php\nclass C\n{\n    public function f() {}\n    public const A = 1;\n}\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// elements=[method]: only methods are targeted, the const is left alone
	cfg := VisibilityRequired{}.WithConfig(map[string]any{"elements": []any{"method"}})
	got, changed = apply(t, cfg, src)
	if want := "<?php\nclass C\n{\n    public function f() {}\n    const A = 1;\n}\n"; !changed || got != want {
		t.Fatalf("elements=[method]: changed=%v got=%q want=%q", changed, got, want)
	}
}
