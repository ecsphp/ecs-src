package rules

import "testing"

func TestNoAlternativeSyntaxConfig(t *testing.T) {
	src := "<?php if ($a): ?>\nHi\n<?php endif; ?>\n"
	if _, changed := apply(t, NoAlternativeSyntax{}, src); !changed {
		t.Fatal("default should fix inline HTML code")
	}
	f := NoAlternativeSyntax{}.WithConfig(map[string]any{"fix_non_monolithic_code": false})
	if _, changed := apply(t, f.(fixerRule), src); changed {
		t.Fatal("fix_non_monolithic_code=false should skip inline HTML code")
	}
	got, changed := apply(t, f.(fixerRule), "<?php if ($a): echo 1; endif;")
	if want := "<?php if ($a) { echo 1; }"; !changed || got != want {
		t.Fatalf("monolithic: changed=%v got=%q", changed, got)
	}
}
