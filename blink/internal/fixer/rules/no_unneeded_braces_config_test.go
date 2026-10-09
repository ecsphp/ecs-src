package rules

import "testing"

func TestNoUnneededBracesConfig(t *testing.T) {
	t.Parallel()
	src := "<?php\nnamespace Foo {\n    echo 1;\n}"
	if _, changed := apply(t, NoUnneededBraces{}, src); changed {
		t.Fatal("default should keep bracketed namespace")
	}
	f := NoUnneededBraces{}.WithConfig(map[string]any{"namespaces": true})
	got, changed := apply(t, f.(fixerRule), src)
	if want := "<?php\nnamespace Foo;\n    echo 1;\n"; !changed || got != want {
		t.Fatalf("namespaces: changed=%v got=%q want=%q", changed, got, want)
	}
}
