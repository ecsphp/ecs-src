package rules

import (
	"testing"

	"blink/internal/lexer"
	"blink/internal/tokens"
)

func applyAt(t *testing.T, r fixerRule, path, src string) (string, bool) {
	t.Helper()
	s := tokens.New(lexer.Lex(src))
	s.SetPath(path)
	changed := r.Fix(s)
	return s.Render(), changed
}

func TestPsrAutoloading(t *testing.T) {
	t.Parallel()
	f := PsrAutoloading{}

	// class name is corrected to the file basename
	got, changed := applyAt(t, f, "/project/src/Foo.php", "<?php\nnamespace App;\nclass Bar {}\n")
	if want := "<?php\nnamespace App;\nclass Foo {}\n"; !changed || got != want {
		t.Fatalf("rename: changed=%v got=%q", changed, got)
	}

	// interface, trait and enum names are corrected too
	if got, changed := applyAt(t, f, "/src/Foo.php", "<?php\ninterface Bar {}\n"); !changed || got != "<?php\ninterface Foo {}\n" {
		t.Fatalf("interface: changed=%v got=%q", changed, got)
	}
	if got, changed := applyAt(t, f, "/src/Foo.php", "<?php\ntrait Bar {}\n"); !changed || got != "<?php\ntrait Foo {}\n" {
		t.Fatalf("trait: changed=%v got=%q", changed, got)
	}
	if got, changed := applyAt(t, f, "/src/Foo.php", "<?php\nenum Bar {}\n"); !changed || got != "<?php\nenum Foo {}\n" {
		t.Fatalf("enum: changed=%v got=%q", changed, got)
	}

	// already-matching name is a no-op
	if _, changed := applyAt(t, f, "/project/src/Foo.php", "<?php\nnamespace App;\nclass Foo {}\n"); changed {
		t.Fatal("matching name must be a no-op")
	}
}

func TestPsrAutoloadingSkips(t *testing.T) {
	t.Parallel()
	f := PsrAutoloading{}

	// no backing path (e.g. string input) is left untouched
	if _, changed := apply(t, f, "<?php\nclass Bar {}\n"); changed {
		t.Fatal("empty path must be a no-op")
	}

	// non-php extension is skipped
	if _, changed := applyAt(t, f, "/src/Foo.inc", "<?php\nclass Bar {}\n"); changed {
		t.Fatal("non-php file must be skipped")
	}

	// stub/fixture paths are skipped
	if _, changed := applyAt(t, f, "/project/tests/Fixture/Foo.php", "<?php\nclass Bar {}\n"); changed {
		t.Fatal("fixture path must be skipped")
	}

	// a second classy token (here a ::class constant is reclassified away, so it
	// does not count) still renames; but a real second class bails out
	if _, changed := applyAt(t, f, "/src/Foo.php", "<?php\nclass Bar {}\nclass Baz {}\n"); changed {
		t.Fatal("two named classes must bail out")
	}

	// a ::class constant is not a classy declaration and must not block the rename
	if got, changed := applyAt(t, f, "/src/Foo.php", "<?php\nclass Bar {\n    const C = Other::class;\n}\n"); !changed || got != "<?php\nclass Foo {\n    const C = Other::class;\n}\n" {
		t.Fatalf("::class constant: changed=%v got=%q", changed, got)
	}

	// an anonymous class is skipped; the named class is still renamed
	if got, changed := applyAt(t, f, "/src/Foo.php", "<?php\nclass Bar {}\n$x = new class {};\n"); !changed || got != "<?php\nclass Foo {}\n$x = new class {};\n" {
		t.Fatalf("anonymous class: changed=%v got=%q", changed, got)
	}

	// a filename that is not a valid identifier is skipped
	if _, changed := applyAt(t, f, "/src/foo-bar.php", "<?php\nclass Bar {}\n"); changed {
		t.Fatal("invalid identifier filename must be skipped")
	}
}

func TestPsrAutoloadingDir(t *testing.T) {
	t.Parallel()
	f := PsrAutoloading{}.WithConfig(map[string]any{"dir": "/project/src"}).(fixerRule)

	// a file outside the configured dir is skipped
	if _, changed := applyAt(t, f, "/other/Foo.php", "<?php\nclass Bar {}\n"); changed {
		t.Fatal("file outside dir must be skipped")
	}

	// namespace casing is corrected to the path when it differs only by case
	got, changed := applyAt(t, f, "/project/src/App/Sub/Foo.php", "<?php\nnamespace App\\sub;\nclass Foo {}\n")
	if want := "<?php\nnamespace App\\Sub;\nclass Foo {}\n"; !changed || got != want {
		t.Fatalf("namespace case: changed=%v got=%q", changed, got)
	}
}
