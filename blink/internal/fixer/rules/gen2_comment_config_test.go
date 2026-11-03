package rules

import "testing"

func TestAlignMultilineCommentConfig(t *testing.T) {
	t.Parallel()
	// default is "phpdocs_only": a docblock is aligned and a missing "*" is
	// inserted, matching PHP-CS-Fixer
	doc := "<?php\n/**\n * foo\nbar\n */\n"
	got, changed := apply(t, AlignMultilineComment{}, doc)
	if want := "<?php\n/**\n * foo\n * bar\n */\n"; !changed || got != want {
		t.Fatalf("default docblock: changed=%v got=%q want=%q", changed, got, want)
	}

	// a plain block comment is left alone by default, but realigned with all_multiline
	plain := "<?php\n/*\nbar\n */\n"
	if got, changed := apply(t, AlignMultilineComment{}, plain); changed || got != plain {
		t.Fatalf("default plain: changed=%v got=%q", changed, got)
	}
	f := AlignMultilineComment{}.WithConfig(map[string]any{"comment_type": "all_multiline"})
	got, changed = apply(t, f.(fixerRule), plain)
	if want := "<?php\n/*\n * bar\n */\n"; !changed || got != want {
		t.Fatalf("all_multiline: changed=%v got=%q want=%q", changed, got, want)
	}
}
