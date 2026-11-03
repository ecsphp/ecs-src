package rules

import "testing"

func TestNewWithParentheses(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NewWithParentheses{}, "<?php $a = new Foo; throw new App\\Bar; $c = new self;")
	if want := "<?php $a = new Foo(); throw new App\\Bar(); $c = new self();"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, NewWithParentheses{}, "<?php $a = new Foo(1);"); changed {
		t.Fatal("already-parenthesized new must not change")
	}
	// anonymous class gains parentheses too (ECS psr12 anonymous_class => true)
	if got, changed := apply(t, NewWithParentheses{}, "<?php $a = new class {};"); !changed || got != "<?php $a = new class() {};" {
		t.Fatalf("anonymous class: changed=%v got=%q", changed, got)
	}
	// dynamic class references gain parentheses too (matches ECS)
	got, changed = apply(t, NewWithParentheses{}, "<?php $a = new $type; $b = new $this->job;")
	if want := "<?php $a = new $type(); $b = new $this->job();"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// an existing call on a dynamic reference is left intact
	if _, changed := apply(t, NewWithParentheses{}, "<?php $a = new $this->make();"); changed {
		t.Fatal("new $this->make() already has parentheses")
	}
}

func TestSingleLineEmptyBody(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, SingleLineEmptyBody{}, "<?php class A\n{\n}")
	if want := "<?php class A {}"; !changed || got != want {
		t.Fatalf("class: changed=%v got=%q want=%q", changed, got, want)
	}
	got, changed = apply(t, SingleLineEmptyBody{}, "<?php function f()\n{\n}")
	if want := "<?php function f() {}"; !changed || got != want {
		t.Fatalf("fn: changed=%v got=%q want=%q", changed, got, want)
	}
	// non-empty body untouched
	if _, changed := apply(t, SingleLineEmptyBody{}, "<?php function f()\n{\n    return 1;\n}"); changed {
		t.Fatal("non-empty body must not collapse")
	}
	// control structure is not a class/function body
	if _, changed := apply(t, SingleLineEmptyBody{}, "<?php if ($a)\n{\n}"); changed {
		t.Fatal("control body is out of scope")
	}
}

// TestSingleLineEmptyBodyMatchesPHPCSFixer covers bodies already "{}" on the
// next line, closures, enums, anonymous classes and a comment before the
// brace; the expected output comes from PHP-CS-Fixer.
func TestSingleLineEmptyBodyMatchesPHPCSFixer(t *testing.T) {
	t.Parallel()
	in := "<?php\nclass A\n{}\nclass B\n{\n}\nfunction f()\n{\n    \n}\n$f = function () use ($x)\n{};\nabstract class C\n{\n    abstract function g();\n    function h() /* c */\n    {\n    }\n}\nuse function Foo\\bar;\ninterface I extends J\n{\n}\nenum E: string\n{\n}\n$o = new class\n{\n};\n"
	want := "<?php\nclass A {}\nclass B {}\nfunction f() {}\n$f = function () use ($x) {};\nabstract class C\n{\n    abstract function g();\n    function h() /* c */\n    {}\n}\nuse function Foo\\bar;\ninterface I extends J {}\nenum E: string {}\n$o = new class {};\n"
	if got, _ := apply(t, SingleLineEmptyBody{}, in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
