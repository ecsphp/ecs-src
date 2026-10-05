package rules

import "testing"

func TestBracesPositionConfig(t *testing.T) {
	// default: a class body brace moves to the next line
	got, changed := apply(t, BracesPosition{}, "<?php\nclass A {\n    public $x;\n}\n")
	if want := "<?php\nclass A\n{\n    public $x;\n}\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q", changed, got)
	}

	// classes_opening_brace=same_line keeps the brace on the class line
	same := BracesPosition{}.WithConfig(map[string]any{"classes_opening_brace": "same_line"}).(fixerRule)
	got, changed = apply(t, same, "<?php\nclass A\n{\n    public $x;\n}\n")
	if want := "<?php\nclass A {\n    public $x;\n}\n"; !changed || got != want {
		t.Fatalf("classes same_line: changed=%v got=%q", changed, got)
	}

	// control_structures_opening_brace=next_line moves the control brace down
	ctrl := BracesPosition{}.WithConfig(map[string]any{"control_structures_opening_brace": "next_line_unless_newline_at_signature_end"}).(fixerRule)
	got, changed = apply(t, ctrl, "<?php\nif ($a) {\n    echo 1;\n}\n")
	if want := "<?php\nif ($a)\n{\n    echo 1;\n}\n"; !changed || got != want {
		t.Fatalf("control next_line: changed=%v got=%q", changed, got)
	}

	// allow_single_line_anonymous_functions=false expands a single-line closure body
	noSingle := BracesPosition{}.WithConfig(map[string]any{"allow_single_line_anonymous_functions": false}).(fixerRule)
	got, changed = apply(t, noSingle, "<?php\n$f = function ($x) { return $x; };\n")
	if want := "<?php\n$f = function ($x) {\n    return $x;\n};\n"; !changed || got != want {
		t.Fatalf("closure expand: changed=%v got=%q", changed, got)
	}
	// an empty body or a comment on the brace line is left alone
	if _, changed := apply(t, noSingle, "<?php\n$f = function ($x) {};\n"); changed {
		t.Fatal("empty closure body must be left alone")
	}
	if _, changed := apply(t, noSingle, "<?php\n$f = function ($x) { // note\n    return $x;\n};\n"); changed {
		t.Fatal("comment on brace line must be left alone")
	}
	// default (option unset) keeps a single-line closure body
	if _, changed := apply(t, BracesPosition{}, "<?php\n$f = function ($x) { return $x; };\n"); changed {
		t.Fatal("default must keep single-line closure body")
	}
}
