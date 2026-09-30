package rules

import "testing"

func TestPhpdocReturnSelfReferenceConfig(t *testing.T) {
	src := "<?php\n/**\n * @return this\n */\n"
	got, changed := apply(t, PhpdocReturnSelfReference{}, src)
	if want := "<?php\n/**\n * @return $this\n */\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	f := PhpdocReturnSelfReference{}.WithConfig(map[string]any{"replacements": map[string]any{"this": "self"}})
	got, changed = apply(t, f.(fixerRule), src)
	if want := "<?php\n/**\n * @return self\n */\n"; !changed || got != want {
		t.Fatalf("configured: changed=%v got=%q want=%q", changed, got, want)
	}
}
