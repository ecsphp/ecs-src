package rules

import "testing"

// TestVoidReturnEngine holds cases checked against the PHP engine's output.
func TestVoidReturnEngine(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"<?php function f(): (A&B)|null { return; }\n", "<?php function f(): (A&B)|null { return; }\n"},
		{"<?php function f(int $a): A|(B&C) { return; }\n", "<?php function f(int $a): A|(B&C) { return; }\n"},
		{"<?php $f = function () use ($a) { return; };\n", "<?php $f = function () use ($a): void { return; };\n"},
		{"<?php /** @return void|null */ function f() {}\n", "<?php /** @return void|null */ function f() {}\n"},
		{"<?php /** @return Void */ function f() {}\n", "<?php /** @return Void */ function f() {}\n"},
		{"<?php /** @return void Nothing. */ function f() {}\n", "<?php /** @return void Nothing. */ function f(): void {}\n"},
		{"<?php /** @return void[] */ function f() {}\n", "<?php /** @return void[] */ function f() {}\n"},
		{"<?php /** @return */ function f() {}\n", "<?php /** @return */ function f() {}\n"},
		{"<?php interface I { /** @return void */ public function f(); /** @return int */ public function g(); public function h(); }\n", "<?php interface I { /** @return void */ public function f(): void; /** @return int */ public function g(); public function h(); }\n"},
		{"<?php class A { /** @return void */ #[Attr] final public static function f() { return; } }\n", "<?php class A { /** @return void */ #[Attr] final public static function f(): void { return; } }\n"},
		{"<?php function f() { $a = new class { function g() { return 1; } }; $b = fn () => 1; return; }\n", "<?php function f(): void { $a = new class { function g() { return 1; } }; $b = fn () => 1; return; }\n"},
		{"<?php function f() { yield 1; }\n", "<?php function f() { yield 1; }\n"},
		{"<?php function f() { if ($a) { return; } return $b->yield; }\n", "<?php function f() { if ($a) { return; } return $b->yield; }\n"},
		// a number constant starts with a digit, or a "." followed by one
		{"<?php\n/** @return void&...$x */\nfunction f() {}\n", "<?php\n/** @return void&...$x */\nfunction f(): void {}\n"},
		{"<?php\n/** @return void|.5 */\nfunction f() {}\n", "<?php\n/** @return void|.5 */\nfunction f() {}\n"},
		{"<?php\n/** @return void|-.5 */\nfunction f() {}\n", "<?php\n/** @return void|-.5 */\nfunction f() {}\n"},
		{"<?php\n/** @return void|+..5 */\nfunction f() {}\n", "<?php\n/** @return void|+..5 */\nfunction f() {}\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, VoidReturn{}, c.input); got != c.want {
			t.Errorf("input:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.want)
		}
	}
}
