package rules

import "testing"

func TestOrderedImportsSortAlgorithm(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestOrderedImportsKeepsBlankBetweenGroups(t *testing.T) {
	t.Parallel()
	// imports_order=null (no grouping): already-sorted imports with a blank line
	// separating the class group from the function group are left untouched -
	// the blank line between statements is preserved, not collapsed.
	src := "<?php\nuse Aaa\\Bbb;\nuse Aaa\\Ccc;\n\nuse function Aaa\\ddd;\n"
	cfg := OrderedImports{}.WithConfig(map[string]any{"imports_order": nil})
	if got, changed := apply(t, cfg, src); changed || got != src {
		t.Fatalf("blank between groups must be preserved: changed=%v got=%q", changed, got)
	}
}

func TestOrderedImportsAliasInSortKey(t *testing.T) {
	t.Parallel()
	// the "as <alias>" suffix is part of the alpha key: a plain import sorts
	// before the aliased import of the same class, matching PHP-CS-Fixer
	src := "<?php\nuse Mautic\\LeadBundle\\Entity\\Lead as TheLead;\nuse Mautic\\LeadBundle\\Entity\\Lead;\n"
	want := "<?php\nuse Mautic\\LeadBundle\\Entity\\Lead;\nuse Mautic\\LeadBundle\\Entity\\Lead as TheLead;\n"
	if got, changed := apply(t, OrderedImports{}, src); !changed || got != want {
		t.Fatalf("alias key: changed=%v got=%q want=%q", changed, got, want)
	}

	// "\" and the space before "as" both sort before an identifier char, so an
	// aliased class precedes a deeper path under the same prefix
	src = "<?php\nuse Foo\\Bar\\Qux;\nuse Foo\\Bar as Baz;\n"
	want = "<?php\nuse Foo\\Bar as Baz;\nuse Foo\\Bar\\Qux;\n"
	if got, changed := apply(t, OrderedImports{}, src); !changed || got != want {
		t.Fatalf("alias vs deeper path: changed=%v got=%q want=%q", changed, got, want)
	}
}
