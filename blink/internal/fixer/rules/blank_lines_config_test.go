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
