package rules

import "testing"

func TestSingleQuoteConfig(t *testing.T) {
	t.Parallel()
	src := `<?php $a = "sample"; $b = "it's";`
	got, _ := apply(t, SingleQuote{}, src)
	if want := `<?php $a = 'sample'; $b = "it's";`; got != want {
		t.Fatalf("default got=%q want=%q", got, want)
	}
	r := SingleQuote{}.WithConfig(map[string]any{"strings_containing_single_quote_chars": true}).(fixerRule)
	got, _ = apply(t, r, src)
	if want := `<?php $a = 'sample'; $b = 'it\'s';`; got != want {
		t.Fatalf("configured got=%q want=%q", got, want)
	}
}
