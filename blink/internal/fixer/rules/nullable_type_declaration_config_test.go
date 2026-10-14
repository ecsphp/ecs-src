package rules

import "testing"

func TestNullableTypeDeclarationConfig(t *testing.T) {
	t.Parallel()

	src := "<?php function f(?int $x): ?string {}"

	// default (question_mark) leaves the "?" syntax as is.
	if _, changed := apply(t, NullableTypeDeclaration{}, src); changed {
		t.Fatal("question_mark default must leave ?int / ?string untouched")
	}

	// union rewrites "?T" to "T|null".
	f := NullableTypeDeclaration{}.WithConfig(map[string]any{"syntax": "union"})
	got, changed := apply(t, f.(fixerRule), src)
	if want := "<?php function f(int|null $x): string|null {}"; !changed || got != want {
		t.Fatalf("union changed=%v got=%q want=%q", changed, got, want)
	}

	// union still collapses nothing when the type is already a union.
	already := "<?php function f(int|null $x) {}"
	if _, changed := apply(t, f.(fixerRule), already); changed {
		t.Fatal("union must leave an existing union alone")
	}
}
