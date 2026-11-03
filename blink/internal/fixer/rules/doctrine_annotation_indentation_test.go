package rules

import (
	"strings"
	"testing"
)

func TestDoctrineAnnotationIndentation(t *testing.T) {
	t.Parallel()
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

	// "private(set)" is a modifier even on a promoted constructor parameter
	src := "<?php\nclass A {\n    public function __construct(\n        /**\n         *   @Foo(\n         * x = 1)\n         */\n        private(set) int $p,\n    ) {}\n}\n"
	if got, changed := apply(t, f, src); !changed || got != strings.Replace(src, "*   @Foo(", "* @Foo(", 1) {
		t.Fatalf("private(set) parameter: changed=%v got=%q", changed, got)
	}
}
