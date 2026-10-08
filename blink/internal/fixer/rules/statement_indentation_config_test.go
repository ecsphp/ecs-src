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

func TestStatementIndentationTernaryDynamicMember(t *testing.T) {
	// a dynamic member access ("->{$key}") in a multiline ternary must not be read
	// as a block/statement end; the ":" line stays aligned with the "?" line
	src := "<?php\nclass A\n{\n    public function x($key)\n    {\n        return $this->condition\n            ? $this->target->{$key}\n            : $this->target;\n    }\n}\n"
	got, changed := apply(t, StatementIndentation{}, src)
	if changed || got != src {
		t.Fatalf("ternary with dynamic member must stay aligned: changed=%v got=%q", changed, got)
	}
}
