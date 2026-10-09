package rules

import "testing"

func TestReturnTypeDeclarationSpaceBefore(t *testing.T) {
	t.Parallel()
	// default: no space before the colon, one space after
	got, changed := apply(t, ReturnTypeDeclaration{}, "<?php function f() : int {}")
	if want := "<?php function f(): int {}"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// space_before=one: exactly one space before the colon
	cfg := ReturnTypeDeclaration{}.WithConfig(map[string]any{"space_before": "one"})
	got, changed = apply(t, cfg, "<?php function f(): int {}")
	if want := "<?php function f() : int {}"; !changed || got != want {
		t.Fatalf("space_before=one: changed=%v got=%q want=%q", changed, got, want)
	}
}
