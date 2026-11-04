package lexer

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// The expected comments come from PHP's token_get_all.
func TestLineCommentEnds(t *testing.T) {
	cases := []struct{ src, comment string }{
		{"<?php // c\n$a;", "// c"},
		{"<?php // c\r\n$a;", "// c"},
		{"<?php // c\r$a;", "// c"},
		{"<?php # c  \r\n", "# c  "},
		{"<?php // c ?>", "// c "},
	}
	for _, c := range cases {
		found := false
		for _, tk := range Lex(c.src) {
			if tk.Kind == token.Comment {
				found = true
				if tk.Value != c.comment {
					t.Errorf("%q comment=%q want %q", c.src, tk.Value, c.comment)
				}
			}
		}
		if !found {
			t.Errorf("%q has no comment", c.src)
		}
	}
}

// PHP's token_get_all gives one T_YIELD_FROM here, comment included.
func TestYieldFromAcrossCommentEndingAtCarriageReturn(t *testing.T) {
	src := "<?php yield // c\r    from [];"
	for _, tk := range Lex(src) {
		if tk.Value == "yield // c\r    from" {
			return
		}
	}
	t.Errorf("no single yield from token in %q", src)
}
