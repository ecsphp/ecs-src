package rules

import "testing"

func TestGeneralPhpdocTagRenameConfig(t *testing.T) {
	t.Parallel()
	cfg := func(extra map[string]any) fixerRule {
		config := map[string]any{"replacements": map[string]any{"inheritDocs": "inheritDoc"}}
		for k, v := range extra {
			config[k] = v
		}
		return GeneralPhpdocTagRename{}.WithConfig(config).(fixerRule)
	}

	// unconfigured: no replacements, a no-op
	src := "<?php\n/**\n * @inheritDocs\n * {@inheritDocs}\n */\nfunction f() {}"
	assertFix(t, GeneralPhpdocTagRename{}, src, src, false)

	// annotations and inline tags, case-insensitively by default
	assertFix(t, cfg(nil),
		"<?php\n/**\n * @inheritdocs\n * {@INHERITDOCS}\n */\nfunction f() {}",
		"<?php\n/**\n * @inheritDoc\n * {@inheritDoc}\n */\nfunction f() {}", true)

	// whole tags only: a longer tag is not renamed
	assertFix(t, cfg(nil),
		"<?php\n/**\n * @inheritDocsFoo\n */\nfunction f() {}",
		"<?php\n/**\n * @inheritDocsFoo\n */\nfunction f() {}", false)

	// case_sensitive: only the exact casing
	assertFix(t, cfg(map[string]any{"case_sensitive": true}),
		"<?php\n/**\n * @inheritdocs\n * @inheritDocs\n */\nfunction f() {}",
		"<?php\n/**\n * @inheritdocs\n * @inheritDoc\n */\nfunction f() {}", true)

	// fix_inline off: inline tags stay
	assertFix(t, cfg(map[string]any{"fix_inline": false}),
		"<?php\n/**\n * @inheritDocs\n * {@inheritDocs}\n */\nfunction f() {}",
		"<?php\n/**\n * @inheritDoc\n * {@inheritDocs}\n */\nfunction f() {}", true)

	// fix_annotation off: only inline tags
	assertFix(t, cfg(map[string]any{"fix_annotation": false}),
		"<?php\n/**\n * @inheritDocs\n * {@inheritDocs}\n */\nfunction f() {}",
		"<?php\n/**\n * @inheritDocs\n * {@inheritDoc}\n */\nfunction f() {}", true)

	// text from a quote to the last same quote is skipped
	quoted := "<?php\n/**\n * Don't use @inheritDocs, it's gone.\n */\nfunction f() {}"
	assertFix(t, cfg(nil), quoted, quoted, false)

	// from JSON, the replacements keep their order: the first key that
	// matches wins, as in PHP's regex alternation
	doc := "<?php\n/**\n * @foo-bar\n * @foo\n */\nfunction f() {}"
	assertFix(t, configuredFromJSON(t, GeneralPhpdocTagRename{}, `{"replacements": {"foo-bar": "qux", "foo": "baz"}}`),
		doc, "<?php\n/**\n * @qux\n * @baz\n */\nfunction f() {}", true)
	assertFix(t, configuredFromJSON(t, GeneralPhpdocTagRename{}, `{"replacements": {"foo": "baz", "foo-bar": "qux"}}`),
		doc, "<?php\n/**\n * @baz-bar\n * @baz\n */\nfunction f() {}", true)
	// a key lowercased onto an earlier one keeps the earlier position
	assertFix(t, configuredFromJSON(t, GeneralPhpdocTagRename{}, `{"replacements": {"Foo-Bar": "qux", "foo": "baz", "foo-bar": "qux"}}`),
		doc, "<?php\n/**\n * @qux\n * @baz\n */\nfunction f() {}", true)
	// a target may end with one newline, which is trimmed
	assertFix(t, configuredFromJSON(t, GeneralPhpdocTagRename{}, `{"replacements": {"inheritDocs": "inheritDoc\n"}}`),
		"<?php\n/**\n * @inheritDocs\n */\nfunction f() {}", "<?php\n/**\n * @inheritDoc\n */\nfunction f() {}", true)

	// \b is Unicode: a letter or combining mark after the tag is part of the word
	assertFix(t, cfg(nil),
		"<?php\n/**\n * @inheritDocsé\n * @inheritDocs\u00a0x\n * {@inheritDocsé}\n * @inheritDocs\u0301\n */\nfunction f() {}",
		"<?php\n/**\n * @inheritDocsé\n * @inheritDoc\u00a0x\n * {@inheritDocsé}\n * @inheritDocs\u0301\n */\nfunction f() {}", true)

	// a replacement chain PHP rejects leaves the fixer a no-op
	chain := GeneralPhpdocTagRename{}.WithConfig(map[string]any{"replacements": map[string]any{"a": "b", "b": "c"}}).(fixerRule)
	assertFix(t, chain, "<?php\n/**\n * @a\n */\nfunction f() {}", "<?php\n/**\n * @a\n */\nfunction f() {}", false)
}
