package rules

import "testing"

func TestSpaceAfterCommaHereNowDoc(t *testing.T) {
	f := SpaceAfterCommaHereNowDoc{}

	got, changed := apply(t, f, "<?php\nfoo(<<<EOT\nbody\nEOT, $x);\n")
	if want := "<?php\nfoo(<<<EOT\nbody\nEOT\n, $x);\n"; !changed || got != want {
		t.Fatalf("comma: changed=%v got=%q", changed, got)
	}

	got, changed = apply(t, f, "<?php\n$a = [<<<EOT\nx\nEOT];\n")
	if want := "<?php\n$a = [<<<EOT\nx\nEOT\n];\n"; !changed || got != want {
		t.Fatalf("bracket: changed=%v got=%q", changed, got)
	}

	// already on its own line: no-op
	if _, changed := apply(t, f, "<?php\nfoo(<<<EOT\nbody\nEOT\n, $x);\n"); changed {
		t.Fatal("already separated must be a no-op")
	}
}
