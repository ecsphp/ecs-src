package rules

import "testing"

func TestCommentToPhpdocSkipsAttributes(t *testing.T) {
	t.Parallel()
	f := CommentToPhpdoc{}
	for src, want := range map[string]string{
		"<?php\n$x = 1;\n/* @see Foo */\n#[A]\n#[B([1])]\nfunction f() {}\n": "<?php\n$x = 1;\n/** @see Foo */\n#[A]\n#[B([1])]\nfunction f() {}\n",
		"<?php\n$x = 1;\n/* @var X */\n(#[A] fn () => 1)();\n":               "<?php\n$x = 1;\n/** @var X */\n(#[A] fn () => 1)();\n",
	} {
		if got, changed := apply(t, f, src); !changed || got != want {
			t.Fatalf("%q: changed=%v got=%q", src, changed, got)
		}
	}
}
