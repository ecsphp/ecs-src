package rules

import "testing"

func TestStandaloneLineInMultilineArray(t *testing.T) {
	f := StandaloneLineInMultilineArray{}

	// a single-line associative array is broken to one item per line
	got, changed := apply(t, f, "<?php\n$a = [1 => \"x\", 2 => \"y\"];\n")
	want := "<?php\n$a = [\n    1 => \"x\",\n    2 => \"y\"\n];\n"
	if !changed || got != want {
		t.Fatalf("single-line: changed=%v got=%q", changed, got)
	}

	// a plain (non-associative) list is left alone
	if _, changed := apply(t, f, "<?php\n$a = [\n    1, 2, 3,\n];\n"); changed {
		t.Fatal("non-associative array must be left alone")
	}

	// a lone associative item not introduced by => is left alone
	if _, changed := apply(t, f, "<?php\n$a = [\n    1 => \"x\",\n];\n"); changed {
		t.Fatal("single associative item must be left alone")
	}

	// the first skipped array stops the whole pass (upstream fix() returns)
	if _, changed := apply(t, f, "<?php\n$a = [[\"k\" => 1, \"j\" => 2]];\n"); changed {
		t.Fatal("a leading non-associative array aborts the pass")
	}
}
