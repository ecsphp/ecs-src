package rules

import "testing"

func TestPhpdocScalarConfig(t *testing.T) {
	src := "<?php\n/**\n * @param integer $a\n * @param boolean $b\n */\n"

	// default converts every scalar alias
	got, changed := apply(t, PhpdocScalar{}, src)
	if want := "<?php\n/**\n * @param int $a\n * @param bool $b\n */\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// configured to convert only "boolean": integer is left alone
	f := PhpdocScalar{}.WithConfig(map[string]any{"types": []any{"boolean"}})
	got, changed = apply(t, f.(fixerRule), src)
	if want := "<?php\n/**\n * @param integer $a\n * @param bool $b\n */\n"; !changed || got != want {
		t.Fatalf("configured: changed=%v got=%q want=%q", changed, got, want)
	}
}
