package rules

import (
	"strings"
	"testing"
)

func TestDoctrineAnnotationSpaces(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	noop := []string{
		// already clean
		`<?php /** @ORM\Column(type="string", nullable=true) */ class C {}`,
		`<?php /** @Empty() */ class C {}`,
		// plain phpdoc tags are not Doctrine annotations
		"<?php\n/**\n * @param string $x\n * @return int\n */\nfunction f($x) {}",
		// lowercase tag with parens is left alone
		`<?php /** @see method() */ class C {}`,
		// a gap holding a newline just inside the parentheses is kept
		"<?php\n/**\n * @Multi(\n *     a=1\n * )\n */\nclass C {}",
	}
	for _, src := range noop {
		if got, changed := apply(t, DoctrineAnnotationSpaces{}, src); changed || got != src {
			t.Fatalf("expected no-op, changed=%v\n src=%q\n got=%q", changed, src, got)
		}
	}
}

// TestDoctrineAnnotationSpacesCandidates follows nextElementAcceptsDoctrineAnnotations,
// as the other Doctrine annotation rules do: a "static", "var" or "array"
// member, or a function outside a class, is no candidate.
func TestDoctrineAnnotationSpacesCandidates(t *testing.T) {
	t.Parallel()
	in := "<?php\nclass C\n{\n    /** @A(a = 1) */\n    public static $s;\n\n    /** @B(a = 1) */\n    var $v;\n\n    /** @C(a = 1) */\n    private array $l;\n\n    /** @D(a = 1) */\n    public $p;\n\n    /** @E(a = 1) */\n    private(set) int $q;\n}\n\n/** @F(a = 1) */\nfunction f() {}\n"
	want := strings.NewReplacer("@D(a = 1)", "@D(a=1)", "@E(a = 1)", "@E(a=1)").Replace(in)
	if got, changed := apply(t, DoctrineAnnotationSpaces{}, in); !changed || got != want {
		t.Fatalf("changed=%v\n got=%q\nwant=%q", changed, got, want)
	}
}
