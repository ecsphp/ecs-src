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

// Each split constant keeps the type. Expected output from PHP-CS-Fixer.
func TestSingleClassElementPerStatementTypedConst(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass C\n{\n    const string A = 'a', B = 'b';\n    public const ?int C = 1, D = 2;\n    final protected const int|string E = 1, F = [1, 2];\n    const array G = [], H = [];\n    const \\Foo\\Bar I = null, J = null;\n}\n"
	want := "<?php\nclass C\n{\n    const string A = 'a';\n    const string B = 'b';\n    public const ?int C = 1;\n    public const ?int D = 2;\n    final protected const int|string E = 1;\n    final protected const int|string F = [1, 2];\n    const array G = [];\n    const array H = [];\n    const \\Foo\\Bar I = null;\n    const \\Foo\\Bar J = null;\n}\n"
	if got, changed := apply(t, SingleClassElementPerStatement{}, src); !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}
