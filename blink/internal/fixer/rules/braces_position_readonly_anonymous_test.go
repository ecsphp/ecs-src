package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestBracesPositionReadonlyAnonymousClass(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"$a = new class {};", "$a = new class {};"},
		{"$a = new class {\n};", "$a = new class {\n};"},
		{"$a = new class\n{\n};", "$a = new class {\n};"},
		{"$a = new readonly class {\n};", "$a = new readonly class {\n};"},
		{"$a = new readonly class\n{\n};", "$a = new readonly class {\n};"},
		{"$a = new readonly class() {\n};", "$a = new readonly class() {\n};"},
		{"$a = new readonly class()\n{\n};", "$a = new readonly class() {\n};"},
		{"$a = new readonly class($b) {\n};", "$a = new readonly class($b) {\n};"},
		{"$a = new readonly class($b) implements C {\n};", "$a = new readonly class($b) implements C {\n};"},
		{"$a = new readonly class($b) implements C\n{\n};", "$a = new readonly class($b) implements C {\n};"},
		{"$a = new readonly class($b) extends D implements C {\n};", "$a = new readonly class($b) extends D implements C {\n};"},
		{"$a = new READONLY class {\n};", "$a = new READONLY class {\n};"},
		{"$a = new #[Foo] class {\n};", "$a = new #[Foo] class {\n};"},
		{"$a = new #[Foo] readonly class {\n};", "$a = new #[Foo] readonly class {\n};"},
		{"$a = new #[Foo] readonly class\n{\n};", "$a = new #[Foo] readonly class {\n};"},
		{"$a = new /* c */ readonly class {\n};", "$a = new /* c */ readonly class {\n};"},
		{"$a = new readonly /* c */ class {\n};", "$a = new readonly /* c */ class {\n};"},
		{"$a = new\n    readonly class {\n};", "$a = new\n    readonly class {\n};"},
		{"$a = new readonly\n    class {\n};", "$a = new readonly\n    class {\n};"},
		{"foo(new readonly class {\n});", "foo(new readonly class {\n});"},
		{"foo(new readonly class($b) implements C {\n}, 1);", "foo(new readonly class($b) implements C {\n}, 1);"},
		{"return new readonly class {\n};", "return new readonly class {\n};"},
		{"readonly class A {\n}", "readonly class A\n{\n}"},
		{"readonly class A\n{\n}", "readonly class A\n{\n}"},
		{"final readonly class A {\n}", "final readonly class A\n{\n}"},
		{"readonly final class A {\n}", "readonly final class A\n{\n}"},
		{"abstract readonly class A {\n}", "abstract readonly class A\n{\n}"},
		{"#[Foo] readonly class A {\n}", "#[Foo] readonly class A\n{\n}"},
	}
	for _, c := range cases {
		got, changed := apply(t, BracesPosition{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
