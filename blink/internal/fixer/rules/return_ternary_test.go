package rules

import "testing"

func TestReturnTypeDeclaration(t *testing.T) {
	// space before the colon is removed, one space kept after
	got, changed := apply(t, ReturnTypeDeclaration{}, "<?php function f() : int {}")
	if want := "<?php function f(): int {}"; !changed || got != want {
		t.Fatalf("space-before: changed=%v got=%q want=%q", changed, got, want)
	}
	// missing space after the colon is added
	got, changed = apply(t, ReturnTypeDeclaration{}, "<?php function f():int {}")
	if want := "<?php function f(): int {}"; !changed || got != want {
		t.Fatalf("no-space: changed=%v got=%q want=%q", changed, got, want)
	}
	// nullable return type
	got, changed = apply(t, ReturnTypeDeclaration{}, "<?php function f() :?int {}")
	if want := "<?php function f(): ?int {}"; !changed || got != want {
		t.Fatalf("nullable: changed=%v got=%q want=%q", changed, got, want)
	}
	// named function by reference
	got, changed = apply(t, ReturnTypeDeclaration{}, "<?php function &f() : int {}")
	if want := "<?php function &f(): int {}"; !changed || got != want {
		t.Fatalf("by-ref: changed=%v got=%q want=%q", changed, got, want)
	}
	// arrow function
	got, changed = apply(t, ReturnTypeDeclaration{}, "<?php $f = fn($x):int => $x;")
	if want := "<?php $f = fn($x): int => $x;"; !changed || got != want {
		t.Fatalf("arrow: changed=%v got=%q want=%q", changed, got, want)
	}

	// no-op on already-correct code
	if _, changed := apply(t, ReturnTypeDeclaration{}, "<?php function f(): int {}"); changed {
		t.Fatal("correct return type must not change")
	}
	// idempotent
	once, _ := apply(t, ReturnTypeDeclaration{}, "<?php function f():int {}")
	twice, changed := apply(t, ReturnTypeDeclaration{}, once)
	if changed || twice != once {
		t.Fatalf("not idempotent: once=%q twice=%q changed=%v", once, twice, changed)
	}

	// must NOT touch ternary, static access, case labels or alternative syntax
	for _, src := range []string{
		"<?php $x = $a ? 1 : 2;",
		"<?php $x = $a?1:2;",
		"<?php echo Foo::BAR;",
		"<?php switch ($a) { case 1: break; }",
		"<?php if ($a): echo 1; endif;",
		"<?php $x = foo() ? 1 : 2;",
	} {
		if got, changed := apply(t, ReturnTypeDeclaration{}, src); changed || got != src {
			t.Fatalf("must not touch %q: changed=%v got=%q", src, changed, got)
		}
	}
}

func TestTernaryOperatorSpaces(t *testing.T) {
	got, changed := apply(t, TernaryOperatorSpaces{}, "<?php $x = $a?$b:$c;")
	if want := "<?php $x = $a ? $b : $c;"; !changed || got != want {
		t.Fatalf("tight: changed=%v got=%q want=%q", changed, got, want)
	}
	// messy spacing collapses to single spaces
	got, changed = apply(t, TernaryOperatorSpaces{}, "<?php $x = $a  ?  $b  :  $c;")
	if want := "<?php $x = $a ? $b : $c;"; !changed || got != want {
		t.Fatalf("messy: changed=%v got=%q want=%q", changed, got, want)
	}
	// elvis keeps "?:" adjacent
	got, changed = apply(t, TernaryOperatorSpaces{}, "<?php $x = $a?:$b;")
	if want := "<?php $x = $a ?: $b;"; !changed || got != want {
		t.Fatalf("elvis: changed=%v got=%q want=%q", changed, got, want)
	}
	// nested ternary in the else branch
	got, changed = apply(t, TernaryOperatorSpaces{}, "<?php $x = $a?$b:($c?$d:$e);")
	if want := "<?php $x = $a ? $b : ($c ? $d : $e);"; !changed || got != want {
		t.Fatalf("nested: changed=%v got=%q want=%q", changed, got, want)
	}

	// no-op on already-correct code
	if _, changed := apply(t, TernaryOperatorSpaces{}, "<?php $x = $a ? $b : $c;"); changed {
		t.Fatal("correct ternary must not change")
	}
	if _, changed := apply(t, TernaryOperatorSpaces{}, "<?php $x = $a ?: $b;"); changed {
		t.Fatal("correct elvis must not change")
	}
	// idempotent
	once, _ := apply(t, TernaryOperatorSpaces{}, "<?php $x = $a?$b:$c;")
	twice, changed := apply(t, TernaryOperatorSpaces{}, once)
	if changed || twice != once {
		t.Fatalf("not idempotent: once=%q twice=%q changed=%v", once, twice, changed)
	}

	// must NOT touch nullable types, return types, coalescing or nullsafe access
	for _, src := range []string{
		"<?php function f(?int $x) {}",
		"<?php function f(): ?int {}",
		"<?php private ?int $x;",
		"<?php $x = $a ?? $b;",
		"<?php $x = $a?->b;",
	} {
		if got, changed := apply(t, TernaryOperatorSpaces{}, src); changed || got != src {
			t.Fatalf("must not touch %q: changed=%v got=%q", src, changed, got)
		}
	}

	// multiline ternary alignment is preserved
	if _, changed := apply(t, TernaryOperatorSpaces{}, "<?php $x = $a\n    ? $b\n    : $c;"); changed {
		t.Fatal("multiline ternary must be kept")
	}
}
