package rules

import "testing"

func TestYodaStyleClassConstant(t *testing.T) {
	// non-yoda moves the variable to the left across a class constant
	got, changed := apply(t, YodaStyle{}, "<?php if (Request::METHOD_POST !== $m) {}")
	if want := "<?php if ($m !== Request::METHOD_POST) {}"; !changed || got != want {
		t.Fatalf("class const: changed=%v got=%q", changed, got)
	}
	// namespaced class constant
	got, changed = apply(t, YodaStyle{}, "<?php if (\\App\\Foo::BAR === $x) {}")
	if want := "<?php if ($x === \\App\\Foo::BAR) {}"; !changed || got != want {
		t.Fatalf("ns class const: changed=%v got=%q", changed, got)
	}
}

func TestYodaStyleNonYodaCompound(t *testing.T) {
	cases := []struct{ src, want string }{
		// compound arithmetic / cast right operands are moved to the left
		{"<?php if (0 === $i % 2) {}", "<?php if ($i % 2 === 0) {}"},
		{"<?php if (1 === (int) $x) {}", "<?php if ((int) $x === 1) {}"},
		// a function call right operand is moved
		{"<?php if (NONE === json_last_error()) {}", "<?php if (json_last_error() === NONE) {}"},
		// a "(" wrapped operand (assignment/coalesce) stays put
		{"<?php if (false !== ($h = fopen($f, 'r'))) {}", "<?php if (false !== ($h = fopen($f, 'r'))) {}"},
		{"<?php if ('' !== ($a['k'] ?? '')) {}", "<?php if ('' !== ($a['k'] ?? '')) {}"},
		// a dynamic method call stays put
		{"<?php if (false === $this->{$m}($x)) {}", "<?php if (false === $this->{$m}($x)) {}"},
		// a dynamic property access (no call) is a variable and moves
		{"<?php if (1 === $o->{$p}) {}", "<?php if ($o->{$p} === 1) {}"},
		// a trailing comment after the operand does not block the swap
		{"<?php if (1069 === $e->getCode() /* x */) {}", "<?php if ($e->getCode() === 1069 /* x */) {}"},
	}
	for _, c := range cases {
		if got, _ := apply(t, YodaStyle{}, c.src); got != c.want {
			t.Errorf("src=%q\n got=%q\nwant=%q", c.src, got, c.want)
		}
	}
}

func TestYodaStyleConfig(t *testing.T) {
	// default: non-yoda, the variable moves to the left
	got, changed := apply(t, YodaStyle{}, "<?php if (null === $x) {}")
	if want := "<?php if ($x === null) {}"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q", changed, got)
	}

	// identical=null leaves the comparison alone
	leave := YodaStyle{}.WithConfig(map[string]any{"identical": nil}).(fixerRule)
	if got, changed := apply(t, leave, "<?php if (null === $x) {}"); changed || got != "<?php if (null === $x) {}" {
		t.Fatalf("null leaves alone: changed=%v got=%q", changed, got)
	}

	// identical=true yoda-ifies, the constant moves to the left
	yoda := YodaStyle{}.WithConfig(map[string]any{"identical": true}).(fixerRule)
	got, changed = apply(t, yoda, "<?php if ($x === null) {}")
	if want := "<?php if (null === $x) {}"; !changed || got != want {
		t.Fatalf("yoda: changed=%v got=%q", changed, got)
	}
}
