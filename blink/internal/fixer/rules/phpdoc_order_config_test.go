package rules

import "testing"

func TestPhpdocOrderConfig(t *testing.T) {
	t.Parallel()
	returnFirst := PhpdocOrder{}.WithConfig(map[string]any{"order": []any{"param", "return", "throws"}}).(fixerRule)

	// with return before throws, a @return above a @throws is already in order
	ordered := "<?php\n/**\n * @param int $a\n * @return int\n *\n * @throws \\RuntimeException\n */\nfunction f($a) {}"
	assertFix(t, returnFirst, ordered, ordered, false)

	// and a @throws above the @return moves below it
	assertFix(t, returnFirst,
		"<?php\n/**\n * @throws \\RuntimeException\n * @return int\n */\nfunction f() {}",
		"<?php\n/**\n * @return int\n * @throws \\RuntimeException\n */\nfunction f() {}", true)

	// the default (param, throws, return) moves @throws up instead; the
	// content moves while the blank separator line keeps its slot, as in PHP
	assertFix(t, PhpdocOrder{},
		"<?php\n/**\n * @return int\n *\n * @throws \\RuntimeException\n */\nfunction f() {}",
		"<?php\n/**\n *\n * @throws \\RuntimeException\n * @return int\n */\nfunction f() {}", true)

	// an absent option keeps the default
	assertFix(t, PhpdocOrder{}.WithConfig(map[string]any{}).(fixerRule),
		"<?php\n/**\n * @return int\n * @param int $a\n */\nfunction f($a) {}",
		"<?php\n/**\n * @param int $a\n * @return int\n */\nfunction f($a) {}", true)

	// phpstan-/psalm- variants sort right after their tag; the moved
	// annotations take the vacated line slots, so @see ends up first
	assertFix(t, PhpdocOrder{}.WithConfig(map[string]any{"order": []any{"return", "param"}}).(fixerRule),
		"<?php\n/**\n * @param int $a\n * @see other()\n * @phpstan-return positive-int\n * @return int\n */\nfunction f($a) {}",
		"<?php\n/**\n * @see other()\n * @return int\n * @phpstan-return positive-int\n * @param int $a\n */\nfunction f($a) {}", true)

	// a variant listed itself gets its own slot
	assertFix(t, PhpdocOrder{}.WithConfig(map[string]any{"order": []any{"phpstan-param", "param", "return"}}).(fixerRule),
		"<?php\n/**\n * @param int $a\n * @phpstan-param positive-int $a\n */\nfunction f($a) {}",
		"<?php\n/**\n * @phpstan-param positive-int $a\n * @param int $a\n */\nfunction f($a) {}", true)

	// an order PHP rejects (fewer than two tags, duplicates) keeps the default
	for _, bad := range [][]any{{"return"}, {"return", "param", "return"}} {
		f := PhpdocOrder{}.WithConfig(map[string]any{"order": bad}).(fixerRule)
		assertFix(t, f,
			"<?php\n/**\n * @return int\n * @param int $a\n */\nfunction f($a) {}",
			"<?php\n/**\n * @param int $a\n * @return int\n */\nfunction f($a) {}", true)
	}

	// the docblock's \s, \S and \R are Unicode: a tag name ends at a no-break
	// space, a line holding only one is blank, and \u2028 breaks no line
	assertFix(t, PhpdocOrder{},
		"<?php\n/**\n * @return\u00a0int\n * @param int $a\n */\nfunction f() {}",
		"<?php\n/**\n * @param int $a\n * @return\u00a0int\n */\nfunction f() {}", true)
	assertFix(t, PhpdocOrder{},
		"<?php\n/**\n * @return int\n * \u00a0\n * @param int $a\n */\nfunction f() {}",
		"<?php\n/**\n * \u00a0\n * @param int $a\n * @return int\n */\nfunction f() {}", true)
	assertFix(t, PhpdocOrder{},
		"<?php\n/**\n * @return int\n *\u2028 @param int $a\n */\nfunction f() {}",
		"<?php\n/**\n *\u2028 @param int $a\n * @return int\n */\nfunction f() {}", true)
}
