package rules

import "testing"

func TestNoBlankLineBetweenImports(t *testing.T) {
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
}
