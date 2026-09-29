package rules

import "testing"

func TestStatementIndentation(t *testing.T) {
	got, changed := apply(t, StatementIndentation{}, "<?php\nif ($a) {\nreturn 1;\n}")
	if want := "<?php\nif ($a) {\n    return 1;\n}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// method chaining is a continuation - left alone
	chain := "<?php\n$x = $this\n    ->foo()\n    ->bar();\n"
	if _, changed := apply(t, StatementIndentation{}, chain); changed {
		t.Fatal("method chain must not be reindented")
	}
	// multiline arguments are continuations - left alone
	args := "<?php\nfoo(\n    $a,\n    $b\n);\n"
	if _, changed := apply(t, StatementIndentation{}, args); changed {
		t.Fatal("multiline arguments must not be reindented")
	}
}

func TestBracesPositionMultilineSignature(t *testing.T) {
	// multiline signature keeps ") {" on one line (PSR-12 4.5)
	multi := "<?php function foo(\n    $a,\n    $b\n) {\n}"
	if _, changed := apply(t, BracesPosition{}, multi); changed {
		t.Fatalf("multiline signature brace must stay inline: %q", multi)
	}
	// single-line signature moves the brace to the next line
	got, changed := apply(t, BracesPosition{}, "<?php function foo() {\n}")
	if want := "<?php function foo()\n{\n}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}
