package rules

import "testing"

func TestBlankLinesBeforeNamespaceConfig(t *testing.T) {
	t.Parallel()

	src := "<?php\n\n\nnamespace Foo;"

	// default collapses to exactly one blank line (2 line breaks).
	if got, changed := apply(t, BlankLinesBeforeNamespace{}, src); !changed || got != "<?php\n\nnamespace Foo;" {
		t.Fatalf("default changed=%v got=%q", changed, got)
	}

	// min=max=1 keeps a single line break.
	f := BlankLinesBeforeNamespace{}.WithConfig(map[string]any{"min_line_breaks": float64(1), "max_line_breaks": float64(1)})
	if got, changed := apply(t, f.(fixerRule), src); !changed || got != "<?php\nnamespace Foo;" {
		t.Fatalf("min=max=1 changed=%v got=%q", changed, got)
	}

	// min=3 raises a single break up to three.
	f = BlankLinesBeforeNamespace{}.WithConfig(map[string]any{"min_line_breaks": float64(3), "max_line_breaks": float64(4)})
	if got, changed := apply(t, f.(fixerRule), "<?php\nnamespace Foo;"); !changed || got != "<?php\n\n\nnamespace Foo;" {
		t.Fatalf("min=3 changed=%v got=%q", changed, got)
	}
}
