package rules

import "testing"

// Expectations are PHP-CS-Fixer's own output for a comment after an argument
// list's "(" (its fixNewline): the comment keeps its place and only an
// argument still on the comment's line is moved.
func TestMethodArgumentSpaceCommentAfterOpenParen(t *testing.T) {
	t.Parallel()
	f := MethodArgumentSpace{}.WithConfig(map[string]any{"on_multiline": "ensure_fully_multiline"}).(fixerRule)
	for _, c := range []struct{ name, src, want string }{
		{
			"docblock after a space stays",
			"<?php\n$a->info( /** @phpstan-ignore x */\n    sprintf('%d', 1)\n);\n",
			"<?php\n$a->info( /** @phpstan-ignore x */\n    sprintf('%d', 1)\n);\n",
		},
		{
			"line comment after a space stays",
			"<?php\nfoo( // note\n    $a,\n    $b\n);\n",
			"<?php\nfoo( // note\n    $a,\n    $b\n);\n",
		},
		{
			"comment right after the paren stays",
			"<?php\nfoo(/* c */\n    $a,\n    $b\n);\n",
			"<?php\nfoo(/* c */\n    $a,\n    $b\n);\n",
		},
		{
			"argument on the comment's line is moved",
			"<?php\nfoo( /* c */ $a,\n    $b\n);\n",
			"<?php\nfoo( /* c */\n    $a,\n    $b\n);\n",
		},
	} {
		got, _ := apply(t, f, c.src)
		if got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
