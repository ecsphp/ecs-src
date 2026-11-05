package rules

import "testing"

func TestStandaloneLineConstructorParam(t *testing.T) {
	t.Parallel()
	f := StandaloneLineConstructorParam{}

	// any count; the indent comes from the indent level, not the line's tabs
	got, changed := apply(t, f, "<?php\nclass A\n{\n\tpublic function __construct($a, $b)\n\t{\n\t}\n}\n")
	if want := "<?php\nclass A\n{\n\tpublic function __construct(\n    $a,\n    $b\n)\n\t{\n\t}\n}\n"; !changed || got != want {
		t.Fatalf("tabs: changed=%v got=%q", changed, got)
	}

	// a newline already after a comma is kept
	got, changed = apply(t, f, "<?php\nclass B\n{\n    public function __construct($a,\n        $b) {}\n}\n")
	if want := "<?php\nclass B\n{\n    public function __construct(\n        $a,\n        $b\n    ) {}\n}\n"; !changed || got != want {
		t.Fatalf("existing newline: changed=%v got=%q", changed, got)
	}

	// no parameters, and the name is case-sensitive
	if got, changed := apply(t, f, "<?php\nclass C\n{\n    public function __construct() {}\n    public function __CONSTRUCT($a) {}\n}\n"); changed {
		t.Fatalf("skip: got=%q", got)
	}
}
