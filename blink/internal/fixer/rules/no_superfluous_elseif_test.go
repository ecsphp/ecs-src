package rules

import "testing"

func TestNoSuperfluousElseif(t *testing.T) {
	f := NoSuperfluousElseif{}

	// preceding branch returns -> elseif becomes if
	got, changed := apply(t, f, "<?php\nif ($a) {\n    return 1;\n} elseif ($b) {\n    return 2;\n}\n")
	want := "<?php\nif ($a) {\n    return 1;\n}\nif ($b) {\n    return 2;\n}\n"
	if !changed || got != want {
		t.Fatalf("return: changed=%v got=%q", changed, got)
	}

	// preceding branch does not exit -> unchanged
	if _, changed := apply(t, f, "<?php\nif ($a) {\n    $x = 1;\n} elseif ($b) {\n    return 2;\n}\n"); changed {
		t.Fatal("non-exiting branch must keep elseif")
	}

	// throw counts as exit
	got, changed = apply(t, f, "<?php\nif ($a) {\n    throw new E();\n} elseif ($b) {\n    return 2;\n}\n")
	want = "<?php\nif ($a) {\n    throw new E();\n}\nif ($b) {\n    return 2;\n}\n"
	if !changed || got != want {
		t.Fatalf("throw: changed=%v got=%q", changed, got)
	}

	// "else if" (two tokens) collapses to "if"
	got, changed = apply(t, f, "<?php\nif ($a) {\n    return 1;\n} else if ($b) {\n    return 2;\n}\n")
	want = "<?php\nif ($a) {\n    return 1;\n}\nif ($b) {\n    return 2;\n}\n"
	if !changed || got != want {
		t.Fatalf("else if: changed=%v got=%q", changed, got)
	}
}
