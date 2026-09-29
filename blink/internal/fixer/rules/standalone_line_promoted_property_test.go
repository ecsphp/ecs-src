package rules

import "testing"

func TestStandaloneLinePromotedProperty(t *testing.T) {
	f := StandaloneLinePromotedProperty{}

	// a promoted property: each parameter on its own line
	got, changed := apply(t, f, "<?php\nclass A {\n    public function __construct(protected int $x)\n    {\n    }\n}")
	if want := "<?php\nclass A {\n    public function __construct(\n        protected int $x\n    )\n    {\n    }\n}"; !changed || got != want {
		t.Fatalf("promoted: changed=%v got=%q", changed, got)
	}

	// a plain constructor is left inline
	if _, changed := apply(t, f, "<?php\nclass A {\n    public function __construct($x, $y)\n    {\n    }\n}"); changed {
		t.Fatal("plain constructor must stay inline")
	}

	// a non-constructor method is untouched
	if _, changed := apply(t, f, "<?php\nclass A {\n    public function make(protected int $x)\n    {\n    }\n}"); changed {
		t.Fatal("only __construct is targeted")
	}
}
