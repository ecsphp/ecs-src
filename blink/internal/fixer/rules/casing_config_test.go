package rules

import "testing"

func TestConstantCaseConfig(t *testing.T) {
	got, changed := apply(t, ConstantCase{}, "<?php $a = TRUE; $b = Null;")
	if want := "<?php $a = true; $b = null;"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}
	f := ConstantCase{}.WithConfig(map[string]any{"case": "upper"})
	got, changed = apply(t, f.(fixerRule), "<?php $a = true; $b = Null;")
	if want := "<?php $a = TRUE; $b = NULL;"; !changed || got != want {
		t.Fatalf("upper: changed=%v got=%q want=%q", changed, got, want)
	}
}
