package rules

import "testing"

func TestNoUnneededControlParenthesesConfig(t *testing.T) {
	t.Parallel()
	src := "<?php function f() { echo ($a); return ($b); }"
	got, _ := apply(t, NoUnneededControlParentheses{}, src)
	if want := "<?php function f() { echo $a; return $b; }"; got != want {
		t.Fatalf("default got=%q want=%q", got, want)
	}
	f := NoUnneededControlParentheses{}.WithConfig(map[string]any{"statements": []any{"return"}})
	got, _ = apply(t, f.(fixerRule), src)
	if want := "<?php function f() { echo ($a); return $b; }"; got != want {
		t.Fatalf("configured got=%q want=%q", got, want)
	}
}
