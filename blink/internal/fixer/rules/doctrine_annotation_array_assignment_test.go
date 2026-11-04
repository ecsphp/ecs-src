package rules

import "testing"

func TestDoctrineAnnotationArrayAssignment(t *testing.T) {
	t.Parallel()
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

func TestDoctrineAnnotationArrayAssignmentConfig(t *testing.T) {
	t.Parallel()
	const in = "<?php\n/**\n * @Foo({bar = \"baz\", qux : 1}, x = 2)\n */\nclass Bar {}\n"
	cases := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{"default (empty config)", map[string]any{}, "<?php\n/**\n * @Foo({bar = \"baz\", qux = 1}, x = 2)\n */\nclass Bar {}\n"},
		{"colon", map[string]any{"operator": ":"}, "<?php\n/**\n * @Foo({bar : \"baz\", qux : 1}, x = 2)\n */\nclass Bar {}\n"},
		{"ignored tag", map[string]any{"operator": ":", "ignored_tags": []any{"Foo"}}, in},
	}
	for _, c := range cases {
		f := DoctrineAnnotationArrayAssignment{}.WithConfig(c.config).(fixerRule)
		if got, _ := apply(t, f, in); got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
