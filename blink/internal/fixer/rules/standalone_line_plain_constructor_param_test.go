package rules

import "testing"

func TestStandaloneLinePlainConstructorParam(t *testing.T) {
	f := StandaloneLinePlainConstructorParam{}

	// 4+ plain params: each on its own line
	got, changed := apply(t, f, "<?php\nclass A\n{\n    public function __construct($a, $b, $c, $d)\n    {\n    }\n}\n")
	if want := "<?php\nclass A\n{\n    public function __construct(\n        $a,\n        $b,\n        $c,\n        $d\n    )\n    {\n    }\n}\n"; !changed || got != want {
		t.Fatalf("four: changed=%v got=%q", changed, got)
	}

	// typed params also break
	got, changed = apply(t, f, "<?php\nclass A\n{\n    public function __construct(int $a, string $b, Foo $c, Bar $d)\n    {\n    }\n}\n")
	if want := "<?php\nclass A\n{\n    public function __construct(\n        int $a,\n        string $b,\n        Foo $c,\n        Bar $d\n    )\n    {\n    }\n}\n"; !changed || got != want {
		t.Fatalf("typed: changed=%v got=%q", changed, got)
	}

	// 3 params stay inline
	if _, changed := apply(t, f, "<?php\nclass A\n{\n    public function __construct($a, $b, $c)\n    {\n    }\n}\n"); changed {
		t.Fatal("fewer than 4 params must stay inline")
	}

	// promoted constructor is left to StandaloneLinePromotedProperty
	if _, changed := apply(t, f, "<?php\nclass A\n{\n    public function __construct($a, $b, $c, private int $d)\n    {\n    }\n}\n"); changed {
		t.Fatal("promoted constructor must be skipped")
	}
}
