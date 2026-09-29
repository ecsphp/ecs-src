package rules

import "testing"

func TestOrderedImportsSortAlgorithm(t *testing.T) {
	src := "<?php\nuse B;\nuse A;\n"

	// default: alpha sort within the class group
	got, changed := apply(t, OrderedImports{}, src)
	if want := "<?php\nuse A;\nuse B;\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// sort_algorithm=none: original order kept, nothing reordered
	cfg := OrderedImports{}.WithConfig(map[string]any{"sort_algorithm": "none"})
	if got, changed := apply(t, cfg, src); changed || got != src {
		t.Fatalf("sort_algorithm=none: changed=%v got=%q want unchanged", changed, got)
	}
}

func TestOrderedImportsImportsOrder(t *testing.T) {
	src := "<?php\nuse A;\nuse function a;\n"

	// default imports_order [class, function, const]: class group stays first
	if got, changed := apply(t, OrderedImports{}, src); changed || got != src {
		t.Fatalf("default: changed=%v got=%q want unchanged", changed, got)
	}

	// imports_order=[function, const, class]: the function group moves ahead
	cfg := OrderedImports{}.WithConfig(map[string]any{
		"imports_order": []any{"function", "const", "class"},
	})
	got, changed := apply(t, cfg, src)
	if want := "<?php\nuse function a;\nuse A;\n"; !changed || got != want {
		t.Fatalf("imports_order: changed=%v got=%q want=%q", changed, got, want)
	}
}
