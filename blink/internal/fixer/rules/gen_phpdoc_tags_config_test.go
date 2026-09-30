package rules

import "testing"

func TestPhpdocTagCasingConfig(t *testing.T) {
	got, changed := apply(t, PhpdocTagCasing{}, "<?php\n/**\n * @inheritdoc\n */\n")
	if want := "<?php\n/**\n * @inheritDoc\n */\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// configured to fix the casing of @foo only
	f := PhpdocTagCasing{}.WithConfig(map[string]any{"tags": []any{"foo"}})
	got, changed = apply(t, f.(fixerRule), "<?php\n/**\n * @Foo\n */\n")
	if want := "<?php\n/**\n * @foo\n */\n"; !changed || got != want {
		t.Fatalf("configured: changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, f.(fixerRule), "<?php\n/**\n * @inheritdoc\n */\n"); changed {
		t.Fatal("inheritdoc must not be touched when only foo is configured")
	}
}

func TestPhpdocInlineTagNormalizerConfig(t *testing.T) {
	got, changed := apply(t, PhpdocInlineTagNormalizer{}, "<?php\n/**\n * @{see}\n */\n")
	if want := "<?php\n/**\n * {@see}\n */\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// configured to normalize @{id} only, leaving @{see} alone
	f := PhpdocInlineTagNormalizer{}.WithConfig(map[string]any{"tags": []any{"id"}})
	got, changed = apply(t, f.(fixerRule), "<?php\n/**\n * @{id}\n */\n")
	if want := "<?php\n/**\n * {@id}\n */\n"; !changed || got != want {
		t.Fatalf("configured: changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, f.(fixerRule), "<?php\n/**\n * @{see}\n */\n"); changed {
		t.Fatal("see must not be normalized when only id is configured")
	}
}
