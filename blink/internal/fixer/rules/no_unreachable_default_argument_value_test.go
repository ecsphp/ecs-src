package rules

import "testing"

func TestNoUnreachableDefaultArgumentValue(t *testing.T) {
	cases := []struct {
		src     string
		want    string
		changed bool
	}{
		{`<?php function example($foo = "two words", $bar) {}`, `<?php function example($foo, $bar) {}`, true},
		{`<?php function f($a=1, $b=2, $c) {}`, `<?php function f($a, $b, $c) {}`, true},
		{`<?php $g = fn($a = 1, $b) => $a;`, `<?php $g = fn($a, $b) => $a;`, true},
		{`<?php function f(int $a = 1, ?int $b = null, $c) {}`, `<?php function f(int $a, ?int $b, $c) {}`, true},
		{`<?php function f($a = [1, 2], $b) {}`, `<?php function f($a, $b) {}`, true},

		// "= null" on a non-nullable typed argument is kept
		{`<?php function f(int $a = null, $b) {}`, `<?php function f(int $a = null, $b) {}`, false},
		// variadic after a default: nothing to remove
		{`<?php function f($a = 1, ...$rest) {}`, `<?php function f($a = 1, ...$rest) {}`, false},
		// trailing optional arg is legal: no-op
		{`<?php function ok($a, $b = 1) {}`, `<?php function ok($a, $b = 1) {}`, false},
		// no defaults at all: no-op
		{`<?php function g($a, $b) {}`, `<?php function g($a, $b) {}`, false},
	}
	for _, c := range cases {
		got, changed := apply(t, NoUnreachableDefaultArgumentValue{}, c.src)
		if got != c.want || changed != c.changed {
			t.Fatalf("src=%q changed=%v got=%q want=%q", c.src, changed, got, c.want)
		}
		if again, changed2 := apply(t, NoUnreachableDefaultArgumentValue{}, got); changed2 || again != got {
			t.Fatalf("not idempotent: %q -> %q changed=%v", got, again, changed2)
		}
	}
}
