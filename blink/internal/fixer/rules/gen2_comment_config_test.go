package rules

import "testing"

func TestAlignMultilineCommentConfig(t *testing.T) {
	t.Parallel()
	// default keeps blink's behavior: a line without a "*" is left untouched
	src := "<?php\n/**\n * foo\nbar\n */\n"
	got, changed := apply(t, AlignMultilineComment{}, src)
	if got != src || changed {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, src)
	}

	// all_multiline inserts the missing "*" and realigns the line
	f := AlignMultilineComment{}.WithConfig(map[string]any{"comment_type": "all_multiline"})
	got, changed = apply(t, f.(fixerRule), src)
	if want := "<?php\n/**\n * foo\n * bar\n */\n"; !changed || got != want {
		t.Fatalf("all_multiline: changed=%v got=%q want=%q", changed, got, want)
	}
}
