package rules

import "testing"

func TestDoctrineAnnotationSpaces(t *testing.T) {
	cases := []struct{ in, want string }{
		// space before "(" and just inside "(" / ")" removed
		{`<?php /** @Spaced ( "a" ) */ class C {}`, `<?php /** @Spaced("a") */ class C {}`},
		// argument "=" loses its spaces, comma spacing kept
		{`<?php /** @Args(a = 1, b="x") */ class C {}`, `<?php /** @Args(a=1, b="x") */ class C {}`},
		// space before a comma removed
		{`<?php /** @Foo(a="x" , b=1) */ class C {}`, `<?php /** @Foo(a="x", b=1) */ class C {}`},
		// one space added after a comma that had none (multiples are kept)
		{`<?php /** @Route("/p",name="r",  x=1) */ class C {}`, `<?php /** @Route("/p", name="r",  x=1) */ class C {}`},
		// argument "=" tightened, array "=" (inside {}) gets one space each side
		{`<?php /** @Nested(x = {"p" = 1, "q"=2}) */ class C {}`, `<?php /** @Nested(x={"p" = 1, "q" = 2}) */ class C {}`},
		// string contents (commas, "=", parens) are untouched
		{`<?php /** @Str(v = "has , comma = and ( paren") */ class C {}`, `<?php /** @Str(v="has , comma = and ( paren") */ class C {}`},
	}
	for _, c := range cases {
		got, changed := apply(t, DoctrineAnnotationSpaces{}, c.in)
		if !changed || got != c.want {
			t.Fatalf("changed=%v\n got=%q\nwant=%q", changed, got, c.want)
		}
		// idempotent: a second pass is a no-op
		if again, changed2 := apply(t, DoctrineAnnotationSpaces{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: changed=%v got=%q", changed2, again)
		}
	}
}

func TestDoctrineAnnotationSpacesNoOp(t *testing.T) {
	noop := []string{
		// already clean
		`<?php /** @ORM\Column(type="string", nullable=true) */ class C {}`,
		`<?php /** @Empty() */ class C {}`,
		// plain phpdoc tags are not Doctrine annotations
		"<?php\n/**\n * @param string $x\n * @return int\n */\nfunction f($x) {}",
		// lowercase tag with parens is left alone
		`<?php /** @see method() */ class C {}`,
		// a multi-line annotation is skipped (safe under-fire)
		"<?php\n/**\n * @Multi(\n *     a = 1\n * )\n */\nclass C {}",
	}
	for _, src := range noop {
		if got, changed := apply(t, DoctrineAnnotationSpaces{}, src); changed || got != src {
			t.Fatalf("expected no-op, changed=%v\n src=%q\n got=%q", changed, src, got)
		}
	}
}
