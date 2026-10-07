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

	// a cast before "[" opens an array literal, so it still splits
	got, changed = apply(t, f, "<?php\n$a = (object) ['x' => 1, 'y' => 2];")
	if want := "<?php\n$a = (object) [\n    'x' => 1,\n    'y' => 2,\n];"; !changed || got != want {
		t.Fatalf("cast: changed=%v got=%q", changed, got)
	}

	// an index access on a call result is not an array literal, so it is left alone
	if _, changed := apply(t, f, "<?php\n$a = foo()['x'];"); changed {
		t.Fatal("call-result index access must stay inline")
	}

	// a multiline array whose top-level comma still shares a line gets split
	got, changed = apply(t, f, "<?php\n$a = [\n    'x' => 1,\n    'y' => [\n        'z' => 2,\n    ], 'w' => 3,\n];")
	if want := "<?php\n$a = [\n    'x' => 1,\n    'y' => [\n        'z' => 2,\n    ],\n'w' => 3,\n];"; !changed || got != want {
		t.Fatalf("multiline split: changed=%v got=%q", changed, got)
	}

	// a list array whose last element is a single-line associative array: expanding
	// the inner array also breaks the outer list's opener (not its closer)
	got, changed = apply(t, f, "<?php\nreturn [$a, $b, ['k' => 1, 'm' => 2]];")
	if want := "<?php\nreturn [\n$a, $b, [\n    'k' => 1,\n    'm' => 2,\n]];"; !changed || got != want {
		t.Fatalf("outer opener break: changed=%v got=%q", changed, got)
	}

	// but when that inner associative array is already multiline, the outer list
	// opener is left alone (no upstream position shift happens)
	if _, changed := apply(t, f, "<?php\nreturn [$a, $b, [\n    'k' => 1,\n    'm' => 2,\n]];"); changed {
		t.Fatal("already-multiline inner must not break the outer opener")
	}

	// a blank line a user put between array items is preserved
	if _, changed := apply(t, f, "<?php\n$x = [\n    'a' => 1,\n\n    'b' => 2,\n];"); changed {
		t.Fatal("blank line between array items must be preserved")
	}

	// the assoc is wrapped in a call that is the list's last element: the list
	// opener still breaks
	got, changed = apply(t, f, "<?php\nreturn foo([bar(['k' => 1])]);")
	if want := "<?php\nreturn foo([\nbar([\n    'k' => 1,\n])]);"; !changed || got != want {
		t.Fatalf("call-wrapped: changed=%v got=%q", changed, got)
	}

	// wrapped in "new" with other array args before the assoc
	got, changed = apply(t, f, "<?php\nreturn foo([new Bar([], ['k' => 1])]);")
	if want := "<?php\nreturn foo([\nnew Bar([], [\n    'k' => 1,\n])]);"; !changed || got != want {
		t.Fatalf("new-wrapped: changed=%v got=%q", changed, got)
	}

	// the enclosing list's first element is itself an array: its opener is left alone
	got, changed = apply(t, f, "<?php\nreturn [['k' => 1]];")
	if want := "<?php\nreturn [[\n    'k' => 1,\n]];"; !changed || got != want {
		t.Fatalf("first-element-array: changed=%v got=%q", changed, got)
	}
}
