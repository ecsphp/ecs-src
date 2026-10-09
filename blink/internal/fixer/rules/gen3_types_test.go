package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen3TypesTypeDeclarationSpaces(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src, want string
		changed   bool
	}{
		// parameters
		{"<?php function f(int$x){}", "<?php function f(int $x){}", true},
		{"<?php function f(int  $x){}", "<?php function f(int $x){}", true},
		{"<?php function f(Foo$x){}", "<?php function f(Foo $x){}", true},
		// native type that lexes as a keyword (not covered by FunctionTypehintSpace)
		{"<?php function f(array$x){}", "<?php function f(array $x){}", true},
		{"<?php fn(int$x) => $x;", "<?php fn(int $x) => $x;", true},
		// typed properties
		{"<?php class C { private int$foo; }", "<?php class C { private int $foo; }", true},
		{"<?php class C { public ?string  $bar; }", "<?php class C { public ?string $bar; }", true},
		{"<?php class C { protected readonly Foo$baz; }", "<?php class C { protected readonly Foo $baz; }", true},
		// promoted constructor property
		{"<?php function __construct(private int$x){}", "<?php function __construct(private int $x){}", true},
		// already correct - no-op
		{"<?php function f(int $x){}", "<?php function f(int $x){}", false},
		{"<?php class C { private int $foo; }", "<?php class C { private int $foo; }", false},
		// untyped parameter must not gain a space
		{"<?php function f($x){}", "<?php function f($x){}", false},
		// a plain assignment / local variable is not a type declaration
		{"<?php $foo = 1;", "<?php $foo = 1;", false},
		// a function call is not a signature
		{"<?php strlen($x);", "<?php strlen($x);", false},
		// newline between type and variable is left intact
		{"<?php function f(int\n$x){}", "<?php function f(int\n$x){}", false},
	}
	for _, c := range cases {
		got, changed := apply(t, TypeDeclarationSpaces{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, _ := apply(t, TypeDeclarationSpaces{}, got); again != got {
			t.Fatalf("not idempotent: %q -> %q", got, again)
		}
	}
}

func TestGen3TypesNullableTypeDeclarationForDefaultNullValue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src, want string
		changed   bool
	}{
		// happy paths
		{"<?php function f(int $x = null){}", "<?php function f(?int $x = null){}", true},
		{"<?php function f(Foo $x = null){}", "<?php function f(?Foo $x = null){}", true},
		{"<?php function f(\\Foo\\Bar $x = null){}", "<?php function f(?\\Foo\\Bar $x = null){}", true},
		{"<?php function m(string $a, int $b = null){}", "<?php function m(string $a, ?int $b = null){}", true},
		{"<?php function __construct(private int $x = null){}", "<?php function __construct(private ?int $x = null){}", true},
		{"<?php function f(int &$x = null){}", "<?php function f(?int &$x = null){}", true},
		{"<?php function f(int $x = NULL){}", "<?php function f(?int $x = NULL){}", true},
		// two nullable-default params at once
		{"<?php function f(int $a = null, Foo $b = null){}", "<?php function f(?int $a = null, ?Foo $b = null){}", true},
		// already nullable - no-op
		{"<?php function f(?int $x = null){}", "<?php function f(?int $x = null){}", false},
		// no default - no-op
		{"<?php function f(int $x){}", "<?php function f(int $x){}", false},
		// non-null default - no-op
		{"<?php function f(int $x = 5){}", "<?php function f(int $x = 5){}", false},
		// untyped parameter - no-op
		{"<?php function f($x = null){}", "<?php function f($x = null){}", false},
		// mixed / standalone null - no-op
		{"<?php function f(mixed $x = null){}", "<?php function f(mixed $x = null){}", false},
		{"<?php function f(null $x = null){}", "<?php function f(null $x = null){}", false},
		// union / intersection skipped (would need |null, not ?prefix)
		{"<?php function f(int|string $x = null){}", "<?php function f(int|string $x = null){}", false},
		{"<?php function f(A&B $x = null){}", "<?php function f(A&B $x = null){}", false},
		// null as part of a larger default expression is not a plain "= null"
		{"<?php function f(int $x = null ?? 1){}", "<?php function f(int $x = null ?? 1){}", false},
		// not a parameter list - a call argument named null must be untouched
		{"<?php foo($x = null);", "<?php foo($x = null);", false},
	}
	for _, c := range cases {
		got, changed := apply(t, NullableTypeDeclarationForDefaultNullValue{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, _ := apply(t, NullableTypeDeclarationForDefaultNullValue{}, got); again != got {
			t.Fatalf("not idempotent: %q -> %q", got, again)
		}
	}
}

func TestGen3TypesSourceURLs(t *testing.T) {
	t.Parallel()
	fixers := []fixer.Fixer{
		TypeDeclarationSpaces{},
		NullableTypeDeclarationForDefaultNullValue{},
	}
	for _, f := range fixers {
		if want := fixer.SourceURLFor(f.Name()); f.SourceURL() != want {
			t.Fatalf("%T SourceURL()=%q want %q", f, f.SourceURL(), want)
		}
	}
}
