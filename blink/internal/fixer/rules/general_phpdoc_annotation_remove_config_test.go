package rules

import "testing"

func TestGeneralPhpdocAnnotationRemoveConfig(t *testing.T) {
	src := "<?php\n/**\n * @author Bob\n * @internal\n */\n"

	// default removes @author (author/package/group/category)
	got, changed := apply(t, GeneralPhpdocAnnotationRemove{}, src)
	if want := "<?php\n/**\n * @internal\n */\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// configured to remove @internal instead
	f := GeneralPhpdocAnnotationRemove{}.WithConfig(map[string]any{"annotations": []any{"internal"}})
	got, changed = apply(t, f.(fixerRule), src)
	if want := "<?php\n/**\n * @author Bob\n */\n"; !changed || got != want {
		t.Fatalf("configured: changed=%v got=%q want=%q", changed, got, want)
	}

	// case_sensitive keeps a differently-cased annotation
	cs := GeneralPhpdocAnnotationRemove{}.WithConfig(map[string]any{
		"annotations":    []any{"internal"},
		"case_sensitive": true,
	})
	if _, changed := apply(t, cs.(fixerRule), "<?php\n/**\n * @Internal\n */\n"); changed {
		t.Fatal("case_sensitive must not remove @Internal for tag internal")
	}
}
