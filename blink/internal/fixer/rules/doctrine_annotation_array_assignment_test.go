package rules

import "testing"

func TestDoctrineAnnotationArrayAssignment(t *testing.T) {
	f := DoctrineAnnotationArrayAssignment{}

	// a colon inside an annotation array becomes an equals
	got, changed := apply(t, f, "<?php\n/**\n * @Foo({bar : \"baz\"})\n */\nclass Bar {}\n")
	if want := "<?php\n/**\n * @Foo({bar = \"baz\"})\n */\nclass Bar {}\n"; !changed || got != want {
		t.Fatalf("colon: changed=%v got=%q", changed, got)
	}

	// an argument "=" (outside an array) is left alone
	if _, changed := apply(t, f, "<?php\n/**\n * @Foo(bar = \"baz\")\n */\nclass Bar {}\n"); changed {
		t.Fatal("argument assignment must not change")
	}

	// a plain phpdoc tag is not a Doctrine annotation
	if _, changed := apply(t, f, "<?php\n/**\n * @param string $x\n */\nclass Bar {}\n"); changed {
		t.Fatal("ignored tag must not change")
	}

	// a docblock that does not precede a class is skipped
	if _, changed := apply(t, f, "<?php\n/**\n * @Foo({bar : \"baz\"})\n */\nfunction f() {}\n"); changed {
		t.Fatal("non-class docblock must be skipped")
	}
}
