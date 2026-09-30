package rules

import "testing"

func TestPhpdocLineSpanConfig(t *testing.T) {
	single := "<?php\nclass A {\n    /** @var int */\n    public $x;\n}\n"
	multi := "<?php\nclass A {\n    /**\n     * @var int\n     */\n    public $x;\n}\n"

	// default expands a single-line member docblock to multi line
	got, changed := apply(t, PhpdocLineSpan{}, single)
	if !changed || got != multi {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, multi)
	}

	// property=single collapses the multi-line docblock back to one line
	f := PhpdocLineSpan{}.WithConfig(map[string]any{"property": "single"})
	got, changed = apply(t, f.(fixerRule), multi)
	if !changed || got != single {
		t.Fatalf("single: changed=%v got=%q want=%q", changed, got, single)
	}
}
