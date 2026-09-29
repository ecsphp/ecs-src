package rules

import "testing"

func TestNoTrailingWhitespaceInComment(t *testing.T) {
	got, changed := apply(t, NoTrailingWhitespaceInComment{}, "<?php // foo   \n/* bar   \n   baz */\n")
	want := "<?php // foo\n/* bar\n   baz */\n"
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoExtraBlankLines(t *testing.T) {
	got, changed := apply(t, NoExtraBlankLines{}, "<?php\n$a = 1;\n\n\n\n$b = 2;\n")
	want := "<?php\n$a = 1;\n\n$b = 2;\n"
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoMultipleStatementsPerLine(t *testing.T) {
	got, changed := apply(t, NoMultipleStatementsPerLine{}, "<?php\n$a = 1; $b = 2;$c = 3;\n")
	want := "<?php\n$a = 1;\n$b = 2;\n$c = 3;\n"
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// for-loop header semicolons must not split
	if _, changed := apply(t, NoMultipleStatementsPerLine{}, "<?php for ($i = 0; $i < 3; $i++) {}\n"); changed {
		t.Fatal("for-header semicolons must not split")
	}
}

func TestSwitchCaseSpaceAndSemicolon(t *testing.T) {
	got, changed := apply(t, SwitchCaseSpace{}, "<?php switch ($a) { case 1 : break; default : break; }")
	want := "<?php switch ($a) { case 1: break; default: break; }"
	if !changed || got != want {
		t.Fatalf("space: changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, SwitchCaseSemicolonToColon{}, "<?php switch ($a) { case 1; break; }")
	want = "<?php switch ($a) { case 1: break; }"
	if !changed || got != want {
		t.Fatalf("semicolon: changed=%v got=%q want=%q", changed, got, want)
	}
	// enum case must not be touched
	if _, changed := apply(t, SwitchCaseSemicolonToColon{}, "<?php enum E { case A; case B; }"); changed {
		t.Fatal("enum case must not be converted")
	}
	// ternary colon inside a case label is not the terminator
	got, _ = apply(t, SwitchCaseSpace{}, "<?php switch ($a) { case $x ? 1 : 2: break; }")
	if want := "<?php switch ($a) { case $x ? 1 : 2: break; }"; got != want {
		t.Fatalf("ternary: got=%q want=%q", got, want)
	}
}

func TestSingleClassElementPerStatement(t *testing.T) {
	got, changed := apply(t, SingleClassElementPerStatement{}, "<?php class A {\n    public int $a, $b;\n    const X = 1, Y = 2;\n}")
	want := "<?php class A {\n    public int $a;\n    public int $b;\n    const X = 1;\n    const Y = 2;\n}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestOrderedImports(t *testing.T) {
	got, changed := apply(t, OrderedImports{}, "<?php\nuse const C\\Z;\nuse B\\Y;\nuse function F\\g;\nuse A\\X;\n")
	// ECS imports_order: class group first (alpha), then function, then const
	want := "<?php\nuse A\\X;\nuse B\\Y;\nuse function F\\g;\nuse const C\\Z;\n"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// already grouped -> no change (idempotent)
	if _, changed := apply(t, OrderedImports{}, want); changed {
		t.Fatal("already-grouped imports should not change")
	}
}

func TestBlankLineBetweenImportGroups(t *testing.T) {
	got, changed := apply(t, BlankLineBetweenImportGroups{}, "<?php\nuse A\\X;\nuse function F\\g;\nuse const C\\Z;\n")
	want := "<?php\nuse A\\X;\n\nuse function F\\g;\n\nuse const C\\Z;\n"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}
