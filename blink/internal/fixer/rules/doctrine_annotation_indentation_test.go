package rules

import "testing"

func TestDoctrineAnnotationIndentation(t *testing.T) {
	f := DoctrineAnnotationIndentation{}

	// annotation continuation lines get four-space indentation
	got, changed := apply(t, f, "<?php\n/**\n *  @Foo(\n *   foo=\"foo\"\n *  )\n */\nclass Bar {}\n")
	if want := "<?php\n/**\n * @Foo(\n *     foo=\"foo\"\n * )\n */\nclass Bar {}\n"; !changed || got != want {
		t.Fatalf("basic: changed=%v got=%q", changed, got)
	}

	// a docblock that does not precede a class is skipped
	if _, changed := apply(t, f, "<?php\n/**\n *  @Foo(\n *   foo=\"foo\"\n *  )\n */\nfunction f() {}\n"); changed {
		t.Fatal("non-class docblock must be skipped")
	}
}
