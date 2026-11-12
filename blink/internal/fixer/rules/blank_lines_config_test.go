package rules

import "testing"

func TestNoExtraBlankLinesConfig(t *testing.T) {
	t.Parallel()
	// default: collapse runs of blank lines only
	got, changed := apply(t, NoExtraBlankLines{}, "<?php\n$a = 1;\n\n\n\n$b = 2;\n")
	if want := "<?php\n$a = 1;\n\n$b = 2;\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q", changed, got)
	}

	f := NoExtraBlankLines{}.WithConfig(map[string]any{"tokens": []any{"extra", "throw", "use"}}).(fixerRule)

	// throw: blank line after a throw statement is removed
	got, _ = apply(t, f, "<?php\nfunction foo()\n{\n    throw new Exception(\"x\");\n\n}\n")
	if want := "<?php\nfunction foo()\n{\n    throw new Exception(\"x\");\n}\n"; got != want {
		t.Fatalf("throw: got=%q", got)
	}

	// use: blank line between two imports is removed
	got, _ = apply(t, f, "<?php\nuse A\\B;\n\nuse C\\D;\n")
	if want := "<?php\nuse A\\B;\nuse C\\D;\n"; got != want {
		t.Fatalf("use: got=%q", got)
	}
}

func TestNoExtraBlankLinesCurlyBraceBlock(t *testing.T) {
	t.Parallel()
	f := NoExtraBlankLines{}.WithConfig(map[string]any{"tokens": []any{"curly_brace_block"}}).(fixerRule)

	got, changed := apply(t, f, "<?php\nclass A\n{\n\n    public function run()\n    {\n\n        return 1;\n\n    }\n\n}\n")
	if want := "<?php\nclass A\n{\n    public function run()\n    {\n        return 1;\n    }\n}\n"; !changed || got != want {
		t.Fatalf("block: changed=%v got=%q", changed, got)
	}

	// single-line block, dynamic property, group import and interpolation braces are left alone
	for _, in := range []string{
		"<?php\n$f = function () { return 1; };\n$a->{\n\n'b'\n\n};\n",
		"<?php\nuse A\\{\n\n    B,\n\n    C\n};\n",
		"<?php\n$s = \"x {$a\n\n} y\";\n",
	} {
		if got, _ := apply(t, f, in); got != in {
			t.Fatalf("skip: got=%q", got)
		}
	}

	// a multi-line docblock before "}" still counts as the last line break
	got, _ = apply(t, f, "<?php\nif ($x) {\n    /**\n     * d\n     */\n\n}\n")
	if want := "<?php\nif ($x) {\n    /**\n     * d\n     */\n}\n"; got != want {
		t.Fatalf("docblock: got=%q", got)
	}
}
