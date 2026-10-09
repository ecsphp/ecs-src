package rules

import "testing"

func TestNoUselessConcatOperatorConfig(t *testing.T) {
	t.Parallel()
	src := `<?php $a = "a" . 'b';`
	if _, changed := apply(t, NoUselessConcatOperator{}, src); changed {
		t.Fatal("default should not juggle quotes")
	}
	f := NoUselessConcatOperator{}.WithConfig(map[string]any{"juggle_simple_strings": true})
	got, changed := apply(t, f.(fixerRule), src)
	if want := `<?php $a = "ab";`; !changed || got != want {
		t.Fatalf("juggle: changed=%v got=%q want=%q", changed, got, want)
	}
}
