package rules

import "testing"

func TestNoBreakComment(t *testing.T) {
	f := NoBreakComment{}

	// fall-through with a non-empty body gains a "// no break" comment
	got, changed := apply(t, f, "<?php\nswitch ($f) {\n    case 1:\n        foo();\n    case 2:\n        bar();\n        break;\n}\n")
	if want := "<?php\nswitch ($f) {\n    case 1:\n        foo();\n        // no break\n    case 2:\n        bar();\n        break;\n}\n"; !changed || got != want {
		t.Fatalf("insert: changed=%v got=%q", changed, got)
	}

	// a case that breaks needs no comment
	if _, changed := apply(t, f, "<?php\nswitch ($f) {\n    case 1:\n        foo();\n        break;\n    case 2:\n        bar();\n}\n"); changed {
		t.Fatal("case with break must not gain a comment")
	}

	// an empty fall-through case is left alone
	if _, changed := apply(t, f, "<?php\nswitch ($f) {\n    case 1:\n    case 2:\n        bar();\n}\n"); changed {
		t.Fatal("empty fall-through must not gain a comment")
	}

	// a stale comment on a non-fall-through case is removed
	got, changed = apply(t, f, "<?php\nswitch ($f) {\n    case 1:\n        foo();\n        break;\n        // no break\n    case 2:\n        bar();\n}\n")
	if want := "<?php\nswitch ($f) {\n    case 1:\n        foo();\n        break;\n    case 2:\n        bar();\n}\n"; !changed || got != want {
		t.Fatalf("remove: changed=%v got=%q", changed, got)
	}

	// an already-correct comment is a no-op
	if _, changed := apply(t, f, "<?php\nswitch ($f) {\n    case 1:\n        foo();\n        // no break\n    case 2:\n        bar();\n        break;\n}\n"); changed {
		t.Fatal("correct comment must not change")
	}

	// enum cases are never touched
	if _, changed := apply(t, f, "<?php\nenum Suit {\n    case Hearts;\n    case Spades;\n}\n"); changed {
		t.Fatal("enum cases must be ignored")
	}
}
