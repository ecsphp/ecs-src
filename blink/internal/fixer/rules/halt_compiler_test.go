package rules

import "testing"

// The data after "__halt_compiler();" is one inline-data token that no rule
// rewrites.
func TestHaltCompilerDataIsLeftAlone(t *testing.T) {
	t.Parallel()
	src := "<?php\n$a = 1;\n__halt_compiler();\nSTRLEN('x'); if ($a) { echo 1 ; }\n"
	for _, rule := range []fixerRule{NativeFunctionCasing{}, SpaceAfterSemicolon{}, NoMultipleStatementsPerLine{}, NoSinglelineWhitespaceBeforeSemicolons{}, BlankLineAfterOpeningTag{}} {
		if got, _ := apply(t, rule, src); got != src {
			t.Errorf("%T changed the data:\n got: %q", rule, got)
		}
	}
	got, _ := apply(t, SpaceAfterSemicolon{}, "<?php\n$a = 1;$b = 2;\n__halt_compiler();data")
	if want := "<?php\n$a = 1; $b = 2;\n__halt_compiler();data"; got != want {
		t.Errorf("code before the data:\n got: %q\nwant: %q", got, want)
	}
}
