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
