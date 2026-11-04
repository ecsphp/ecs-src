package rules

import "testing"

// TestStaticLambdaEngine holds cases checked against the PHP engine's output.
func TestStaticLambdaEngine(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"<?php $a = function () { return \"x $this->y\"; };\n", "<?php $a = function () { return \"x $this->y\"; };\n"},
		{"<?php $a = function () { return \"x \\$this\"; };\n", "<?php $a = static function () { return \"x \\$this\"; };\n"},
		{"<?php $a = function () { return 'x $this'; };\n", "<?php $a = static function () { return 'x $this'; };\n"},
		{"<?php $a = function () { return <<<TXT\n  $this->y\n  TXT;\n};\n", "<?php $a = function () { return <<<TXT\n  $this->y\n  TXT;\n};\n"},
		{"<?php $a = function () { return <<<'TXT'\n  $this->y\n  TXT;\n};\n", "<?php $a = static function () { return <<<'TXT'\n  $this->y\n  TXT;\n};\n"},
		{"<?php $a = fn () => $b->parent;\n", "<?php $a = fn () => $b->parent;\n"},
		{"<?php $a = fn () => $$b;\n", "<?php $a = fn () => $$b;\n"},
		{"<?php $a = fn () => ${$b};\n", "<?php $a = fn () => ${$b};\n"},
		{"<?php $a = fn () => $b->eval();\n", "<?php $a = static fn () => $b->eval();\n"},
		{"<?php $a = fn () => foo(class: 1);\n", "<?php $a = static fn () => foo(class: 1);\n"},
		{"<?php $a = fn () => new class ($this) {};\n", "<?php $a = fn () => new class ($this) {};\n"},
		{"<?php $a = fn () => new class { function f() { return $this; } };\n", "<?php $a = static fn () => new class { function f() { return $this; } };\n"},
		{"<?php $a = (function () {})->call($b);\n", "<?php $a = (function () {})->call($b);\n"},
		{"<?php $a = foo(function () {})->call($b);\n", "<?php $a = foo(static function () {})->call($b);\n"},
		// a short-array close before the wrapping parentheses is not an index close
		{"<?php\n$a = [$o, \"m\"](function () {})->call($b);\n", "<?php\n$a = [$o, \"m\"](function () {})->call($b);\n"},
		{"<?php\n$a = $x[\"m\"](function () {})->call($b);\n", "<?php\n$a = $x[\"m\"](static function () {})->call($b);\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, StaticLambda{}, c.input); got != c.want {
			t.Errorf("input:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.want)
		}
	}
}
