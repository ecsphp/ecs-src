package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGenControlSwitchContinueToBreak(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		// bare continue directly in a switch case -> break
		{
			"<?php switch ($a) { case 1: continue; }",
			"<?php switch ($a) { case 1: break; }",
			true,
		},
		// "continue 1;" keeps the level, only the keyword changes
		{
			"<?php switch ($a) { case 1: continue 1; }",
			"<?php switch ($a) { case 1: break 1; }",
			true,
		},
		// "continue 0;" behaves like "continue 1;"
		{
			"<?php switch ($a) { case 1: continue 0; }",
			"<?php switch ($a) { case 1: break 0; }",
			true,
		},
		// continue inside an if within a switch still targets the switch
		{
			"<?php switch ($a) { case 1: if ($b) { continue; } }",
			"<?php switch ($a) { case 1: if ($b) { break; } }",
			true,
		},
		// switch nested in a loop: the continue targets the switch
		{
			"<?php while ($x) { switch ($a) { case 1: continue; } }",
			"<?php while ($x) { switch ($a) { case 1: break; } }",
			true,
		},
		// continue 2 that targets an enclosing switch -> break 2
		{
			"<?php switch ($a) { case 1: foreach ($b as $c) { continue 2; } }",
			"<?php switch ($a) { case 1: foreach ($b as $c) { break 2; } }",
			true,
		},

		// loop directly encloses the continue: must stay a continue
		{
			"<?php foreach ($a as $b) { continue; }",
			"<?php foreach ($a as $b) { continue; }",
			false,
		},
		// loop nested in a switch: continue targets the loop, left alone
		{
			"<?php switch ($a) { case 1: foreach ($b as $c) { continue; } }",
			"<?php switch ($a) { case 1: foreach ($b as $c) { continue; } }",
			false,
		},
		// do-while nested in a switch: continue targets the do loop
		{
			"<?php switch ($a) { case 1: do { continue; } while ($x); }",
			"<?php switch ($a) { case 1: do { continue; } while ($x); }",
			false,
		},
		// continue 2 targeting an enclosing loop (not the switch) is left alone
		{
			"<?php foreach ($a as $b) { switch ($c) { case 1: continue 2; } }",
			"<?php foreach ($a as $b) { switch ($c) { case 1: continue 2; } }",
			false,
		},
		// a method named continue is not a control keyword
		{
			"<?php switch ($a) { case 1: $o->continue(); break; }",
			"<?php switch ($a) { case 1: $o->continue(); break; }",
			false,
		},
		// no switch anywhere: plain loop continue untouched
		{
			"<?php for ($i = 0; $i < 3; $i++) { continue; }",
			"<?php for ($i = 0; $i < 3; $i++) { continue; }",
			false,
		},
		// alternative (brace-less) switch syntax is not touched
		{
			"<?php switch ($a): case 1: continue; endswitch;",
			"<?php switch ($a): case 1: continue; endswitch;",
			false,
		},
	}
	for _, c := range cases {
		got, changed := apply(t, SwitchContinueToBreak{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		// idempotent: a second pass never changes a fixed result
		if again, changed2 := apply(t, SwitchContinueToBreak{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}

// TestGenControlSourceURLs verifies each SourceURL matches SourceURLFor(Name()).
func TestGenControlSourceURLs(t *testing.T) {
	t.Parallel()
	for _, f := range []fixer.Fixer{SwitchContinueToBreak{}} {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}
