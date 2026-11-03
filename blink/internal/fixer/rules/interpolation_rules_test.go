package rules

import "testing"

// Rules that look at string or heredoc tokens, on interpolated literals (cases
// checked against PHP-CS-Fixer and Symplify).
func TestRulesOnInterpolatedLiterals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		f         fixerRule
		src, want string
	}{
		{"ESV run continues over {$...}", ExplicitStringVariable{}, `<?php $t = "$a{$b}$c";`, `<?php $t = "{$a}{$b}{$c}";`},
		{"ESV adjacent variables", ExplicitStringVariable{}, `<?php $s = "$a$b";`, `<?php $s = "{$a}{$b}";`},
		{"ESV nullsafe property", ExplicitStringVariable{}, `<?php $v = "$o?->p";`, `<?php $v = "{$o?->p}";`},
		{"ESV heredoc parts", ExplicitStringVariable{}, "<?php $w = <<<EOT\n  x $a[k] {$b} $c->d\n  EOT;\n", "<?php $w = <<<EOT\n  x {$a['k']} {$b} {$c->d}\n  EOT;\n"},
		{"ESV inside an interpolated expression", ExplicitStringVariable{}, `<?php $s = "{$a["k $b"]}";`, `<?php $s = "{$a["k {$b}"]}";`},
		{"SingleQuote leaves heredoc text", SingleQuote{}, "<?php $a = <<<EOT\n$x\"y\"$z\nEOT;\n", "<?php $a = <<<EOT\n$x\"y\"$z\nEOT;\n"},
		{"NoBinaryString leaves heredoc text", NoBinaryString{}, "<?php $a = <<<EOT\n$a[0]b'x'\nEOT;\n$b = b\"x $y\";\n", "<?php $a = <<<EOT\n$a[0]b'x'\nEOT;\n$b = \"x $y\";\n"},
		{"HeredocToNowdoc leaves interpolation", HeredocToNowdoc{}, "<?php $a = <<<EOT\nx $y\nEOT;\n", "<?php $a = <<<EOT\nx $y\nEOT;\n"},
		{"LambdaNotUsedImport sees ${name}", LambdaNotUsedImport{}, `<?php $f = function () use ($a, $b) { return "$x ${a}"; };`, `<?php $f = function () use ($a ) { return "$x ${a}"; };`},
		{"SpaceAfterCommaHereNowDoc after interpolated heredoc", SpaceAfterCommaHereNowDoc{}, "<?php $a = [<<<EOT\n$x\nEOT, 1];\n", "<?php $a = [<<<EOT\n$x\nEOT\n, 1];\n"},
		{"NoWhitespaceBeforeCommaInArray keeps heredoc end", NoWhitespaceBeforeCommaInArray{}.WithConfig(map[string]any{"after_heredoc": false}).(fixerRule), "<?php $a = [<<<EOT\n{$x}\nEOT , 2];", "<?php $a = [<<<EOT\n{$x}\nEOT , 2];"},
		{"TrailingComma keeps heredoc end", TrailingCommaInMultiline{}.WithConfig(map[string]any{"after_heredoc": false}).(fixerRule), "<?php $a = [\n    <<<EOT\n$x\nEOT\n];", "<?php $a = [\n    <<<EOT\n$x\nEOT\n];"},
		{"NoUselessConcat leaves string chunks", NoUselessConcatOperator{}, `<?php $a = "a\"$b" . "c";`, `<?php $a = "a\"$b" . "c";`},
	}
	for _, c := range cases {
		if got, _ := apply(t, c.f, c.src); got != c.want {
			t.Errorf("%s:\n got=%q\nwant=%q", c.name, got, c.want)
		}
	}
}
