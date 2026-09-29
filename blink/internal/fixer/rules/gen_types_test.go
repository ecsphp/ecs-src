package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGenTypesCompactNullableTypeDeclaration(t *testing.T) {
	cases := []struct {
		src, want string
		changed   bool
	}{
		{"<?php function f(? int $x){}", "<?php function f(?int $x){}", true},
		{"<?php function f(): ? int {}", "<?php function f(): ?int {}", true},
		{"<?php function __construct(public ? int $x){}", "<?php function __construct(public ?int $x){}", true},
		{"<?php function f(int|? Foo $x){}", "<?php function f(int|?Foo $x){}", true},
		// already compact - no-op
		{"<?php function f(?int $x){}", "<?php function f(?int $x){}", false},
		// ternary and friends must never be touched
		{"<?php $a = $b ? $c : $d;", "<?php $a = $b ? $c : $d;", false},
		{"<?php $a = $b ?: $c;", "<?php $a = $b ?: $c;", false},
		{"<?php $a = $b ?? $c;", "<?php $a = $b ?? $c;", false},
		{"<?php $a = $b?->c;", "<?php $a = $b?->c;", false},
	}
	for _, c := range cases {
		got, changed := apply(t, CompactNullableTypeDeclaration{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, _ := apply(t, CompactNullableTypeDeclaration{}, got); again != got {
			t.Fatalf("not idempotent: %q -> %q", got, again)
		}
	}
}

func TestGenTypesNativeFunctionTypeDeclarationCasing(t *testing.T) {
	cases := []struct {
		src, want string
		changed   bool
	}{
		{"<?php function f(INT $x): STRING {}", "<?php function f(int $x): string {}", true},
		{"<?php function f(?INT $x): ?STRING {}", "<?php function f(?int $x): ?string {}", true},
		{"<?php function f(): INT|STRING {}", "<?php function f(): int|string {}", true},
		// already lowercase - no-op
		{"<?php function f(int $x): string {}", "<?php function f(int $x): string {}", false},
		// class names must never be lowercased
		{"<?php function f(Foo $x): Bar {}", "<?php function f(Foo $x): Bar {}", false},
		// a constant in the body is not a type declaration
		{"<?php function f() { return INT; }", "<?php function f() { return INT; }", false},
	}
	for _, c := range cases {
		got, changed := apply(t, NativeFunctionTypeDeclarationCasing{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, _ := apply(t, NativeFunctionTypeDeclarationCasing{}, got); again != got {
			t.Fatalf("not idempotent: %q -> %q", got, again)
		}
	}
}

func TestGenTypesSourceURLs(t *testing.T) {
	fixers := []fixer.Fixer{
		CompactNullableTypeDeclaration{},
		NativeFunctionTypeDeclarationCasing{},
	}
	for _, f := range fixers {
		if want := fixer.SourceURLFor(f.Name()); f.SourceURL() != want {
			t.Fatalf("%T SourceURL()=%q want %q", f, f.SourceURL(), want)
		}
	}
}
