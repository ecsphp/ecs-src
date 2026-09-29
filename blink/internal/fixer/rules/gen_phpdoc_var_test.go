package rules

import (
	"testing"

	"blink/internal/fixer"
)

// assertFix runs r once and checks the output and changed flag.
func assertFix(t *testing.T, r fixerRule, src, want string, wantChanged bool) {
	t.Helper()
	got, changed := apply(t, r, src)
	if changed != wantChanged || got != want {
		t.Fatalf("changed=%v (want %v)\n got: %q\nwant: %q", changed, wantChanged, got, want)
	}
	// idempotent: a second pass over the output must be a no-op
	if got2, changed2 := apply(t, r, got); changed2 || got2 != got {
		t.Fatalf("not idempotent: changed=%v got=%q", changed2, got2)
	}
}

func TestGenPhpdocVarWithoutName(t *testing.T) {
	f := PhpdocVarWithoutName{}

	// single-line property docblock
	assertFix(t, f,
		"<?php\nclass A{\n/** @var int $bar */\npublic $bar;\n}",
		"<?php\nclass A{\n/** @var int */\npublic $bar;\n}", true)

	// multi-line property docblock
	assertFix(t, f,
		"<?php\nclass A{\n/**\n * @var int $bar\n */\npublic $bar;\n}",
		"<?php\nclass A{\n/**\n * @var int\n */\npublic $bar;\n}", true)

	// @type tag
	assertFix(t, f,
		"<?php\nclass A{\n/** @type float $baz */\npublic $baz;\n}",
		"<?php\nclass A{\n/** @type float */\npublic $baz;\n}", true)

	// readonly + typed property
	assertFix(t, f,
		"<?php\nclass A{\n/** @var int $y */\npublic readonly int $y;\n}",
		"<?php\nclass A{\n/** @var int */\npublic readonly int $y;\n}", true)

	// "static public" property
	assertFix(t, f,
		"<?php\nclass A{\n/** @var int $y */\nstatic public $y;\n}",
		"<?php\nclass A{\n/** @var int */\nstatic public $y;\n}", true)

	// already without a name - no-op
	assertFix(t, f,
		"<?php\nclass A{\n/** @var int */\npublic $bar;\n}",
		"<?php\nclass A{\n/** @var int */\npublic $bar;\n}", false)

	// $this must be preserved
	assertFix(t, f,
		"<?php\nclass A{\n/** @var static $this */\npublic $bar;\n}",
		"<?php\nclass A{\n/** @var static $this */\npublic $bar;\n}", false)

	// inline variable (not a property) - untouched
	assertFix(t, f,
		"<?php\n/** @var int $x */\n$x = 1;",
		"<?php\n/** @var int $x */\n$x = 1;", false)

	// array-shape braces - conservatively skipped
	assertFix(t, f,
		"<?php\nclass A{\n/**\n * @var array{a: int} $bar\n */\npublic $bar;\n}",
		"<?php\nclass A{\n/**\n * @var array{a: int} $bar\n */\npublic $bar;\n}", false)
}

func TestGenPhpdocVarSourceURLs(t *testing.T) {
	pv := PhpdocVarWithoutName{}
	if got, want := pv.SourceURL(), fixer.SourceURLFor(pv.Name()); got != want {
		t.Fatalf("PhpdocVarWithoutName SourceURL %q, want %q", got, want)
	}
}
