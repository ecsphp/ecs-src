package rules

import "testing"

func TestFunctionToConstantConfig(t *testing.T) {
	t.Parallel()
	src := "<?php $a = pi(); $b = phpversion();"
	got, _ := apply(t, FunctionToConstant{}, src)
	if want := "<?php $a = M_PI; $b = PHP_VERSION;"; got != want {
		t.Fatalf("default got=%q want=%q", got, want)
	}
	f := FunctionToConstant{}.WithConfig(map[string]any{"functions": []any{"pi"}})
	got, _ = apply(t, f.(fixerRule), src)
	if want := "<?php $a = M_PI; $b = phpversion();"; got != want {
		t.Fatalf("configured got=%q want=%q", got, want)
	}
}
