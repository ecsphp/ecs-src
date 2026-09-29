package rules

import "testing"

func TestOperatorLinebreak(t *testing.T) {
	f := OperatorLinebreak{}

	got, changed := apply(t, f, "<?php\n$a = $b ||\n    $c;\n")
	if want := "<?php\n$a = $b\n    || $c;\n"; !changed || got != want {
		t.Fatalf("|| move: changed=%v got=%q", changed, got)
	}

	got, changed = apply(t, f, "<?php\n$s = $a .\n    $b;\n")
	if want := "<?php\n$s = $a\n    . $b;\n"; !changed || got != want {
		t.Fatalf(". move: changed=%v got=%q", changed, got)
	}

	if _, changed := apply(t, f, "<?php\n$a = $b\n    || $c;\n"); changed {
		t.Fatal("already-correct must be a no-op")
	}
	if _, changed := apply(t, f, "<?php\n$a = $b || $c;\n"); changed {
		t.Fatal("single-line must be a no-op")
	}

	// a multiline ternary moves both "?" and ":" to the start of the line
	got, changed = apply(t, f, "<?php\n$a = $b ?\n    $c :\n    $d;\n")
	if want := "<?php\n$a = $b\n    ? $c\n    : $d;\n"; !changed || got != want {
		t.Fatalf("ternary move: changed=%v got=%q", changed, got)
	}

	// a return-type colon is left alone
	if _, changed := apply(t, f, "<?php\nfunction f() :\n    int\n{\n}\n"); changed {
		t.Fatal("return-type colon must be left alone")
	}
	// a nullable type marker is left alone
	if _, changed := apply(t, f, "<?php\nfunction f(\n    ?int $a\n) {}\n"); changed {
		t.Fatal("nullable type must be left alone")
	}
	// a union type separator is left alone
	if _, changed := apply(t, f, "<?php\nfunction f(\n    A|B $a\n) {}\n"); changed {
		t.Fatal("union type must be left alone")
	}
	// a switch case colon is left alone
	if _, changed := apply(t, f, "<?php\nswitch ($x) {\n    case 1:\n        break;\n}\n"); changed {
		t.Fatal("switch case colon must be left alone")
	}
	// a multiline bitwise operator moves
	got, changed = apply(t, f, "<?php\n$a = $x |\n    $y;\n")
	if want := "<?php\n$a = $x\n    | $y;\n"; !changed || got != want {
		t.Fatalf("bitwise or move: changed=%v got=%q", changed, got)
	}
}
