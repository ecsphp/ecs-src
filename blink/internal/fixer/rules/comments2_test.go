package rules

import "testing"

func TestNoEmptyCommentBlock(t *testing.T) {
	t.Parallel()
	// empty // line inside a non-empty // block is preserved (PHP leaves it as-is)
	block := "<?php\n// Native #[Route] attributes declared directly on bundle controllers.\n" +
		"// Paths already carry their full prefix (e.g. /s, /api), so they are added\n" +
		"// to the root collection; forceSSL is applied here like every other group.\n" +
		"//\n" +
		"// Skipped during installation: scanning every controller for attributes\n" +
		"$a = 1;\n"
	if got, changed := apply(t, NoEmptyComment{}, block); changed || got != block {
		t.Fatalf("non-empty // block must be untouched: changed=%v got=%q", changed, got)
	}

	// same rule for a # block
	hashBlock := "<?php\n# one\n#\n# two\n$a = 1;\n"
	if got, changed := apply(t, NoEmptyComment{}, hashBlock); changed || got != hashBlock {
		t.Fatalf("non-empty # block must be untouched: changed=%v got=%q", changed, got)
	}

	// truly empty comments are still removed, matching the PHP fixer
	cases := []struct{ name, src, want string }{
		{"lone //", "<?php\n//\n$a = 1;\n", "<?php\n\n$a = 1;\n"},
		{"lone #", "<?php\n#\n$a = 1;\n", "<?php\n\n$a = 1;\n"},
		{"block comment", "<?php\n/* */\n$a = 1;\n", "<?php\n\n$a = 1;\n"},
		{"whole empty // block", "<?php\n//\n//\n$a = 1;\n", "<?php\n\n\n$a = 1;\n"},
	}
	for _, c := range cases {
		got, changed := apply(t, NoEmptyComment{}, c.src)
		if !changed || got != c.want {
			t.Fatalf("%s: changed=%v got=%q want=%q", c.name, changed, got, c.want)
		}
	}
}
