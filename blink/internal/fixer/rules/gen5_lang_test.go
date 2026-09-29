package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen5LangInclude(t *testing.T) {
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		// collapse multiple spaces to one
		{`<?php require_once  "x.php";`, `<?php require_once "x.php";`, true},
		// collapse tab to a single space
		{"<?php include\t\"x.php\";", `<?php include "x.php";`, true},
		// insert a space when the argument is directly adjacent
		{`<?php include"x.php";`, `<?php include "x.php";`, true},
		{`<?php require$file;`, `<?php require $file;`, true},

		// already a single space: no-op
		{`<?php require "x.php";`, `<?php require "x.php";`, false},
		// paren form is left untouched (parens not removed)
		{`<?php include("x.php");`, `<?php include("x.php");`, false},
		{`<?php include ("x.php");`, `<?php include ("x.php");`, false},
		// multi-line spacing (alignment) is kept
		{"<?php include\n\t\"x.php\";", "<?php include\n\t\"x.php\";", false},
		// a method named include is not the language construct
		{`<?php $o->include ("x");`, `<?php $o->include ("x");`, false},
	}
	for _, c := range cases {
		got, changed := apply(t, Include{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, changed2 := apply(t, Include{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}

func TestGen5LangEmptyLoopBody(t *testing.T) {
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		// empty braced body -> semicolon (upstream default "semicolon")
		{`<?php while ($x){}`, `<?php while ($x);`, true},
		{`<?php for ($i=0;$i<3;$i++){}`, `<?php for ($i=0;$i<3;$i++);`, true},
		{`<?php foreach ($a as $b){}`, `<?php foreach ($a as $b);`, true},
		// whitespace-only body still counts as empty
		{`<?php while ($x){ }`, `<?php while ($x);`, true},

		// non-empty body: untouched
		{`<?php while ($x){ f(); }`, `<?php while ($x){ f(); }`, false},
		// already a semicolon body: no-op
		{`<?php while ($x);`, `<?php while ($x);`, false},
		// do-while: the "while (...)" is followed by ";", not "{"
		{`<?php do {} while ($x);`, `<?php do {} while ($x);`, false},
		// a comment inside the body is preserved (not empty)
		{`<?php while ($x){ /* c */ }`, `<?php while ($x){ /* c */ }`, false},
		// alternative syntax is not touched
		{`<?php while ($x): endwhile;`, `<?php while ($x): endwhile;`, false},
		// method named while is not a loop
		{`<?php $o->while();`, `<?php $o->while();`, false},
	}
	for _, c := range cases {
		got, changed := apply(t, EmptyLoopBody{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, changed2 := apply(t, EmptyLoopBody{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}

func TestGen5LangEmptyLoopCondition(t *testing.T) {
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		// empty for header -> while (true) (upstream default "while")
		{`<?php for (;;) {}`, `<?php while (true) {}`, true},
		{`<?php for(;;){ foo(); }`, `<?php while (true){ foo(); }`, true},
		// interior whitespace in the header is normalized away
		{`<?php for ( ; ; ) { foo(); }`, `<?php while (true) { foo(); }`, true},

		// non-empty condition: untouched
		{`<?php for ($i=0;$i<3;$i++) {}`, `<?php for ($i=0;$i<3;$i++) {}`, false},
		// only one part empty: untouched
		{`<?php for (;$i<3;) {}`, `<?php for (;$i<3;) {}`, false},
		// already while (true): left alone (idempotent, do-while not handled)
		{`<?php while (true) { foo(); }`, `<?php while (true) { foo(); }`, false},
		// comment inside the header: skipped to avoid reordering it
		{`<?php for (/* a */;;) {}`, `<?php for (/* a */;;) {}`, false},
		// method named for is not a loop
		{`<?php $o->for(;;);`, `<?php $o->for(;;);`, false},
	}
	for _, c := range cases {
		got, changed := apply(t, EmptyLoopCondition{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, changed2 := apply(t, EmptyLoopCondition{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}

// TestGen5LangSourceURLs verifies each SourceURL matches SourceURLFor(Name()).
func TestGen5LangSourceURLs(t *testing.T) {
	for _, f := range []fixer.Fixer{Include{}, EmptyLoopBody{}, EmptyLoopCondition{}} {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}
