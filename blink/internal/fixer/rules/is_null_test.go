package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestIsNull(t *testing.T) {
	f := IsNull{}

	assertFix(t, f, "<?php\nif (is_null($x)) {}", "<?php\nif (null === $x) {}", true)
	assertFix(t, f, "<?php\nif (! is_null($x)) {}", "<?php\nif (null !== $x) {}", true)
	assertFix(t, f, "<?php\n$y = is_null($a->b());", "<?php\n$y = null === $a->b();", true)
	assertFix(t, f, "<?php\n$y = is_null($a['k']);", "<?php\n$y = null === $a['k'];", true)

	// argument with a top-level operator keeps its parentheses
	assertFix(t, f, "<?php\nif (! is_null($h = f())) {}", "<?php\nif (null !== ($h = f())) {}", true)

	// method/static call of the same name is left alone
	assertFix(t, f, "<?php\n$o->is_null($x);", "<?php\n$o->is_null($x);", false)
	assertFix(t, f, "<?php\nFoo::is_null($x);", "<?php\nFoo::is_null($x);", false)

	// no is_null present
	assertFix(t, f, "<?php\n$x = 1;", "<?php\n$x = 1;", false)
}

func TestIsNullSourceURL(t *testing.T) {
	f := IsNull{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Fatalf("SourceURL %q, want %q", got, want)
	}
}
