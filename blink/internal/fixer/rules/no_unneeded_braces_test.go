package rules

import "testing"

func TestNoUnneededBraces(t *testing.T) {
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		// standalone block after open tag
		{"<?php {\n    echo 1;\n}", "<?php \n    echo 1;\n", true},
		// standalone block after ";"
		{`<?php $a=1; { $b=2; } $c=3;`, `<?php $a=1;  $b=2;  $c=3;`, true},
		// block after case colon
		{"<?php switch ($b) {\n    case 1: {\n        break;\n    }\n}", "<?php switch ($b) {\n    case 1: \n        break;\n    \n}", true},
		// single-element group import
		{`<?php use Foo\{Bar};`, `<?php use Foo\Bar;`, true},

		// control body: kept
		{`<?php if ($x) { echo 1; }`, `<?php if ($x) { echo 1; }`, false},
		// function body: kept
		{`<?php function f() { echo 1; }`, `<?php function f() { echo 1; }`, false},
		// bracketed namespace: kept (namespaces option off)
		{`<?php namespace Foo { function Bar(){} }`, `<?php namespace Foo { function Bar(){} }`, false},
		// multi-element group import: kept
		{`<?php use Foo\{Bar, Baz};`, `<?php use Foo\{Bar, Baz};`, false},
	}
	for _, c := range cases {
		got, changed := apply(t, NoUnneededBraces{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, changed2 := apply(t, NoUnneededBraces{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}
