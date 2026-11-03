package rules

import "testing"

func TestBracesPositionConfig(t *testing.T) {
	t.Parallel()
	// default: a class body brace moves to the next line
	got, changed := apply(t, BracesPosition{}, "<?php\nclass A {\n    public $x;\n}\n")
	if want := "<?php\nclass A\n{\n    public $x;\n}\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q", changed, got)
	}

	// classes_opening_brace=same_line keeps the brace on the class line
	same := BracesPosition{}.WithConfig(map[string]any{"classes_opening_brace": "same_line"}).(fixerRule)
	got, changed = apply(t, same, "<?php\nclass A\n{\n    public $x;\n}\n")
	if want := "<?php\nclass A {\n    public $x;\n}\n"; !changed || got != want {
		t.Fatalf("classes same_line: changed=%v got=%q", changed, got)
	}

	// control_structures_opening_brace=next_line moves the control brace down
	ctrl := BracesPosition{}.WithConfig(map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"}).(fixerRule)
	got, changed = apply(t, ctrl, "<?php\nif ($a) {\n    echo 1;\n}\n")
	if want := "<?php\nif ($a)\n{\n    echo 1;\n}\n"; !changed || got != want {
		t.Fatalf("control next_line: changed=%v got=%q", changed, got)
	}

	// allow_single_line_anonymous_functions=false expands a single-line closure
	// body, at the "{" line's indentation (StatementIndentation indents it)
	noSingle := BracesPosition{}.WithConfig(map[string]any{"allow_single_line_anonymous_functions": false}).(fixerRule)
	got, changed = apply(t, noSingle, "<?php\n$f = function ($x) { return $x; };\n")
	if want := "<?php\n$f = function ($x) {\nreturn $x;\n};\n"; !changed || got != want {
		t.Fatalf("closure expand: changed=%v got=%q", changed, got)
	}
	// an empty body is expanded too
	got, changed = apply(t, noSingle, "<?php\n$f = function ($x) {};\n")
	if want := "<?php\n$f = function ($x) {\n};\n"; !changed || got != want {
		t.Fatalf("empty closure: changed=%v got=%q", changed, got)
	}
	// a comment on the brace line keeps the line break after it as is
	if _, changed := apply(t, noSingle, "<?php\n$f = function ($x) { // note\n    return $x;\n};\n"); changed {
		t.Fatal("comment on brace line must be left alone")
	}
	// default (option unset) keeps a single-line closure body
	if _, changed := apply(t, BracesPosition{}, "<?php\n$f = function ($x) { return $x; };\n"); changed {
		t.Fatal("default must keep single-line closure body")
	}
}

// TestBracesPositionSignatureEnd pins the next_line_unless_newline_at_signature_end
// exception against the PHP fixer: the signature ends at the ")" found by
// stepping back from "{" over return-type tokens (DNF groups included) and an
// anonymous class's extends/implements target, and a ")" elsewhere (a call
// chain before "new class") is not the signature.
func TestBracesPositionSignatureEnd(t *testing.T) {
	t.Parallel()
	const nl = "next_line_unless_newline_at_signature_end"
	withOpt := func(opt string) fixerRule {
		return BracesPosition{}.WithConfig(map[string]any{opt: nl}).(fixerRule)
	}
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{
			"default: DNF return types and a comment before the multi-line ')' keep the brace",
			BracesPosition{},
			"<?php\nclass A\n{\n    public function h(\n        int $a,\n    ): (A&B)|null {\n        return null;\n    }\n\n    public function i(\n        int $a,\n    ): A|(B&C) {\n        return null;\n    }\n\n    public function j(\n        int $a,\n    ): ?\\Foo\\Bar {\n        return null;\n    }\n\n    public function k(\n        int $a\n    /* c */): static {\n        return $this;\n    }\n\n    public function l(int $a): array|callable {\n        return [];\n    }\n}\n",
			"<?php\nclass A\n{\n    public function h(\n        int $a,\n    ): (A&B)|null {\n        return null;\n    }\n\n    public function i(\n        int $a,\n    ): A|(B&C) {\n        return null;\n    }\n\n    public function j(\n        int $a,\n    ): ?\\Foo\\Bar {\n        return null;\n    }\n\n    public function k(\n        int $a\n    /* c */): static {\n        return $this;\n    }\n\n    public function l(int $a): array|callable\n    {\n        return [];\n    }\n}\n",
		},
		{
			"default: a multi-line attribute is not the class signature",
			BracesPosition{},
			"<?php\n#[Attr(\n    1\n)]\nfinal class C extends D implements E {\n}\n",
			"<?php\n#[Attr(\n    1\n)]\nfinal class C extends D implements E\n{\n}\n",
		},
		{
			"anonymous classes: extends/implements after the arguments, or a call chain, is not the signature end",
			withOpt("anonymous_classes_opening_brace"),
			"<?php\n$x = new class(\n    $a,\n) extends B {\n    public $p;\n};\n$y = new class(\n    $a,\n) {\n    public $p;\n};\n$z = bar(\n    $a\n)->baz(new class {\n    public $p;\n});\n$v = new class(\n    $a\n) implements \\N\\I {\n    public $p;\n};\n",
			"<?php\n$x = new class(\n    $a,\n) extends B\n{\n    public $p;\n};\n$y = new class(\n    $a,\n) {\n    public $p;\n};\n$z = bar(\n    $a\n)->baz(new class\n{\n    public $p;\n});\n$v = new class(\n    $a\n) implements \\N\\I\n{\n    public $p;\n};\n",
		},
		{
			"anonymous functions: the use list's ')' and a DNF return type",
			withOpt("anonymous_functions_opening_brace"),
			"<?php\n$f = function (\n    $a\n) use ($b) {\n    return 1;\n};\n$g = function ($a) use (\n    $b\n): (A&B)|null {\n    return null;\n};\n$e = function (\n    $a\n): A|(B&C) {\n    return null;\n};\n",
			"<?php\n$f = function (\n    $a\n) use ($b)\n{\n    return 1;\n};\n$g = function ($a) use (\n    $b\n): (A&B)|null {\n    return null;\n};\n$e = function (\n    $a\n): A|(B&C) {\n    return null;\n};\n",
		},
		{
			"control structures: a comment before the multi-line ')'",
			withOpt("control_structures_opening_brace"),
			"<?php\nwhile (\n    $a\n    /* c */) {\n    echo 1;\n}\nif (\n    $a\n) {\n    echo 1;\n} else {\n    echo 2;\n}\n",
			"<?php\nwhile (\n    $a\n    /* c */) {\n    echo 1;\n}\nif (\n    $a\n) {\n    echo 1;\n} else\n{\n    echo 2;\n}\n",
		},
		{
			"classes and functions set explicitly behave as the default",
			withOpt("functions_opening_brace"),
			"<?php\nfunction f(\n    int $a,\n): (A&B)|null {\n    return null;\n}\nfunction g(int $a): (A&B)|null {\n    return null;\n}\n",
			"<?php\nfunction f(\n    int $a,\n): (A&B)|null {\n    return null;\n}\nfunction g(int $a): (A&B)|null\n{\n    return null;\n}\n",
		},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s:\n got: %q\nwant: %q", c.name, got, c.want)
			continue
		}
		idempotent(t, c.rule, got)
	}
}
