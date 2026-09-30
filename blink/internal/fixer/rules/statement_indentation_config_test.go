package rules

import "testing"

func TestStatementIndentationConfig(t *testing.T) {
	src := "<?php\nif ($foo) {\n    echo \"foo\";\n        // c\n} else {\n    $a = 1;\n}\n"

	// default: the trailing comment keeps the inner block indentation
	got, changed := apply(t, StatementIndentation{}, src)
	if want := "<?php\nif ($foo) {\n    echo \"foo\";\n    // c\n} else {\n    $a = 1;\n}\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q", changed, got)
	}

	// stick_comment...: the trailing comment dedents to the enclosing level
	stick := StatementIndentation{}.WithConfig(map[string]any{"stick_comment_to_next_continuous_control_statement": true}).(fixerRule)
	got, changed = apply(t, stick, src)
	if want := "<?php\nif ($foo) {\n    echo \"foo\";\n// c\n} else {\n    $a = 1;\n}\n"; !changed || got != want {
		t.Fatalf("stick: changed=%v got=%q", changed, got)
	}
}
