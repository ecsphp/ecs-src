package rules

import "testing"

func TestControlStructureContinuationConfig(t *testing.T) {
	// default: the continuation keyword joins the closing brace line
	got, changed := apply(t, ControlStructureContinuationPosition{}, "<?php\nif ($a) {\n}\nelse {\n}\n")
	if want := "<?php\nif ($a) {\n} else {\n}\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q", changed, got)
	}

	// position=next_line moves the continuation keyword onto its own line
	nl := ControlStructureContinuationPosition{}.WithConfig(map[string]any{"position": "next_line"}).(fixerRule)
	got, changed = apply(t, nl, "<?php\nif ($a) {\n} else {\n}\n")
	if want := "<?php\nif ($a) {\n}\nelse {\n}\n"; !changed || got != want {
		t.Fatalf("next_line: changed=%v got=%q", changed, got)
	}
}
