package rules

import "testing"

func TestNewWithParentheses(t *testing.T) {
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
