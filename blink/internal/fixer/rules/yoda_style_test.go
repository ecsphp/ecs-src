package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
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

func TestYodaStyleYodaMode(t *testing.T) {
	t.Parallel()
	f := YodaStyle{}.WithConfig(map[string]any{"equal": true, "identical": true, "less_and_greater": nil})

	assertFix(t, f, "<?php\nif ($x === null) {}", "<?php\nif (null === $x) {}", true)
	assertFix(t, f, "<?php\nif ($a->b($c) !== 'x') {}", "<?php\nif ('x' !== $a->b($c)) {}", true)
	assertFix(t, f, "<?php\nif (isset($a) === true) {}", "<?php\nif (true === isset($a)) {}", true)
	// a group as the base of a member access is part of the variable side
	assertFix(t, f, "<?php\nif ((new A())->b() === null) {}", "<?php\nif (null === (new A())->b()) {}", true)
	// less_and_greater=null leaves "<" alone
	assertFix(t, f, "<?php\nif ($b < 3) {}", "<?php\nif ($b < 3) {}", false)

	// a bare parenthesized group is not a variable (always_move_variable=false):
	// the comparison stays as it is, wherever the group starts
	assertFix(t, f, "<?php\nif (($a ?? '') === 'x') {}", "<?php\nif (($a ?? '') === 'x') {}", false)
	assertFix(t, f, "<?php\nif ((($x = f($y))) !== null) {}", "<?php\nif ((($x = f($y))) !== null) {}", false)
	assertFix(t, f, "<?php\nreturn ($a[0] ?? false) === true;", "<?php\nreturn ($a[0] ?? false) === true;", false)
	assertFix(t, f, "<?php\n$v = ($a == $b ?: ($c ?? false) === '1');", "<?php\n$v = ($a == $b ?: ($c ?? false) === '1');", false)
}

// TestYodaStyleMatchesPHPCSFixer covers left sides PHP-CS-Fixer's
// findComparisonStart takes in whole (arithmetic, concatenation, casts, unary
// operators, instanceof, class constants) in every mode; the expected outputs
// come from PHP-CS-Fixer.
func TestYodaStyleMatchesPHPCSFixer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		config   map[string]any
		in, want string
	}{
		{"yoda", map[string]any{"equal": true, "identical": true, "less_and_greater": nil}, "<?php\nif ($a + 1 === 2) {}\nif ($a - 1 === 2) {}\nif ($a * 2 === 4) {}\nif ($a . 'x' === 'y') {}\nif (!$a === true) {}\nif ((int) $a === 1) {}\nif (-$a === 1) {}\nif (@$a === 1) {}\nif ($$a === 1) {}\nif ($a instanceof B === true) {}\nif ($a-- === 0) {}\nif (++$a === 2) {}\nif ($a++ === 2) {}\nif ($this->a === self::X) {}\nif ($a->b === C::D) {}\nif ($a <> null) {}\nif ($x === null) {}\nif ($a->b($c) !== 'x') {}\nif (isset($a) === true) {}\nif ((new A())->b() === null) {}\nif ($b < 3) {}\nif (($a ?? '') === 'x') {}\nif ((($x = f($y))) !== null) {}\nreturn ($a[0] ?? false) === true;\n$v = ($a == $b ?: ($c ?? false) === '1');\n$b = array_filter($e, fn ($c) => preg_match('/x/', $c) === 1) !== [];\n$r = match (true) {\n    $x !== null => 1,\n};\nfoo(a: $x === 1);\n$z = $q === -1 && $w == [];\n$y = $a === 1 ? $b === 2 : $c === 3;\nif ($a === $b) {}\nif ($a->{$b} === 1) {}\nif ($a->{$b}() === 1) {}\nif ($a[1]->b === 1) {}\nif (A::$b === 1) {}\nif (\\A\\B::C === $x) {}\nwhile (null === $x = f()) {}\nyield from $a === 1;\n$k = $a === 1 and $b == 2;\necho $a === '1', $b != 2;\n?>\n<?= $a === 1 ?>\n", "<?php\nif (2 === $a + 1) {}\nif (2 === $a - 1) {}\nif (4 === $a * 2) {}\nif ('y' === $a . 'x') {}\nif (true === !$a) {}\nif (1 === (int) $a) {}\nif (1 === -$a) {}\nif (1 === @$a) {}\nif (1 === $$a) {}\nif (true === $a instanceof B) {}\nif (0 === $a--) {}\nif (2 === ++$a) {}\nif (2 === $a++) {}\nif (self::X === $this->a) {}\nif (C::D === $a->b) {}\nif (null <> $a) {}\nif (null === $x) {}\nif ('x' !== $a->b($c)) {}\nif (true === isset($a)) {}\nif (null === (new A())->b()) {}\nif ($b < 3) {}\nif (($a ?? '') === 'x') {}\nif ((($x = f($y))) !== null) {}\nreturn ($a[0] ?? false) === true;\n$v = ($a == $b ?: ($c ?? false) === '1');\n$b = [] !== array_filter($e, fn ($c) => 1 === preg_match('/x/', $c));\n$r = match (true) {\n    null !== $x => 1,\n};\nfoo(a: 1 === $x);\n$z = -1 === $q && [] == $w;\n$y = 1 === $a ? 2 === $b : 3 === $c;\nif ($a === $b) {}\nif (1 === $a->{$b}) {}\nif ($a->{$b}() === 1) {}\nif (1 === $a[1]->b) {}\nif (1 === A::$b) {}\nif (\\A\\B::C === $x) {}\nwhile (null === $x = f()) {}\nyield from 1 === $a;\n$k = 1 === $a and 2 == $b;\necho '1' === $a, 2 != $b;\n?>\n<?= 1 === $a ?>\n"},
		{"non-yoda", map[string]any{"equal": false, "identical": false, "less_and_greater": false}, "<?php\nif (2 === $a + 1) {}\nif (2 === $a - 1) {}\nif (4 === $a * 2) {}\nif ('y' === $a . 'x') {}\nif (true === !$a) {}\nif (1 === (int) $a) {}\nif (1 === -$a) {}\nif (1 === @$a) {}\nif (1 === $$a) {}\nif (true === $a instanceof B) {}\nif (0 === $a--) {}\nif (2 === ++$a) {}\nif (2 === $a++) {}\nif (self::X === $this->a) {}\nif (C::D === $a->b) {}\nif (null <> $a) {}\nif (null === $x) {}\nif ('x' !== $a->b($c)) {}\nif (true === isset($a)) {}\nif (null === (new A())->b()) {}\nif ($b < 3) {}\nif (($a ?? '') === 'x') {}\nif ((($x = f($y))) !== null) {}\nreturn ($a[0] ?? false) === true;\n$v = ($a == $b ?: ($c ?? false) === '1');\n$b = [] !== array_filter($e, fn ($c) => 1 === preg_match('/x/', $c));\n$r = match (true) {\n    null !== $x => 1,\n};\nfoo(a: 1 === $x);\n$z = -1 === $q && [] == $w;\n$y = 1 === $a ? 2 === $b : 3 === $c;\nif ($a === $b) {}\nif (1 === $a->{$b}) {}\nif ($a->{$b}() === 1) {}\nif (1 === $a[1]->b) {}\nif (1 === A::$b) {}\nif (\\A\\B::C === $x) {}\nwhile (null === $x = f()) {}\nyield from 1 === $a;\n$k = 1 === $a and 2 == $b;\necho '1' === $a, 2 != $b;\n?>\n<?= 1 === $a ?>\n", "<?php\nif ($a + 1 === 2) {}\nif ($a - 1 === 2) {}\nif ($a * 2 === 4) {}\nif ($a . 'x' === 'y') {}\nif (!$a === true) {}\nif ((int) $a === 1) {}\nif (-$a === 1) {}\nif (@$a === 1) {}\nif ($$a === 1) {}\nif ($a instanceof B === true) {}\nif ($a-- === 0) {}\nif (++$a === 2) {}\nif ($a++ === 2) {}\nif ($this->a === self::X) {}\nif ($a->b === C::D) {}\nif ($a <> null) {}\nif ($x === null) {}\nif ($a->b($c) !== 'x') {}\nif (isset($a) === true) {}\nif ((new A())->b() === null) {}\nif ($b < 3) {}\nif (($a ?? '') === 'x') {}\nif ((($x = f($y))) !== null) {}\nreturn ($a[0] ?? false) === true;\n$v = ($a == $b ?: ($c ?? false) === '1');\n$b = array_filter($e, fn ($c) => preg_match('/x/', $c) === 1) !== [];\n$r = match (true) {\n    $x !== null => 1,\n};\nfoo(a: $x === 1);\n$z = $q === -1 && $w == [];\n$y = $a === 1 ? $b === 2 : $c === 3;\nif ($a === $b) {}\nif ($a->{$b} === 1) {}\nif ($a->{$b}() === 1) {}\nif ($a[1]->b === 1) {}\nif (A::$b === 1) {}\nif ($x === \\A\\B::C) {}\nwhile (null === $x = f()) {}\nyield from $a === 1;\n$k = $a === 1 and $b == 2;\necho $a === '1', $b != 2;\n?>\n<?= $a === 1 ?>\n"},
		{"always_move_variable", map[string]any{"equal": true, "identical": true, "less_and_greater": true, "always_move_variable": true}, "<?php\nif ($a + 1 === 2) {}\nif ($a - 1 === 2) {}\nif ($a * 2 === 4) {}\nif ($a . 'x' === 'y') {}\nif (!$a === true) {}\nif ((int) $a === 1) {}\nif (-$a === 1) {}\nif (@$a === 1) {}\nif ($$a === 1) {}\nif ($a instanceof B === true) {}\nif ($a-- === 0) {}\nif (++$a === 2) {}\nif ($a++ === 2) {}\nif ($this->a === self::X) {}\nif ($a->b === C::D) {}\nif ($a <> null) {}\nif ($x === null) {}\nif ($a->b($c) !== 'x') {}\nif (isset($a) === true) {}\nif ((new A())->b() === null) {}\nif ($b < 3) {}\nif (($a ?? '') === 'x') {}\nif ((($x = f($y))) !== null) {}\nreturn ($a[0] ?? false) === true;\n$v = ($a == $b ?: ($c ?? false) === '1');\n$b = array_filter($e, fn ($c) => preg_match('/x/', $c) === 1) !== [];\n$r = match (true) {\n    $x !== null => 1,\n};\nfoo(a: $x === 1);\n$z = $q === -1 && $w == [];\n$y = $a === 1 ? $b === 2 : $c === 3;\nif ($a === $b) {}\nif ($a->{$b} === 1) {}\nif ($a->{$b}() === 1) {}\nif ($a[1]->b === 1) {}\nif (A::$b === 1) {}\nif (\\A\\B::C === $x) {}\nwhile (null === $x = f()) {}\nyield from $a === 1;\n$k = $a === 1 and $b == 2;\necho $a === '1', $b != 2;\n?>\n<?= $a === 1 ?>\n", "<?php\nif ($a + 1 === 2) {}\nif ($a - 1 === 2) {}\nif ($a * 2 === 4) {}\nif ($a . 'x' === 'y') {}\nif (!$a === true) {}\nif ((int) $a === 1) {}\nif (1 === -$a) {}\nif (1 === @$a) {}\nif (1 === $$a) {}\nif ($a instanceof B === true) {}\nif (0 === $a--) {}\nif (2 === ++$a) {}\nif (2 === $a++) {}\nif (self::X === $this->a) {}\nif (C::D === $a->b) {}\nif (null <> $a) {}\nif (null === $x) {}\nif ($a->b($c) !== 'x') {}\nif (true === isset($a)) {}\nif (null === (new A())->b()) {}\nif (3 > $b) {}\nif ('x' === ($a ?? '')) {}\nif (null !== (($x = f($y)))) {}\nreturn true === ($a[0] ?? false);\n$v = ($a == $b ?: '1' === ($c ?? false));\n$b = array_filter($e, fn ($c) => preg_match('/x/', $c) === 1) !== [];\n$r = match (true) {\n    null !== $x => 1,\n};\nfoo(a: 1 === $x);\n$z = -1 === $q && [] == $w;\n$y = 1 === $a ? 2 === $b : 3 === $c;\nif ($a === $b) {}\nif (1 === $a->{$b}) {}\nif ($a->{$b}() === 1) {}\nif (1 === $a[1]->b) {}\nif (1 === A::$b) {}\nif (\\A\\B::C === $x) {}\nwhile (null === $x = f()) {}\nyield from 1 === $a;\n$k = 1 === $a and 2 == $b;\necho '1' === $a, 2 != $b;\n?>\n<?= 1 === $a ?>\n"},
		{"string literals yoda", map[string]any{"equal": true, "identical": true}, "<?php\nif ($a === b'x') {}\nif ($a === B\"x\") {}\nif ($a === <<<EOT\nx\nEOT) {}\nif ($a === <<<'EOT'\nx\nEOT) {}\nif ($a === `ls`) {}\nif ($a === \"x $y\") {}\nif ($a === \"x\") {}\nif ($a->b === <<<EOT\nx $y\nEOT) {}\nfunction g() { return yield   from $a === 1; }\n", "<?php\nif (b'x' === $a) {}\nif (B\"x\" === $a) {}\nif ($a === <<<EOT\nx\nEOT) {}\nif ($a === <<<'EOT'\nx\nEOT) {}\nif ($a === `ls`) {}\nif ($a === \"x $y\") {}\nif (\"x\" === $a) {}\nif ($a->b === <<<EOT\nx $y\nEOT) {}\nfunction g() { return yield   from 1 === $a; }\n"},
		{"string literals non-yoda", map[string]any{"equal": false, "identical": false}, "<?php\nif (b'x' == $a) {}\nif (<<<EOT\nx\nEOT === $a) {}\nif (`ls` === $a) {}\nif (\"x\" === $a) {}\n", "<?php\nif ($a == b'x') {}\nif (<<<EOT\nx\nEOT === $a) {}\nif (`ls` === $a) {}\nif ($a === \"x\") {}\n"},
	}
	for _, c := range cases {
		f := YodaStyle{}.WithConfig(c.config).(fixerRule)
		if got, _ := apply(t, f, c.in); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
