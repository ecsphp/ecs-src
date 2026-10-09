package rules

import "testing"

func TestNoBreakCommentConfig(t *testing.T) {
	t.Parallel()
	src := "<?php\nswitch ($a) {\n    case 1:\n        echo 1;\n    case 2:\n        break;\n}\n"
	got, changed := apply(t, NoBreakComment{}, src)
	if want := "<?php\nswitch ($a) {\n    case 1:\n        echo 1;\n        // no break\n    case 2:\n        break;\n}\n"; !changed || got != want {
		t.Fatalf("default: changed=%v got=%q", changed, got)
	}
	f := NoBreakComment{}.WithConfig(map[string]any{"comment_text": "fall through"})
	got, changed = apply(t, f.(fixerRule), src)
	if want := "<?php\nswitch ($a) {\n    case 1:\n        echo 1;\n        // fall through\n    case 2:\n        break;\n}\n"; !changed || got != want {
		t.Fatalf("configured: changed=%v got=%q", changed, got)
	}
}
