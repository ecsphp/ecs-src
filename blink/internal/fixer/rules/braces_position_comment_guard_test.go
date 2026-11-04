package rules

import (
	"strings"
	"testing"
)

// PHP-CS-Fixer first puts the body on its own line, which blink does not do, so its output
// is no reference. The brace must not jump over the code to reach the comment.
func TestBracesPositionBraceStaysBeforeCode(t *testing.T) {
	t.Parallel()
	got, _ := apply(t, BracesPosition{}, "<?php\nclass A { public function f() { // c\n        echo 1;\n    }\n}\n")
	if !strings.HasPrefix(got, "<?php\nclass A\n{ public function f()") {
		t.Fatalf("got=%q", got)
	}
}

// The PHP engine throws on an unclosed brace, so there is no output to match. The brace
// must stay where it is, and it must not stop the run.
func TestBracesPositionUnclosedBraceAtEndOfFile(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"<?php\nif ($b) /* c */{", "<?php\nif ($b) /* c */ {"},
		{"<?php\nif ($b) // c\n{", "<?php\nif ($b) // c\n{"},
		{"<?php\nif ($b) /* c */ {", "<?php\nif ($b) /* c */ {"},
	}
	for _, c := range cases {
		if got, _ := apply(t, BracesPosition{}, c.src); got != c.want {
			t.Errorf("%s\n got=%q\nwant=%q", c.src, got, c.want)
		}
	}
}

// PHP-CS-Fixer writes the brace inside the comment on these, so its output is no reference.
func TestBracesPositionBraceStaysOutOfLineComment(t *testing.T) {
	t.Parallel()
	for _, src := range []string{
		"<?php\nif ($b) /*a*/// c\n{\n    echo 1;\n}\n",
		"<?php\nif ($b)/*a*/// c\n{\n    echo 1;\n}\n",
		"<?php\nif ($b) /*a*/# c\n{\n    echo 1;\n}\n",
		"<?php\nif ($b) /*a*/// c\n/*d*/{\n    echo 1;\n}\n",
	} {
		got, _ := apply(t, BracesPosition{}, src)
		for line := range strings.Lines(got) {
			if comment := strings.Index(line, "//"); comment >= 0 && strings.Contains(line[comment:], "{") {
				t.Errorf("%s got=%q", src, got)
			}
			if comment := strings.Index(line, "#"); comment >= 0 && strings.Contains(line[comment:], "{") {
				t.Errorf("%s got=%q", src, got)
			}
		}
	}
}

// PHP-CS-Fixer ends the comment before the space, so its output is no reference: the brace
// must not go past the close tag, where it would be inline HTML.
func TestBracesPositionBraceStaysBeforeCloseTag(t *testing.T) {
	t.Parallel()
	got, _ := apply(t, BracesPosition{}, "<?php\nfunction f() { // e ?>\n<b>x</b>\n<?php\n}\n")
	if want := "<?php\nfunction f()\n{ // e ?>\n<b>x</b>\n<?php\n}\n"; got != want {
		t.Fatalf("\n got=%q\nwant=%q", got, want)
	}
}
