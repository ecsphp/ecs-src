package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen2TypesTypesSpaces(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src, want string
		changed   bool
	}{
		// parameter union / intersection
		{"<?php function f(int | string $x){}", "<?php function f(int|string $x){}", true},
		{"<?php function f(A & B $x){}", "<?php function f(A&B $x){}", true},
		{"<?php function f(int|string $x){}", "<?php function f(int|string $x){}", false},
		// return type
		{"<?php function f(): int | string {}", "<?php function f(): int|string {}", true},
		{"<?php function f(): A & B {}", "<?php function f(): A&B {}", true},
		// typed property after a visibility modifier
		{"<?php class C { public int | string $x; }", "<?php class C { public int|string $x; }", true},
		{"<?php class C { private A & B $x; }", "<?php class C { private A&B $x; }", true},
		// promoted constructor property
		{"<?php function __construct(public int | string $x){}", "<?php function __construct(public int|string $x){}", true},
		// three-part union
		{"<?php function f(int | string | float $x){}", "<?php function f(int|string|float $x){}", true},
		// nullable member inside a union is kept, only surrounding space removed
		{"<?php function f(int | ?Foo $x){}", "<?php function f(int|?Foo $x){}", true},

		// dangerous bitwise cases - must never be touched
		{"<?php $a = $b | $c;", "<?php $a = $b | $c;", false},
		{"<?php $a = $b & $c;", "<?php $a = $b & $c;", false},
		{"<?php $a = 1 | 2;", "<?php $a = 1 | 2;", false},
		{"<?php $x = FOO | BAR;", "<?php $x = FOO | BAR;", false},
		// by-reference parameter is not an intersection type
		{"<?php function f(int &$x){}", "<?php function f(int &$x){}", false},
		{"<?php function f(A &$x){}", "<?php function f(A &$x){}", false},
		{"<?php function f(&$x){}", "<?php function f(&$x){}", false},
		// bitwise default value inside a parameter list must be left alone
		{"<?php function f($a = FOO | BAR){}", "<?php function f($a = FOO | BAR){}", false},
		// newline around the operator is preserved
		{"<?php function f(int\n| string $x){}", "<?php function f(int\n|string $x){}", true},
	}
	for _, c := range cases {
		got, changed := apply(t, TypesSpaces{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		// idempotent
		if again, _ := apply(t, TypesSpaces{}, got); again != got {
			t.Fatalf("not idempotent: %q -> %q", got, again)
		}
	}
}

func TestGen2TypesSourceURLs(t *testing.T) {
	t.Parallel()
	fixers := []fixer.Fixer{
		TypesSpaces{},
	}
	for _, f := range fixers {
		if want := fixer.SourceURLFor(f.Name()); f.SourceURL() != want {
			t.Fatalf("%T SourceURL()=%q want %q", f, f.SourceURL(), want)
		}
	}
}
