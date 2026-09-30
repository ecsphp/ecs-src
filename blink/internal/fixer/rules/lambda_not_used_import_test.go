package rules

import "testing"

func TestLambdaNotUsedImport(t *testing.T) {
	f := LambdaNotUsedImport{}

	// a single unused import removes the whole use()
	got, changed := apply(t, f, "<?php\n$f = function() use ($bar) {};\n")
	if want := "<?php\n$f = function() {};\n"; !changed || got != want {
		t.Fatalf("all: changed=%v got=%q", changed, got)
	}

	// one unused import among two is removed
	got, changed = apply(t, f, "<?php\n$f = function() use ($a, $b) { return $a; };\n")
	if want := "<?php\n$f = function() use ($a ) { return $a; };\n"; !changed || got != want {
		t.Fatalf("one: changed=%v got=%q", changed, got)
	}

	// both used: unchanged
	if _, changed := apply(t, f, "<?php\n$f = function() use ($a, $b) { return $a + $b; };\n"); changed {
		t.Fatal("used imports must be kept")
	}

	// by-reference import is always kept
	got, changed = apply(t, f, "<?php\n$f = function() use (&$a, $b) { return 1; };\n")
	if want := "<?php\n$f = function() use (&$a ) { return 1; };\n"; !changed || got != want {
		t.Fatalf("ref: changed=%v got=%q", changed, got)
	}

	// compact() bails out - nothing removed
	if _, changed := apply(t, f, "<?php\n$f = function() use ($a) { echo compact(\"a\"); };\n"); changed {
		t.Fatal("compact() must bail out")
	}

	// variable variable bails out
	if _, changed := apply(t, f, "<?php\n$f = function() use ($a) { $$a = 1; };\n"); changed {
		t.Fatal("$$a must bail out")
	}

	// used only inside a "{$var}" interpolation: kept
	if _, changed := apply(t, f, "<?php\n$f = function() use ($attr) { return \"<a {$attr}>\"; };\n"); changed {
		t.Fatal("var used in interpolation must be kept")
	}

	// simple "$var" interpolation counts as usage
	if _, changed := apply(t, f, "<?php\n$f = function() use ($attr) { return \"x $attr y\"; };\n"); changed {
		t.Fatal("var used in simple interpolation must be kept")
	}

	// mautic repro: $attr used only via interpolation, all imports kept
	src := "<?php\n$f = function ($match) use ($protocol, &$links, $attr): string {\n" +
		"    if ($match[1]) { $protocol = $match[1]; }\n" +
		"    return '<'.array_push($links, \"<a {$attr} href=\\\"{$protocol}\\\">\").'>';\n" +
		"};\n"
	if _, changed := apply(t, f, src); changed {
		t.Fatal("mautic case: no import should be removed")
	}

	// single-quoted '$var' does NOT interpolate: import still removed
	got, changed = apply(t, f, "<?php\n$f = function() use ($attr) { return '$attr'; };\n")
	if want := "<?php\n$f = function() { return '$attr'; };\n"; !changed || got != want {
		t.Fatalf("single-quote: changed=%v got=%q", changed, got)
	}
}
