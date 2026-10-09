package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGenImportsNoUnneededImportAlias(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoUnneededImportAlias{}, "<?php use App\\Foo as Foo;")
	if want := "<?php use App\\Foo;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// alias differs from the last segment - left alone
	if _, changed := apply(t, NoUnneededImportAlias{}, "<?php use App\\Foo as Bar;"); changed {
		t.Fatal("alias differing from last segment must be kept")
	}
	// case-sensitive: foo != Foo
	if _, changed := apply(t, NoUnneededImportAlias{}, "<?php use App\\Foo as foo;"); changed {
		t.Fatal("case-different alias must be kept")
	}
	// already clean - no-op
	if _, changed := apply(t, NoUnneededImportAlias{}, "<?php use App\\Foo;"); changed {
		t.Fatal("import without alias must not change")
	}
	// use function / use const
	got, changed = apply(t, NoUnneededImportAlias{}, "<?php use function A\\b as b; use const A\\C as C;")
	if want := "<?php use function A\\b; use const A\\C;"; !changed || got != want {
		t.Fatalf("func/const: changed=%v got=%q want=%q", changed, got, want)
	}
	// grouped import
	got, changed = apply(t, NoUnneededImportAlias{}, "<?php use App\\{Foo as Foo, Bar as Baz};")
	if want := "<?php use App\\{Foo, Bar as Baz};"; !changed || got != want {
		t.Fatalf("group: changed=%v got=%q want=%q", changed, got, want)
	}
	// closure use must not be touched
	if _, changed := apply(t, NoUnneededImportAlias{}, "<?php $f = function () use ($a) {};"); changed {
		t.Fatal("closure use must not change")
	}
	// idempotent
	once, _ := apply(t, NoUnneededImportAlias{}, "<?php use App\\Foo as Foo;")
	if twice, changed := apply(t, NoUnneededImportAlias{}, once); changed || twice != once {
		t.Fatalf("not idempotent: %q -> %q", once, twice)
	}
}

func TestGenImportsCleanNamespace(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, CleanNamespace{}, "<?php namespace A \\ B \\ C;")
	if want := "<?php namespace A\\B\\C;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// use path
	got, changed = apply(t, CleanNamespace{}, "<?php use A \\ B;")
	if want := "<?php use A\\B;"; !changed || got != want {
		t.Fatalf("use: changed=%v got=%q want=%q", changed, got, want)
	}
	// comment inside the path
	got, changed = apply(t, CleanNamespace{}, "<?php namespace App /* x */ \\ Sub;")
	if want := "<?php namespace App\\Sub;"; !changed || got != want {
		t.Fatalf("comment: changed=%v got=%q want=%q", changed, got, want)
	}
	// already clean - no-op
	if _, changed := apply(t, CleanNamespace{}, "<?php namespace App\\Sub;"); changed {
		t.Fatal("clean namespace must not change")
	}
	if _, changed := apply(t, CleanNamespace{}, "<?php use App\\Foo;"); changed {
		t.Fatal("clean use must not change")
	}
	// trailing space before ; is not around a backslash - kept
	if _, changed := apply(t, CleanNamespace{}, "<?php namespace App ;"); changed {
		t.Fatal("space before ; without a separator must be kept")
	}
	// idempotent
	once, _ := apply(t, CleanNamespace{}, "<?php namespace A \\ B \\ C;")
	if twice, changed := apply(t, CleanNamespace{}, once); changed || twice != once {
		t.Fatalf("not idempotent: %q -> %q", once, twice)
	}
}

// TestGenImportsSourceURLs verifies each SourceURL matches SourceURLFor(Name()).
func TestGenImportsSourceURLs(t *testing.T) {
	t.Parallel()
	for _, f := range []fixer.Fixer{NoUnneededImportAlias{}, CleanNamespace{}} {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}
