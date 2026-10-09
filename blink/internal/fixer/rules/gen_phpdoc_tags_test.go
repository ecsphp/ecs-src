package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGenPhpdocTagsTagCasing(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * @inheritdoc\n */\nfunction f() {}"
	got, changed := apply(t, PhpdocTagCasing{}, src)
	want := "<?php\n/**\n * @inheritDoc\n */\nfunction f() {}"
	if !changed || got != want {
		t.Fatalf("annotation: changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, PhpdocTagCasing{}, got)

	// inline position is normalized too
	got, _ = apply(t, PhpdocTagCasing{}, "<?php\n/**\n * {@INHERITDOC}\n */\nfunction f() {}")
	if want := "<?php\n/**\n * {@inheritDoc}\n */\nfunction f() {}"; got != want {
		t.Fatalf("inline: got %q", got)
	}

	// already correct is a no-op
	if _, changed := apply(t, PhpdocTagCasing{}, want); changed {
		t.Fatal("correct casing must not change")
	}
	// "@inheritdocs" and bare prose are left alone
	if _, changed := apply(t, PhpdocTagCasing{}, "<?php\n/**\n * @inheritdocs and inheritdoc prose\n */\nfunction f() {}"); changed {
		t.Fatal("longer tag and prose must not change")
	}
}

func TestGenPhpdocTagsInlineNormalizer(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * {{ @link }}\n * @{see http://x}\n */\nfunction f() {}"
	got, changed := apply(t, PhpdocInlineTagNormalizer{}, src)
	want := "<?php\n/**\n * {@link}\n * {@see http://x}\n */\nfunction f() {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, PhpdocInlineTagNormalizer{}, got)

	// inner spaces are trimmed
	got, _ = apply(t, PhpdocInlineTagNormalizer{}, "<?php\n/**\n * { @see X }\n */\nfunction f() {}")
	if want := "<?php\n/**\n * {@see X}\n */\nfunction f() {}"; got != want {
		t.Fatalf("trim: got %q", got)
	}

	// tag casing is preserved (that belongs to PhpdocTagCasing)
	if _, changed := apply(t, PhpdocInlineTagNormalizer{}, "<?php\n/**\n * {@inheritdoc}\n */\nfunction f() {}"); changed {
		t.Fatal("normalized inline tag must not change")
	}
	// prose braces without a known tag are untouched
	if _, changed := apply(t, PhpdocInlineTagNormalizer{}, "<?php\n/**\n * an { @unknown } thing\n */\nfunction f() {}"); changed {
		t.Fatal("unknown inline tag must not change")
	}
}

func TestGenPhpdocTagsNoDuplicateTypes(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * @param int|int|string $bar\n */\nfunction f($bar) {}"
	got, changed := apply(t, PhpdocNoDuplicateTypes{}, src)
	want := "<?php\n/**\n * @param int|string $bar\n */\nfunction f($bar) {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, PhpdocNoDuplicateTypes{}, got)

	// dedupe is case-insensitive and keeps the first spelling
	got, _ = apply(t, PhpdocNoDuplicateTypes{}, "<?php\n/**\n * @var Foo|FOO|Bar $x\n */\nfunction f() {}")
	if want := "<?php\n/**\n * @var Foo|Bar $x\n */\nfunction f() {}"; got != want {
		t.Fatalf("case-insensitive: got %q", got)
	}

	// already unique is a no-op
	if _, changed := apply(t, PhpdocNoDuplicateTypes{}, want); changed {
		t.Fatal("unique union must not change")
	}
	// a nested generic union is left untouched (its "|" is not a top-level separator)
	if _, changed := apply(t, PhpdocNoDuplicateTypes{}, "<?php\n/**\n * @param array<int|string> $a\n */\nfunction f($a) {}"); changed {
		t.Fatal("nested generic type must not change")
	}
}

// TestGenPhpdocTagsSourceURLs verifies every SourceURL matches SourceURLFor(Name()).
func TestGenPhpdocTagsSourceURLs(t *testing.T) {
	t.Parallel()
	fixers := []fixer.Fixer{
		PhpdocTagCasing{},
		PhpdocInlineTagNormalizer{},
		PhpdocNoDuplicateTypes{},
	}
	for _, f := range fixers {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}
