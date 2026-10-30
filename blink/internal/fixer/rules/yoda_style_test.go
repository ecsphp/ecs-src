package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestYodaStyle(t *testing.T) {
	t.Parallel()
	f := YodaStyle{}

	// constant on the left moves to the right (non-yoda, ECS common config)
	assertFix(t, f, "<?php\nif (null === $x) {}", "<?php\nif ($x === null) {}", true)
	assertFix(t, f, "<?php\nif ('' === $value) {}", "<?php\nif ($value === '') {}", true)
	assertFix(t, f, "<?php\nif (false === $x) {}", "<?php\nif ($x === false) {}", true)
	assertFix(t, f, "<?php\nif (null === static::$r) {}", "<?php\nif (static::$r === null) {}", true)
	assertFix(t, f, "<?php\nif (null !== $this->pending) {}", "<?php\nif ($this->pending !== null) {}", true)
	assertFix(t, f, "<?php\nif (true === $app->resolved($a)) {}", "<?php\nif ($app->resolved($a) === true) {}", true)
	// right side is a parenthesized group that is the base of a larger primary,
	// e.g. "(new X())->y()": still a variable expression, so it is de-yoda'd
	assertFix(t, f, "<?php\nif (null !== (new static($a))->user()) {}", "<?php\nif ((new static($a))->user() !== null) {}", true)

	// constant name, ::class, signed number, empty array on the left
	assertFix(t, f, "<?php\nif (JSON_ERROR_NONE === json_last_error()) {}", "<?php\nif (json_last_error() === JSON_ERROR_NONE) {}", true)
	// namespaced/global constant with a leading backslash on the left
	assertFix(t, f, "<?php\nif (\\INF === $beta) {}", "<?php\nif ($beta === \\INF) {}", true)
	assertFix(t, f, "<?php\nif (\\E_USER_DEPRECATED !== $type) {}", "<?php\nif ($type !== \\E_USER_DEPRECATED) {}", true)
	// embedded in a larger boolean expression
	assertFix(t, f, "<?php\nif (\\E_DEPRECATED !== $type && $ok) {}", "<?php\nif ($type !== \\E_DEPRECATED && $ok) {}", true)
	// leading-backslash constant against a namespaced call on the right
	assertFix(t, f, "<?php\nif (\\PREG_NO_ERROR !== preg_last_error()) {}", "<?php\nif (preg_last_error() !== \\PREG_NO_ERROR) {}", true)
	// class constant on the left after a compound assignment
	assertFix(t, f, "<?php\n$n += Events::RESPONSE === $tag ? 1 : 0;", "<?php\n$n += $tag === Events::RESPONSE ? 1 : 0;", true)
	assertFix(t, f, "<?php\nif (Closure::class === $l) {}", "<?php\nif ($l === Closure::class) {}", true)
	assertFix(t, f, "<?php\nif (-1 === $i) {}", "<?php\nif ($i === -1) {}", true)
	assertFix(t, f, "<?php\nif ([] === $seq) {}", "<?php\nif ($seq === []) {}", true)

	// < > <= >= are de-yoda'd with the operator flipped
	assertFix(t, f, "<?php\nif (1000 > $x) {}", "<?php\nif ($x < 1000) {}", true)
	assertFix(t, f, "<?php\n$a = 5 >= $y;", "<?php\n$a = $y <= 5;", true)
	// match-arm condition (=> boundary) is de-yoda'd too
	assertFix(t, f, "<?php\n$r = match (true) {\n    null !== $x => 1,\n};", "<?php\n$r = match (true) {\n    $x !== null => 1,\n};", true)

	// comparison that is a multiline ternary condition starting right after a
	// preceding block's "}" - the "}" is a left boundary, so it is de-yoda'd
	assertFix(t, f,
		"<?php\nif ($a) {\n    $b = 1;\n}\n\nnull === $delay\n    ? x()\n    : y($delay);",
		"<?php\nif ($a) {\n    $b = 1;\n}\n\n$delay === null\n    ? x()\n    : y($delay);",
		true)

	// already non-yoda - no-op
	assertFix(t, f, "<?php\nif ($x === null) {}", "<?php\nif ($x === null) {}", false)

	// both sides variables - no-op
	assertFix(t, f, "<?php\nif ($a === $b) {}", "<?php\nif ($a === $b) {}", false)

	// parenthesized left with the constant already on the right - no-op
	assertFix(t, f, "<?php\nreturn ($this->html ?? '') === '';", "<?php\nreturn ($this->html ?? '') === '';", false)

	// a yoda nested inside another yoda's operand: both are de-yoda'd, innermost first
	assertFix(t, f,
		"<?php\n$b = [] !== array_filter($e, fn ($c) => 1 === preg_match('/x/', $c));",
		"<?php\n$b = array_filter($e, fn ($c) => preg_match('/x/', $c) === 1) !== [];",
		true)
}

func TestYodaStyleSourceURL(t *testing.T) {
	t.Parallel()
	f := YodaStyle{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Fatalf("SourceURL %q, want %q", got, want)
	}
}
