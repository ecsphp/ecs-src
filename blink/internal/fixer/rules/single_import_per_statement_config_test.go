package rules

import "testing"

func TestSingleImportPerStatementConfig(t *testing.T) {
	t.Parallel()

	// group_to_single_imports:false keeps a brace group but still splits commas.
	f := SingleImportPerStatement{}.WithConfig(map[string]any{"group_to_single_imports": false})
	if _, changed := apply(t, f.(fixerRule), "<?php use A\\{B, C};"); changed {
		t.Fatal("group_to_single_imports:false must keep the group")
	}
	if got, changed := apply(t, f.(fixerRule), "<?php use A, B;"); !changed || got != "<?php use A;\nuse B;" {
		t.Fatalf("comma split changed=%v got=%q", changed, got)
	}

	// a group with per-member function/const modifiers expands each with its own.
	got, changed := apply(t, SingleImportPerStatement{}, "<?php use Foo\\{function a, const B, C};")
	if want := "<?php use function Foo\\a;\nuse const Foo\\B;\nuse Foo\\C;"; !changed || got != want {
		t.Fatalf("mixed group changed=%v got=%q want=%q", changed, got, want)
	}

	// an outer "function" modifier is carried to every member.
	got, changed = apply(t, SingleImportPerStatement{}, "<?php use function Foo\\{a, b};")
	if want := "<?php use function Foo\\a;\nuse function Foo\\b;"; !changed || got != want {
		t.Fatalf("outer modifier changed=%v got=%q want=%q", changed, got, want)
	}

	// aliases are preserved.
	got, changed = apply(t, SingleImportPerStatement{}, "<?php use Foo\\{Bar as Baz, Qux};")
	if want := "<?php use Foo\\Bar as Baz;\nuse Foo\\Qux;"; !changed || got != want {
		t.Fatalf("alias changed=%v got=%q want=%q", changed, got, want)
	}
}
