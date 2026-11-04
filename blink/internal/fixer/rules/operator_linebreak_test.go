package rules

import "testing"

func TestOperatorLinebreak(t *testing.T) {
	t.Parallel()
	f := OperatorLinebreak{}

	got, changed := apply(t, f, "<?php\n$a = $b ||\n    $c;\n")
	if want := "<?php\n$a = $b\n    || $c;\n"; !changed || got != want {
		t.Fatalf("|| move: changed=%v got=%q", changed, got)
	}

	got, changed = apply(t, f, "<?php\n$s = $a .\n    $b;\n")
	if want := "<?php\n$s = $a\n    . $b;\n"; !changed || got != want {
		t.Fatalf(". move: changed=%v got=%q", changed, got)
	}

	// an object operator "->" at the end of a line moves to the start of the next
	got, changed = apply(t, f, "<?php\n$x = $this->model->\n    call($a);\n")
	if want := "<?php\n$x = $this->model\n    ->call($a);\n"; !changed || got != want {
		t.Fatalf("-> move: changed=%v got=%q", changed, got)
	}

	// "::" at the end of a line moves too
	got, changed = apply(t, f, "<?php\n$x = Foo::\n    BAR;\n")
	if want := "<?php\n$x = Foo\n    ::BAR;\n"; !changed || got != want {
		t.Fatalf(":: move: changed=%v got=%q", changed, got)
	}

	// the PHP 8.5 pipe operator "|>" moves as one operator
	got, changed = apply(t, f, "<?php\n$x = $a |>\n    f(...);\n")
	if want := "<?php\n$x = $a\n    |> f(...);\n"; !changed || got != want {
		t.Fatalf("|> move: changed=%v got=%q", changed, got)
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

// The context classifiers OperatorLinebreak shares with TernaryOperatorSpaces
// leave a token alone when PHP-CS-Fixer's tokenizer gives it a non-operator
// role. Expected outputs come from the PHP engine.
func TestOperatorLinebreakTokenContexts(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, in, want string }{
		{
			"reserved-word named arguments",
			"<?php\nf(default:\n    $x, new:\n    $y);\n",
			"<?php\nf(default:\n    $x, new:\n    $y);\n",
		},
		{
			"ternary in a case expression",
			"<?php\nswitch ($a) {\n    case $b ?\n        1 :\n        2:\n        break;\n}\n",
			"<?php\nswitch ($a) {\n    case $b\n        ? 1\n        : 2:\n        break;\n}\n",
		},
		{
			"nullable after attribute and asymmetric visibility",
			"<?php\nclass A { public function __construct(public private(set) ?\n    int $a, #[X] ?\n    int $b) {} }\nfunction f(#[X] ?\n    int $x) {}\n",
			"<?php\nclass A { public function __construct(public private(set) ?\n    int $a, #[X] ?\n    int $b) {} }\nfunction f(#[X] ?\n    int $x) {}\n",
		},
		{
			"brace switch nested in alternative switch",
			"<?php\nswitch ($a):\n    case 1:\n        switch ($b) { case 2: break; }\n        break;\n    case 3:\n        break;\nendswitch;\n",
			"<?php\nswitch ($a):\n    case 1:\n        switch ($b) { case 2: break; }\n        break;\n    case 3:\n        break;\nendswitch;\n",
		},
		{
			"reserved-word function name before return type",
			"<?php\nclass A { function &list():\n    int { return 1; } public function &print()\n    : int {} }\n",
			"<?php\nclass A { function &list():\n    int { return 1; } public function &print()\n    : int {} }\n",
		},
	}
	for _, c := range cases {
		got, _ := apply(t, OperatorLinebreak{}, c.in)
		if got != c.want {
			t.Errorf("%s\n  in: %q\n got: %q\nwant: %q", c.name, c.in, got, c.want)
		}
	}
}
