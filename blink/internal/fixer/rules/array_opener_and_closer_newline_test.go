package rules

import "testing"

func TestArrayOpenerAndCloserNewline(t *testing.T) {
	f := ArrayOpenerAndCloserNewline{}

	// associative array: "[" and "]" go on their own lines, items untouched
	got, changed := apply(t, f, "<?php\n$a = [\"x\" => 1, \"y\" => 2];\n")
	if want := "<?php\n$a = [\n\"x\" => 1, \"y\" => 2\n];\n"; !changed || got != want {
		t.Fatalf("assoc: changed=%v got=%q", changed, got)
	}

	// plain list (no "=>") is left inline
	if _, changed := apply(t, f, "<?php\n$m = [\"k\" => 1];\n$a = [1, 2, 3];\n"); changed {
		got, _ := apply(t, f, "<?php\n$a = [1, 2, 3];\n")
		if got != "<?php\n$a = [1, 2, 3];\n" {
			t.Fatalf("plain list must stay inline, got %q", got)
		}
	}

	// first element is itself an array: skip
	if _, changed := apply(t, f, "<?php\n$a = [[1, 2], \"k\" => 3];\n"); changed {
		t.Fatal("array whose first element is an array must be skipped")
	}

	// already on own lines: no-op
	if _, changed := apply(t, f, "<?php\n$a = [\n    \"x\" => 1,\n];\n"); changed {
		t.Fatal("already-broken array must be a no-op")
	}
}
