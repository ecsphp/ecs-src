package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestTrailingCommaDestructuring(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"[\n    $a,\n    $b\n] = foo();", "[\n    $a,\n    $b\n] = foo();"},
		{"[\n    $a,\n    $b,\n] = foo();", "[\n    $a,\n    $b,\n] = foo();"},
		{"['x' => $a, 'y' => $b] = foo();", "['x' => $a, 'y' => $b] = foo();"},
		{"[\n    'x' => $a,\n    'y' => $b\n] = foo();", "[\n    'x' => $a,\n    'y' => $b\n] = foo();"},
		{"list(\n    $a,\n    $b\n) = foo();", "list(\n    $a,\n    $b\n) = foo();"},
		{"foreach ($g as [\n    $a,\n    $b\n]) {\n}", "foreach ($g as [\n    $a,\n    $b\n]) {\n}"},
		{"[\n    [$a, $b],\n    $c\n] = foo();", "[\n    [$a, $b],\n    $c\n] = foo();"},
		{"$x = [\n    1,\n    2\n];", "$x = [\n    1,\n    2,\n];"},
		{"foo([\n    1,\n    2\n]);", "foo([\n    1,\n    2,\n]);"},
		{"$a[\n    0\n] = 1;", "$a[\n    0\n] = 1;"},
		{"foo(\n    1,\n    2\n);", "foo(\n    1,\n    2\n);"},
		{"[\n    'k' => [\n        $a,\n        $b\n    ],\n    'l' => $c\n] = foo();", "[\n    'k' => [\n        $a,\n        $b,\n    ],\n    'l' => $c\n] = foo();"},
		{"foreach ($g as ['a' => [\n    $b,\n    $c\n]]) {\n}", "foreach ($g as ['a' => [\n    $b,\n    $c,\n]]) {\n}"},
		{"$f = fn ($x) => [\n    $a,\n    $b\n] = $x;", "$f = fn ($x) => [\n    $a,\n    $b,\n] = $x;"},
		{"foreach ($g as $k => [\n    $a,\n    $b\n]) {\n}", "foreach ($g as $k => [\n    $a,\n    $b\n]) {\n}"},
		{"$x = ['k' => [\n    1,\n    2\n]];", "$x = ['k' => [\n    1,\n    2,\n]];"},
		{"['k' => [[\n    $a\n]]] = foo();", "['k' => [[\n    $a,\n]]] = foo();"},
		{"$f = fn ($x) => [\n    $a,\n    [\n        $b\n    ],\n] = $x;", "$f = fn ($x) => [\n    $a,\n    [\n        $b,\n    ],\n] = $x;"},
		{"foreach ($g as $this->k => [\n    $a\n]) {\n}", "foreach ($g as $this->k => [\n    $a,\n]) {\n}"},
		{"foreach ($g as $m['x'] => [\n    $a\n]) {\n}", "foreach ($g as $m['x'] => [\n    $a,\n]) {\n}"},
		{"[$x . [\n    1\n][0] => $a] = foo();", "[$x . [\n    1,\n][0] => $a] = foo();"},
		{"$x = ['k' => [[\n    $a\n]] = foo()];", "$x = ['k' => [[\n    $a,\n]] = foo()];"},
	}
	for _, c := range cases {
		got, changed := apply(t, TrailingCommaInMultiline{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
