package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGenOpsCastNoShortBoolCast(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		{"<?php $x = !!$b;", "<?php $x = (bool) $b;", true},
		{"<?php $x = !! $b;", "<?php $x = (bool) $b;", true},
		{"<?php $x = ! !$b;", "<?php $x = (bool) $b;", true},
		{"<?php $x = !!!$a;", "<?php $x = !(bool) $a;", true},
		// single negation is left alone
		{"<?php $x = !$b;", "<?php $x = !$b;", false},
		// not-equal operators must not be mistaken for double not
		{"<?php $a != $b;", "<?php $a != $b;", false},
		{"<?php $a !== $b;", "<?php $a !== $b;", false},
	}
	for _, c := range cases {
		got, changed := apply(t, NoShortBoolCast{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		// idempotent: re-running never changes a fixed result
		if again, changed2 := apply(t, NoShortBoolCast{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}

func TestGenOpsCastNoUnsetCast(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		{"<?php $a = (unset) $b;", "<?php $a = null;", true},
		{"<?php $a = (unset)$b;", "<?php $a = null;", true},
		{"<?php $a=(unset)$b;", "<?php $a= null;", true},
		{"<?php $a = (UNSET) $b;", "<?php $a = null;", true},
		{"<?php $a = (unset) $b ?>", "<?php $a = null ?>", true},
		// only "= (unset) $var ;" is rewritten
		{"<?php $a = (unset) $b + 1;", "<?php $a = (unset) $b + 1;", false},
		{"<?php foo((unset) $b);", "<?php foo((unset) $b);", false},
		{"<?php $a == (unset) $b;", "<?php $a == (unset) $b;", false},
		// unrelated casts stay put
		{"<?php $a = (int) $b;", "<?php $a = (int) $b;", false},
	}
	for _, c := range cases {
		got, changed := apply(t, NoUnsetCast{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, changed2 := apply(t, NoUnsetCast{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}

func TestGenOpsCastSourceURLs(t *testing.T) {
	t.Parallel()
	for _, f := range []fixer.Fixer{NoShortBoolCast{}, NoUnsetCast{}} {
		if want := fixer.SourceURLFor(f.Name()); f.SourceURL() != want {
			t.Errorf("%s: SourceURL %q inconsistent with name, want %q", f.Name(), f.SourceURL(), want)
		}
	}
}
