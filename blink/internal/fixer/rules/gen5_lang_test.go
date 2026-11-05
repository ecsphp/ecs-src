package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

func TestGen5LangInclude(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		// empty braced body -> semicolon (upstream default "semicolon")
		{`<?php while ($x){}`, `<?php while ($x);`, true},
		{`<?php for ($i=0;$i<3;$i++){}`, `<?php for ($i=0;$i<3;$i++);`, true},
		{`<?php foreach ($a as $b){}`, `<?php foreach ($a as $b);`, true},
		// whitespace-only body still counts as empty; its whitespace stays when
		// nothing follows the "}" to merge it with (clearTokenAndMergeSurroundingWhitespace)
		{`<?php while ($x){ }`, `<?php while ($x); `, true},
		{"<?php while ($x){\n}\n$y = 1;", "<?php while ($x);\n\n$y = 1;", true},

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

// TestGen5LangSourceURLs verifies each SourceURL matches SourceURLFor(Name()).
func TestGen5LangSourceURLs(t *testing.T) {
	t.Parallel()
	for _, f := range []fixer.Fixer{Include{}, EmptyLoopBody{}, EmptyLoopCondition{}} {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}

// A file whose loops need no conversion must come back untouched, its stream
// version included: the tags are folded only when a loop is converted.
func TestEmptyLoopConditionLeavesOtherLoopsUntouched(t *testing.T) {
	t.Parallel()
	const src = "<?php\n\n\nfor ($i = 0; $i < 3; $i++) {\n}\nwhile ($x) {\n}\n?>\n<p>x</p>\n<?php while ($y) {} ?>\n"
	for name, f := range map[string]EmptyLoopCondition{
		"while": {},
		"for":   {For: true},
	} {
		src := src
		if !f.For {
			src += "<?php while (true) {}\n" // already in the while style
		}
		s := tokens.New(lexer.Lex(src))
		before := s.Version()
		if f.Fix(s) {
			t.Errorf("%s: reported a change", name)
		}
		if s.Version() != before {
			t.Errorf("%s: stream version moved from %d to %d", name, before, s.Version())
		}
		if got := s.Render(); got != src {
			t.Errorf("%s: got %q", name, got)
		}
	}
}
