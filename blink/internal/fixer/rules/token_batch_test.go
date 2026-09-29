package rules

import "testing"

func TestAttributeBlockNoSpaces(t *testing.T) {
	got, changed := apply(t, AttributeBlockNoSpaces{}, "<?php #[ Foo ] function f() {}")
	if want := "<?php #[Foo] function f() {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// inner content is preserved, only the outer boundary is trimmed
	got, changed = apply(t, AttributeBlockNoSpaces{}, "<?php #[ Route('/x', methods: ['GET']) ]")
	if want := "<?php #[Route('/x', methods: ['GET'])]"; !changed || got != want {
		t.Fatalf("inner: changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, AttributeBlockNoSpaces{}, "<?php #[Foo]"); changed {
		t.Fatal("already-trimmed attribute must not change")
	}
	idempotent(t, AttributeBlockNoSpaces{}, "<?php #[Foo] class A {}")
}

func TestNoSpaceAroundDoubleColon(t *testing.T) {
	got, changed := apply(t, NoSpaceAroundDoubleColon{}, "<?php Foo :: bar(); Baz ::CONST;")
	if want := "<?php Foo::bar(); Baz::CONST;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, NoSpaceAroundDoubleColon{}, "<?php Foo::bar();"); changed {
		t.Fatal("already-tight :: must not change")
	}
	// whitespace spanning a newline is preserved
	if _, changed := apply(t, NoSpaceAroundDoubleColon{}, "<?php Foo::\n    bar();"); changed {
		t.Fatal("multi-line :: must be kept")
	}
	idempotent(t, NoSpaceAroundDoubleColon{}, "<?php Foo::bar();")
}

func TestTrimArraySpaces(t *testing.T) {
	got, changed := apply(t, TrimArraySpaces{}, "<?php $a = [ 1, 2 ];")
	if want := "<?php $a = [1, 2];"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// offset access must be left alone (that is NoSpacesAroundOffset's job)
	if _, changed := apply(t, TrimArraySpaces{}, "<?php $x = $a[ 0 ];"); changed {
		t.Fatal("array offset must not be trimmed")
	}
	if _, changed := apply(t, TrimArraySpaces{}, "<?php $a = [1, 2];"); changed {
		t.Fatal("already-trimmed literal must not change")
	}
	// multi-line arrays keep their newlines
	if _, changed := apply(t, TrimArraySpaces{}, "<?php $a = [\n    1,\n];"); changed {
		t.Fatal("multi-line array must be kept")
	}
	idempotent(t, TrimArraySpaces{}, "<?php $a = [1, 2];")
}

func TestNativeTypeDeclarationCasing(t *testing.T) {
	got, changed := apply(t, NativeTypeDeclarationCasing{}, "<?php function f(Int $a, ?STRING $b): VOID {}")
	if want := "<?php function f(int $a, ?string $b): void {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// property type and a union return type
	got, changed = apply(t, NativeTypeDeclarationCasing{}, "<?php class A { public Array $items; function g(): Int|FALSE {} }")
	if want := "<?php class A { public array $items; function g(): int|false {} }"; !changed || got != want {
		t.Fatalf("prop/union: changed=%v got=%q want=%q", changed, got, want)
	}
	// class names (not reserved words) must never be touched
	if _, changed := apply(t, NativeTypeDeclarationCasing{}, "<?php function f(DateTime $a): Response {}"); changed {
		t.Fatal("class name must not be lowercased")
	}
	// value-position constants are ConstantCase's job, not this fixer's
	if _, changed := apply(t, NativeTypeDeclarationCasing{}, "<?php $x = TRUE; return NULL;"); changed {
		t.Fatal("constant values must not be touched")
	}
	if _, changed := apply(t, NativeTypeDeclarationCasing{}, "<?php function f(int $a): void {}"); changed {
		t.Fatal("already-lowercase types must not change")
	}
	idempotent(t, NativeTypeDeclarationCasing{}, "<?php function f(int $a): void {}")
}

func TestHeredocToNowdoc(t *testing.T) {
	got, changed := apply(t, HeredocToNowdoc{}, "<?php $a = <<<EOT\nhello world\nEOT;\n")
	if want := "<?php $a = <<<'EOT'\nhello world\nEOT;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// interpolation, braces or escapes keep it a heredoc
	if _, changed := apply(t, HeredocToNowdoc{}, "<?php $a = <<<EOT\nhas $var\nEOT;\n"); changed {
		t.Fatal("heredoc with $ must not convert")
	}
	if _, changed := apply(t, HeredocToNowdoc{}, "<?php $a = <<<EOT\ntab\\t\nEOT;\n"); changed {
		t.Fatal("heredoc with escape must not convert")
	}
	// already a nowdoc
	if _, changed := apply(t, HeredocToNowdoc{}, "<?php $a = <<<'EOT'\nplain\nEOT;\n"); changed {
		t.Fatal("nowdoc must not change")
	}
	idempotent(t, HeredocToNowdoc{}, "<?php $a = <<<'EOT'\nhello world\nEOT;\n")
}

func TestNoUselessConcat(t *testing.T) {
	got, changed := apply(t, NoUselessConcatOperator{}, "<?php $s = 'a' . 'b'; $t = \"x\".\"y\";")
	if want := "<?php $s = 'ab'; $t = \"xy\";"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// chained same-quote literals collapse fully
	got, changed = apply(t, NoUselessConcatOperator{}, "<?php $s = 'a'.'b'.'c';")
	if want := "<?php $s = 'abc';"; !changed || got != want {
		t.Fatalf("chain: changed=%v got=%q want=%q", changed, got, want)
	}
	// interpolated double-quoted strings must not merge
	if _, changed := apply(t, NoUselessConcatOperator{}, "<?php $s = \"a$b\" . \"c\";"); changed {
		t.Fatal("interpolated string must not merge")
	}
	// different quote styles must not merge
	if _, changed := apply(t, NoUselessConcatOperator{}, "<?php $s = 'a' . \"b\";"); changed {
		t.Fatal("mismatched quotes must not merge")
	}
	// a real operand (not a literal) is left alone
	if _, changed := apply(t, NoUselessConcatOperator{}, "<?php $s = 'a' . $b;"); changed {
		t.Fatal("string . variable must not merge")
	}
	idempotent(t, NoUselessConcatOperator{}, "<?php $s = 'abc';")
}

func TestNoBinaryString(t *testing.T) {
	got, changed := apply(t, NoBinaryString{}, "<?php $a = b\"foo\"; $b = B'bar';")
	if want := "<?php $a = \"foo\"; $b = 'bar';"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// a plain string has no prefix to drop
	if _, changed := apply(t, NoBinaryString{}, "<?php $a = \"foo\";"); changed {
		t.Fatal("plain string must not change")
	}
	// a lone identifier "b" that is not a string prefix is left alone
	if _, changed := apply(t, NoBinaryString{}, "<?php $a = b; foo(b, 1);"); changed {
		t.Fatal("identifier b must not be removed")
	}
	idempotent(t, NoBinaryString{}, "<?php $a = \"foo\";")
}
