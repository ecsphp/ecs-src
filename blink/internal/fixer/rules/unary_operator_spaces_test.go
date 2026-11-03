package rules

import "testing"

// TestUnaryOperatorSpacesMatchesPHPCSFixer covers the operators and contexts of
// TokensAnalyzer's unary predecessor/successor checks; the expected outputs come
// from PHP-CS-Fixer.
func TestUnaryOperatorSpacesMatchesPHPCSFixer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		rule     fixerRule
		in, want string
	}{
		{
			"only_dec_inc false",
			UnaryOperatorSpaces{Full: true},
			"<?php\nf(... $args);\n$m = & $n;\n$a = - $b;\n$c = $d - $e;\n$g = (int) - $h;\nforeach ($l as & $v) {}\nfunction & r(Foo & $x, & $y) {}\n$i = A & $j;\n$k = $o ++;\n$p = -- $q;\n$s = $t\n    ++;\n$u = !\n    $w;\n$z = $arr[0] - 1;\n$y = fn & ($q) => 1;\n$s = <<<EOT\nx\nEOT - 1;\n",
			"<?php\nf(...$args);\n$m = &$n;\n$a = -$b;\n$c = $d - $e;\n$g = (int) -$h;\nforeach ($l as &$v) {}\nfunction &r(Foo &$x, &$y) {}\n$i = A & $j;\n$k = $o++;\n$p = --$q;\n$s = $t++;\n$u = !$w;\n$z = $arr[0] - 1;\n$y = fn &($q) => 1;\n$s = <<<EOT\nx\nEOT -1;\n",
		},
		{
			"only_dec_inc true",
			UnaryOperatorSpaces{},
			"<?php\n$m = & $n;\n$k = $o ++;\n$p = -- $q;\n$s = $t\n    ++;\n",
			"<?php\n$m = & $n;\n$k = $o++;\n$p = --$q;\n$s = $t++;\n",
		},
	}
	for _, c := range cases {
		if got, _ := apply(t, c.rule, c.in); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
