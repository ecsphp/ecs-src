package rules

import "testing"

func TestClassAttributesSeparationElements(t *testing.T) {
	t.Parallel()
	// two properties separated by two blank lines
	src := "<?php\nclass C\n{\n    public $a;\n\n\n    public $b;\n}\n"

	// default: property spacing is "one" -> collapse to a single blank line
	got, changed := apply(t, ClassAttributesSeparation{}, src)
	if want := "<?php\nclass C\n{\n    public $a;\n\n    public $b;\n}\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q want=%q", changed, got, want)
	}

	// elements={property: none}: no blank line between same-type properties
	cfg := ClassAttributesSeparation{}.WithConfig(map[string]any{
		"elements": map[string]any{"property": "none"},
	})
	got, changed = apply(t, cfg, src)
	if want := "<?php\nclass C\n{\n    public $a;\n    public $b;\n}\n"; !changed || got != want {
		t.Fatalf("property=none: changed=%v got=%q want=%q", changed, got, want)
	}
}
