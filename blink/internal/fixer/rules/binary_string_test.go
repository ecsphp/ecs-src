package rules

import "testing"

// The binary-string prefix is part of the string token; string rules keep it
// or drop it as PHP-CS-Fixer's do.
func TestBinaryStringPrefix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"prefix dropped", NoBinaryString{}, "<?php\n$a = b\"x\";\n$b = B'y';\n$c = b\"$v {$w}\";\n$d = b<<<EOT\nz\nEOT;\n", "<?php\n$a = \"x\";\n$b = 'y';\n$c = \"$v {$w}\";\n$d = <<<EOT\nz\nEOT;\n"},
		{"single quotes keep the prefix", SingleQuote{}, "<?php\n$a = b\"x\";\n$b = B\"y\";\n", "<?php\n$a = b'x';\n$b = B'y';\n"},
		{"nowdoc keeps the prefix", HeredocToNowdoc{}, "<?php\n$a = b<<<EOT\nz\nEOT;\n", "<?php\n$a = b<<<'EOT'\nz\nEOT;\n"},
		{"explicit variable in a binary string", ExplicitStringVariable{}, "<?php\n$a = b\"$v {$w} $x\";\n", "<?php\n$a = b\"{$v} {$w} {$x}\";\n"},
		{"concatenated binary strings stay apart", NoUselessConcatOperator{}, "<?php\n$f = b\"a\" . b\"b\";\n", "<?php\n$f = b\"a\" . b\"b\";\n"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s:\n got: %q\nwant: %q", c.name, got, c.want)
		}
	}
}
