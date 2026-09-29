package rules

import "testing"

func TestNoUselessNullsafeOperator(t *testing.T) {
	f := NoUselessNullsafeOperator{}

	got, changed := apply(t, f, "<?php\nclass C extends B {\n    function t() { echo $this?->parentMethod(); }\n}\n")
	if want := "<?php\nclass C extends B {\n    function t() { echo $this->parentMethod(); }\n}\n"; !changed || got != want {
		t.Fatalf("this: changed=%v got=%q", changed, got)
	}

	// nullsafe on a non-$this variable is left alone
	if _, changed := apply(t, f, "<?php\n$a?->b();\n"); changed {
		t.Fatal("only $this?-> is rewritten")
	}

	// already "->" is a no-op
	if _, changed := apply(t, f, "<?php\n$this->b();\n"); changed {
		t.Fatal("plain -> must not change")
	}
}
