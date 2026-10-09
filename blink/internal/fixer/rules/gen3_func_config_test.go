package rules

import "testing"

func TestFunctionDeclarationClosureFunctionSpacing(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	// closure_fn_spacing=none: glue "fn" to "("
	cfg := FunctionDeclaration{}.WithConfig(map[string]any{"closure_fn_spacing": "none"})
	got, changed := apply(t, cfg, "<?php $f = fn () => 1;")
	if want := "<?php $f = fn() => 1;"; !changed || got != want {
		t.Fatalf("closure_fn_spacing=none: changed=%v got=%q want=%q", changed, got, want)
	}

	// a by-reference arrow function ("fn &() => ...") also glues the space after fn
	got, changed = apply(t, cfg, "<?php $f = fn &() => $x;")
	if want := "<?php $f = fn&() => $x;"; !changed || got != want {
		t.Fatalf("by-ref fn: changed=%v got=%q want=%q", changed, got, want)
	}

	// closure_fn_spacing=one keeps a single space for the by-reference form
	one := FunctionDeclaration{}.WithConfig(map[string]any{"closure_fn_spacing": "one"})
	if _, changed := apply(t, one, "<?php $f = fn &() => $x;"); changed {
		t.Fatal("closure_fn_spacing=one must keep the single space in fn &()")
	}
}

func TestFunctionDeclarationTrailingCommaSingleLine(t *testing.T) {
	t.Parallel()
	// trailing_comma_single_line=false: drop the trailing comma in a single-line signature
	cfg := FunctionDeclaration{}.WithConfig(map[string]any{"trailing_comma_single_line": false})
	got, changed := apply(t, cfg, "<?php function f($a, $b,) {}")
	if want := "<?php function f($a, $b) {}"; !changed || got != want {
		t.Fatalf("trailing_comma_single_line=false: changed=%v got=%q want=%q", changed, got, want)
	}
}
