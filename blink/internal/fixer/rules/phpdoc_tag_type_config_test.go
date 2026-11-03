package rules

import "testing"

func TestPhpdocTagTypeConfig(t *testing.T) {
	t.Parallel()
	inline := PhpdocTagType{}.WithConfig(map[string]any{"tags": map[string]any{"inheritDoc": "inline"}}).(fixerRule)

	// configured inline: a standalone annotation becomes an inline tag ...
	assertFix(t, inline,
		"<?php\n/**\n * @inheritdoc\n */\nfunction f() {}",
		"<?php\n/**\n * {@inheritdoc}\n */\nfunction f() {}", true)
	// ... and an inline one is left as it is, where the default would rewrite it
	assertFix(t, inline,
		"<?php\n/** {@inheritDoc} */\nfunction f() {}",
		"<?php\n/** {@inheritDoc} */\nfunction f() {}", false)
	assertFix(t, PhpdocTagType{},
		"<?php\n/** {@inheritDoc} */\nfunction f() {}",
		"<?php\n/** @inheritDoc */\nfunction f() {}", true)

	// the configured map replaces the default one: @link is not in it
	assertFix(t, inline,
		"<?php\n/**\n * {@link https://example.com}\n */\nfunction f() {}",
		"<?php\n/**\n * {@link https://example.com}\n */\nfunction f() {}", false)

	// the default covers more than inheritDoc
	assertFix(t, PhpdocTagType{}.WithConfig(map[string]any{}).(fixerRule),
		"<?php\n/**\n * {@see Other}\n */\nfunction f() {}",
		"<?php\n/**\n * @see Other\n */\nfunction f() {}", true)

	// an inline tag surrounded by text is never made an annotation
	surrounded := "<?php\n/**\n * Summary.\n *\n * {@inheritdoc}\n */\nfunction f() {}"
	assertFix(t, PhpdocTagType{}, surrounded, surrounded, false)

	// an empty map turns the fixer off; PHP dumps it as a JSON list
	off := configuredFromJSON(t, PhpdocTagType{}, `{"tags": []}`)
	assertFix(t, off, "<?php\n/** {@inheritDoc} */\nfunction f() {}", "<?php\n/** {@inheritDoc} */\nfunction f() {}", false)

	// \s is Unicode (a no-break space here): before the closer, and after the name
	assertFix(t, PhpdocTagType{},
		"<?php\n/** {@inheritDoc}\u00a0*/\nfunction f() {}",
		"<?php\n/** @inheritDoc\u00a0*/\nfunction f() {}", true)
	assertFix(t, PhpdocTagType{},
		"<?php\n/**\n * {@inheritDoc\u00a0foo}\n */\nfunction f() {}",
		"<?php\n/**\n * @inheritDoc\u00a0foo\n */\nfunction f() {}", true)
}
