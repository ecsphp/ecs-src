package rules

import "testing"

func TestFunctionDeclarationClosureFunctionSpacing(t *testing.T) {
	// default: one space after "function" for a closure
	got, changed := apply(t, FunctionDeclaration{}, "<?php $f = function() {};")
	if want := "<?php $f = function () {};"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// closure_function_spacing=none: no space after "function"
	cfg := FunctionDeclaration{}.WithConfig(map[string]any{"closure_function_spacing": "none"})
	got, changed = apply(t, cfg, "<?php $f = function () {};")
	if want := "<?php $f = function() {};"; !changed || got != want {
		t.Fatalf("closure_function_spacing=none: changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestFunctionDeclarationClosureFnSpacing(t *testing.T) {
	// closure_fn_spacing=none: glue "fn" to "("
	cfg := FunctionDeclaration{}.WithConfig(map[string]any{"closure_fn_spacing": "none"})
	got, changed := apply(t, cfg, "<?php $f = fn () => 1;")
	if want := "<?php $f = fn() => 1;"; !changed || got != want {
		t.Fatalf("closure_fn_spacing=none: changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestFunctionDeclarationTrailingCommaSingleLine(t *testing.T) {
	// trailing_comma_single_line=false: drop the trailing comma in a single-line signature
	cfg := FunctionDeclaration{}.WithConfig(map[string]any{"trailing_comma_single_line": false})
	got, changed := apply(t, cfg, "<?php function f($a, $b,) {}")
	if want := "<?php function f($a, $b) {}"; !changed || got != want {
		t.Fatalf("trailing_comma_single_line=false: changed=%v got=%q want=%q", changed, got, want)
	}
}
