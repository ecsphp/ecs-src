package rules

import "testing"

func TestMethodArgumentSpaceConfig(t *testing.T) {
	// default: one space after each comma
	got, changed := apply(t, MethodArgumentSpace{}, "<?php foo(1,2);")
	if want := "<?php foo(1, 2);"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q", changed, got)
	}

	// keep_multiple_spaces_after_comma=true leaves the extra spaces
	keep := MethodArgumentSpace{}.WithConfig(map[string]any{"keep_multiple_spaces_after_comma": true}).(fixerRule)
	got, _ = apply(t, keep, "<?php foo(1,   2);")
	if want := "<?php foo(1,   2);"; got != want {
		t.Fatalf("keep spaces: got=%q", got)
	}

	// on_multiline=ignore leaves a multiline list untouched
	ignore := MethodArgumentSpace{}.WithConfig(map[string]any{"on_multiline": "ignore"}).(fixerRule)
	got, _ = apply(t, ignore, "<?php foo(1,\n    2);")
	if want := "<?php foo(1,\n    2);"; got != want {
		t.Fatalf("ignore: got=%q", got)
	}

	// on_multiline=ensure_single_line collapses a multiline list
	single := MethodArgumentSpace{}.WithConfig(map[string]any{"on_multiline": "ensure_single_line"}).(fixerRule)
	got, changed = apply(t, single, "<?php foo(\n    1,\n    2\n);")
	if want := "<?php foo(1, 2);"; !changed || got != want {
		t.Fatalf("ensure_single_line: changed=%v got=%q", changed, got)
	}

	// a trailing line comment stays on the argument's line, break goes after it
	fully := MethodArgumentSpace{}.WithConfig(map[string]any{"on_multiline": "ensure_fully_multiline"}).(fixerRule)
	got, _ = apply(t, fully, "<?php foo(\n    1, // one\n    2, // two\n);")
	if want := "<?php foo(\n    1, // one\n    2, // two\n);"; got != want {
		t.Fatalf("trailing comment: got=%q", got)
	}

	// a newline directly before ")" makes the call multiline: break every argument
	got, _ = apply(t, fully, "<?php foo('a', [\n    1,\n]\n);")
	if want := "<?php foo(\n    'a',\n    [\n    1,\n]\n);"; got != want {
		t.Fatalf("newline before close: got=%q", got)
	}
	// ")" on the same line as the last argument is not multiline: left alone
	got, _ = apply(t, fully, "<?php foo('a', [\n    1,\n]);")
	if want := "<?php foo('a', [\n    1,\n]);"; got != want {
		t.Fatalf("close shares line: got=%q", got)
	}
}
