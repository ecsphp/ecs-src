package rules

import "testing"

func TestArrayIndentation(t *testing.T) {
	f := ArrayIndentation{}

	// nested array under-indented: each level indented exactly once
	got, changed := apply(t, f, "<?php\n$foo = [\n   'bar' => [\n    'baz' => true,\n  ],\n];\n")
	if want := "<?php\n$foo = [\n    'bar' => [\n        'baz' => true,\n    ],\n];\n"; !changed || got != want {
		t.Fatalf("nested: changed=%v got=%q", changed, got)
	}

	// grouping is preserved (multiple items per line only get re-indented)
	got, changed = apply(t, f, "<?php\n$a = [\n  1, 2,\n  3, 4,\n];\n")
	if want := "<?php\n$a = [\n    1, 2,\n    3, 4,\n];\n"; !changed || got != want {
		t.Fatalf("grouping: changed=%v got=%q", changed, got)
	}

	// single-line array untouched; array access untouched
	if _, changed := apply(t, f, "<?php\n$a = [1, 2, 3];\n$b = $arr[0];\n"); changed {
		t.Fatal("single-line array / access must not change")
	}

	// already-correct multiline array is a no-op
	if _, changed := apply(t, f, "<?php\n$a = [\n    'x' => 1,\n];\n"); changed {
		t.Fatal("correctly indented array must be a no-op")
	}
}
