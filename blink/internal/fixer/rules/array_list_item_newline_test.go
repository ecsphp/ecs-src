package rules

import "testing"

func TestArrayListItemNewline(t *testing.T) {
	f := ArrayListItemNewline{}

	// associative single-line array -> one item per line, trailing comma, [] on own lines
	got, changed := apply(t, f, "<?php\n$a = ['x' => 1, 'y' => 2];")
	if want := "<?php\n$a = [\n    'x' => 1,\n    'y' => 2,\n];"; !changed || got != want {
		t.Fatalf("assoc: changed=%v got=%q", changed, got)
	}

	// single-element associative array is still split
	got, changed = apply(t, f, "<?php\n$a = ['x' => 1];")
	if want := "<?php\n$a = [\n    'x' => 1,\n];"; !changed || got != want {
		t.Fatalf("single: changed=%v got=%q", changed, got)
	}

	// a plain list array (no "=>") stays inline
	if _, changed := apply(t, f, "<?php\n$a = [1, 2, 3];"); changed {
		t.Fatal("list array must stay inline")
	}

	// an already-multiline array is left to array_indentation
	if _, changed := apply(t, f, "<?php\n$a = [\n    'x' => 1,\n];"); changed {
		t.Fatal("already-multiline array must be a no-op")
	}
}
