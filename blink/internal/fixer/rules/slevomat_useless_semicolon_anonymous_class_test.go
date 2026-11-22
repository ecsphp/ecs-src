package rules

import "testing"

func TestSlevomatUselessSemicolonKeepsAnonymousClassTerminator(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"$x = new class {};", "$x = new class {};"},
		{"$x = new readonly class {};", "$x = new readonly class {};"},
		{"$x = new READONLY class {};", "$x = new READONLY class {};"},
		{"$x = new #[Foo] class {};", "$x = new #[Foo] class {};"},
		{"$x = new #[Foo] readonly class {};", "$x = new #[Foo] readonly class {};"},
		{"$x = new #[Foo(1)] #[Bar] readonly class {};", "$x = new #[Foo(1)] #[Bar] readonly class {};"},
		{"$x = new readonly /* c */ class($a) {};", "$x = new readonly /* c */ class($a) {};"},
		{"readonly class A {};", "readonly class A {}"},
		{"final readonly class A {};", "final readonly class A {}"},
	}
	for _, c := range cases {
		got, changed := apply(t, SlevomatUselessSemicolon{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
