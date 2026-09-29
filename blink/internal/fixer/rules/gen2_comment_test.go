package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen2Comment(t *testing.T) {
	// misaligned docblock at column 0 -> aligned under the first "*"
	got, changed := apply(t, AlignMultilineComment{}, "<?php\n/**\n     * Foo\n   */\n")
	if want := "<?php\n/**\n * Foo\n */\n"; !changed || got != want {
		t.Fatalf("col0: changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, AlignMultilineComment{}, got)

	// misaligned docblock indented four spaces keeps that indent
	got, changed = apply(t, AlignMultilineComment{}, "<?php\n    /**\n         * Foo\n      */\n")
	if want := "<?php\n    /**\n     * Foo\n     */\n"; !changed || got != want {
		t.Fatalf("indented: changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, AlignMultilineComment{}, got)

	// already aligned is a no-op
	if _, changed := apply(t, AlignMultilineComment{}, "<?php\n/**\n * Foo\n */\n"); changed {
		t.Fatal("already-aligned docblock must not change")
	}

	// a comment whose content lines do not start with "*" is left alone
	src := "<?php\n/*\n    free form\n    another\n */\n"
	if got, changed := apply(t, AlignMultilineComment{}, src); changed || got != src {
		t.Fatalf("free-form comment must be left alone: changed=%v got=%q", changed, got)
	}

	// tab indentation is preserved
	got, changed = apply(t, AlignMultilineComment{}, "<?php\n\t/**\n\t\t\t* Foo\n\t */\n")
	if want := "<?php\n\t/**\n\t * Foo\n\t */\n"; !changed || got != want {
		t.Fatalf("tab: changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, AlignMultilineComment{}, got)

	// plain block comment (not a docblock) is realigned too
	got, changed = apply(t, AlignMultilineComment{}, "<?php\n/*\n     * Foo\n   */\n")
	if want := "<?php\n/*\n * Foo\n */\n"; !changed || got != want {
		t.Fatalf("plain: changed=%v got=%q want=%q", changed, got, want)
	}

	// an inline comment (not preceded by a line break) is left alone
	if _, changed := apply(t, AlignMultilineComment{}, "<?php $x = 1; /**\n     * Foo\n */"); changed {
		t.Fatal("inline comment must not be re-indented")
	}
}

// TestGen2CommentSourceURL verifies the SourceURL matches SourceURLFor(Name()).
func TestGen2CommentSourceURL(t *testing.T) {
	var f fixer.Fixer = AlignMultilineComment{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
	}
}
