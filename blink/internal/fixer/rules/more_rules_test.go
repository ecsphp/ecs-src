package rules

import "testing"

func TestNoTrailingWhitespaceInComment(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoTrailingWhitespaceInComment{}, "<?php // foo   \n/* bar   \n   baz */\n")
	want := "<?php // foo\n/* bar\n   baz */\n"
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoExtraBlankLines(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoExtraBlankLines{}, "<?php\n$a = 1;\n\n\n\n$b = 2;\n")
	want := "<?php\n$a = 1;\n\n$b = 2;\n"
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoMultipleStatementsPerLine(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	got, changed := apply(t, SingleClassElementPerStatement{}, "<?php class A {\n    public int $a, $b;\n    const X = 1, Y = 2;\n}")
	want := "<?php class A {\n    public int $a;\n    public int $b;\n    const X = 1;\n    const Y = 2;\n}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestOrderedImports(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	got, changed := apply(t, BlankLineBetweenImportGroups{}, "<?php\nuse A\\X;\nuse function F\\g;\nuse const C\\Z;\n")
	want := "<?php\nuse A\\X;\n\nuse function F\\g;\n\nuse const C\\Z;\n"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

// PHP-CS-Fixer's SwitchCaseSpace works switch by switch and gives up on a
// switch whose "default" is not followed by whitespace and ":" (so a tidy
// "default:" leaves "case 1 :" alone). Each expected output is the PHP
// engine's.
func TestSwitchCaseSpaceMatchesPhpCsFixer(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		// a tidy default: the cases are not visited
		"<?php\nswitch ($a) {\n    case 1 :\n        break;\n    default:\n        break;\n}\n": "",
		// a commented default: same, but a later switch is still fixed
		"<?php\nswitch ($a) {\n    case 1 :\n        break;\n    default /* c */ :\n        break;\n}\nswitch ($a) {\n    case 1 :\n        break;\n}\n": "<?php\nswitch ($a) {\n    case 1 :\n        break;\n    default /* c */ :\n        break;\n}\nswitch ($a) {\n    case 1:\n        break;\n}\n",
		// newlines go, a comment keeps its space, named-argument and ternary
		// colons are not the label's, nested switch and match have their own
		// default
		"<?php\nswitch ($a) {\n    case 1\n    :\n        break;\n    case 2 /* c */ :\n        break;\n    case foo(x: 1) :\n        break;\n    case $b ? 1 : 2 :\n        break;\n    case 3 ;\n        break;\n    default\n    :\n        switch ($b) {\n            case 1 :\n                break;\n            default:\n                $c = match ($d) { default => 1 };\n        }\n}\n": "<?php\nswitch ($a) {\n    case 1:\n        break;\n    case 2 /* c */ :\n        break;\n    case foo(x: 1):\n        break;\n    case $b ? 1 : 2:\n        break;\n    case 3;\n        break;\n    default:\n        switch ($b) {\n            case 1 :\n                break;\n            default:\n                $c = match ($d) { default => 1 };\n        }\n}\n",
		// alternative syntax
		"<?php\nswitch ($a):\n    case 1 :\n        break;\n    default :\n        break;\nendswitch;\n": "<?php\nswitch ($a):\n    case 1:\n        break;\n    default:\n        break;\nendswitch;\n",
	}
	for src, want := range cases {
		got, changed := apply(t, SwitchCaseSpace{}, src)
		if want == "" {
			want = src
		}
		if changed != (want != src) || got != want {
			t.Fatalf("%q: changed=%v\n got=%q\nwant=%q", src, changed, got, want)
		}
	}
}
