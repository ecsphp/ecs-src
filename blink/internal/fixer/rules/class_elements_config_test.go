package rules

import "testing"

func TestSingleClassElementPerStatementElements(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass C\n{\n    const A = 1, B = 2;\n    public $a, $b;\n}\n"

	// default: both const and property multi-declarations are split
	got, changed := apply(t, SingleClassElementPerStatement{}, src)
	want := "<?php\nclass C\n{\n    const A = 1;\n    const B = 2;\n    public $a;\n    public $b;\n}\n"
	if !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// elements=[property]: only properties are split, the const stays combined
	cfg := SingleClassElementPerStatement{}.WithConfig(map[string]any{"elements": []any{"property"}})
	got, changed = apply(t, cfg, src)
	want = "<?php\nclass C\n{\n    const A = 1, B = 2;\n    public $a;\n    public $b;\n}\n"
	if !changed || got != want {
		t.Fatalf("elements=[property]: changed=%v got=%q want=%q", changed, got, want)
	}
}
