package rules

import "testing"

// Cases ported from PHP-CS-Fixer's HeaderCommentFixerTest; every expected output
// was produced by running the upstream fixer with the matching configuration.
func TestHeaderCommentPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		fixer    HeaderComment
		in, want string
	}{
		{
			"insert after declare",
			HeaderComment{Header: "Made with love."},
			"<?php\ndeclare(strict_types=1);\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\ndeclare(strict_types=1);\n\n/*\n * Made with love.\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"insert after declare with no whitespace",
			HeaderComment{Header: "tmp", Location: "after_declare_strict"},
			"<?php\ndeclare(strict_types=1);namespace A\\B;\n\necho 1;\n",
			"<?php\ndeclare(strict_types=1);\n\n/*\n * tmp\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"after_open",
			HeaderComment{Header: "Made with love.", Location: "after_open"},
			"<?php\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\n\n/*\n * Made with love.\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"after_open skips the declare",
			HeaderComment{Header: "Made with love.", Location: "after_open"},
			"<?php\ndeclare(strict_types=1);\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\n\n/*\n * Made with love.\n */\n\ndeclare(strict_types=1);\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"PHPDoc with separate bottom",
			HeaderComment{Header: "tmp", Location: "after_declare_strict", Separate: "bottom", CommentType: "PHPDoc"},
			"<?php\ndeclare(strict_types=1);\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\ndeclare(strict_types=1);\n/**\n * tmp\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"separate none",
			HeaderComment{Header: "tmp", Separate: "none"},
			"<?php\ndeclare(strict_types=1);\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\ndeclare(strict_types=1);\n/*\n * tmp\n */\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"separate top",
			HeaderComment{Header: "tmp", Separate: "top"},
			"<?php\ndeclare(strict_types=1);\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\ndeclare(strict_types=1);\n\n/*\n * tmp\n */\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"replace existing header",
			HeaderComment{Header: "new"},
			"<?php\ndeclare(strict_types=1);\n/*\n * old\n */\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\ndeclare(strict_types=1);\n\n/*\n * new\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"already correct is a no-op",
			HeaderComment{Header: "Made with love."},
			"<?php\ndeclare(strict_types=1);\n\n/*\n * Made with love.\n */\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\ndeclare(strict_types=1);\n\n/*\n * Made with love.\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"multiline header",
			HeaderComment{Header: "line one\nline two"},
			"<?php\ndeclare(strict_types=1);\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\ndeclare(strict_types=1);\n\n/*\n * line one\n * line two\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"no declare inserts after open",
			HeaderComment{Header: "tmp"},
			"<?php\nnamespace A\\B;\n\necho 1;\n",
			"<?php\n\n/*\n * tmp\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
		{
			"relocate header from open to declare",
			HeaderComment{Header: "tmp", Location: "after_declare_strict"},
			"<?php\n\n/*\n * tmp\n */\n\ndeclare(strict_types=1);\n\nnamespace A\\B;\n\necho 1;\n",
			"<?php\n\n\ndeclare(strict_types=1);\n\n/*\n * tmp\n */\n\nnamespace A\\B;\n\necho 1;\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, changed := apply(t, c.fixer, c.in)
			if got != c.want {
				t.Fatalf("got=%q want=%q", got, c.want)
			}
			if wantChange := c.in != c.want; changed != wantChange {
				t.Fatalf("changed=%v want %v", changed, wantChange)
			}
			if again, changedAgain := apply(t, c.fixer, got); changedAgain || again != got {
				t.Fatalf("not idempotent: changed=%v got=%q", changedAgain, again)
			}
		})
	}
}
