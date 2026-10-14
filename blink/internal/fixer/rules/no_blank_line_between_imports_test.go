package rules

import "testing"

func TestNoBlankLineBetweenImports(t *testing.T) {
	t.Parallel()
	f := NoBlankLineBetweenImports{}

	// blank lines between consecutive imports collapse; the namespace gap stays
	got, changed := apply(t, f, "<?php\nnamespace X;\n\nuse A\\B;\n\nuse C\\D;\n\n\nuse E\\F;\nclass Y {}\n")
	if want := "<?php\nnamespace X;\n\nuse A\\B;\nuse C\\D;\nuse E\\F;\nclass Y {}\n"; !changed || got != want {
		t.Fatalf("collapse: changed=%v got=%q", changed, got)
	}

	// already tight: no-op
	if _, changed := apply(t, f, "<?php\nnamespace X;\n\nuse A\\B;\nuse C\\D;\n"); changed {
		t.Fatal("tight imports must be a no-op")
	}

	// closure use is not an import
	if _, changed := apply(t, f, "<?php\n$f = function () use ($a) {};\n\n$g = function () use ($b) {};\n"); changed {
		t.Fatal("closure use must be ignored")
	}

	// blank line between different import groups stays; within a group it collapses
	got, changed = apply(t, f, "<?php\nnamespace X;\n\nuse A\\B;\n\nuse C\\D;\n\nuse function A\\first;\n\nuse function A\\second;\n\nuse const A\\FIRST;\n\nuse E\\F;\nclass Y {}\n")
	if want := "<?php\nnamespace X;\n\nuse A\\B;\nuse C\\D;\n\nuse function A\\first;\nuse function A\\second;\n\nuse const A\\FIRST;\n\nuse E\\F;\nclass Y {}\n"; !changed || got != want {
		t.Fatalf("groups: changed=%v got=%q", changed, got)
	}

	// tight class group followed by a function import: no-op, the group gap stays
	if _, changed := apply(t, f, "<?php\nnamespace X;\n\nuse A\\B;\nuse C\\D;\n\nuse function A\\first;\n"); changed {
		t.Fatal("distinct import groups must be a no-op")
	}
}
