package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen2OpsAssignNullCoalescing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src, want string
		changed   bool
	}{
		{"<?php $a = $a ?? $b;", "<?php $a ??= $b;", true},
		{"<?php $a = $a ?? 1;", "<?php $a ??= 1;", true},
		{"<?php $a = $a ?? 'x';", "<?php $a ??= 'x';", true},
		{"<?php $a=$a??$b;", "<?php $a??=$b;", true},
		// already shorthand: no-op
		{"<?php $a ??= $b;", "<?php $a ??= $b;", false},
		// different variable on the right
		{"<?php $a = $b ?? $c;", "<?php $a = $b ?? $c;", false},
		// chained right side (precedence risk) must not collapse
		{"<?php $a = $a ?? $b ?? $c;", "<?php $a = $a ?? $b ?? $c;", false},
		// right operand is not self-contained
		{"<?php $a = $a ?? foo();", "<?php $a = $a ?? foo();", false},
		{"<?php $a = $a ?? $b[0];", "<?php $a = $a ?? $b[0];", false},
		// complex lvalue is out of scope
		{"<?php $o->x = $o->x ?? $b;", "<?php $o->x = $o->x ?? $b;", false},
		// plain assignment, no coalescing
		{"<?php $a = $a + 1;", "<?php $a = $a + 1;", false},
	}
	for _, c := range cases {
		got, changed := apply(t, AssignNullCoalescingToCoalesceEqual{}, c.src)
		if got != c.want || changed != c.changed {
			t.Errorf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
	}
}

// TestGen2OpsSourceURLs pins each fixer's SourceURL to the canonical form
// derived from its Name. The blob URL was verified to return HTTP 200.
func TestGen2OpsSourceURLs(t *testing.T) {
	t.Parallel()
	fixers := []fixer.Fixer{
		AssignNullCoalescingToCoalesceEqual{},
	}
	for _, f := range fixers {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL=%q want=%q", f.Name(), got, want)
		}
	}
}

// TestGen2OpsIdempotent guarantees a second pass is a no-op for each fixer.
func TestGen2OpsIdempotent(t *testing.T) {
	t.Parallel()
	fixers := []fixer.Fixer{
		AssignNullCoalescingToCoalesceEqual{},
	}
	corpus := []string{
		"<?php $a = $a ?? $b;",
		"<?php $a = $a ?? 1;\n$c = $c ?? 'x';\n",
		"<?php $a = $a ?? $b ?? $c;\n$o->x = $o->x ?? $b;\n",
	}
	for _, f := range fixers {
		for _, src := range corpus {
			once, _ := apply(t, f, src)
			twice, changed := apply(t, f, once)
			if changed || once != twice {
				t.Errorf("%s not idempotent: src=%q once=%q twice=%q", f.Name(), src, once, twice)
			}
		}
	}
}
