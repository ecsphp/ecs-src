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

// The command text of a backtick literal is not code for any rule, and the
// literal is never a constant (cases checked against PHP-CS-Fixer).
func TestRulesOnBacktickLiterals(t *testing.T) {
	t.Parallel()
	cmd := "<?php\n$a = `echo STRLEN(1) TRUE a=b ECHO \"x\"   \n  next`;\n"
	yoda := YodaStyle{}.WithConfig(map[string]any{"equal": true, "identical": true}).(fixerRule)
	cases := []struct {
		name      string
		f         fixerRule
		src, want string
	}{
		{"NativeFunctionCasing leaves the command", NativeFunctionCasing{}, cmd, cmd},
		{"ConstantCase leaves the command", ConstantCase{}, cmd, cmd},
		{"LowercaseKeywords leaves the command", LowercaseKeywords{}, cmd, cmd},
		{"SingleQuote leaves the command", SingleQuote{}, cmd, cmd},
		{"BinaryOperatorSpaces leaves the command", BinaryOperatorSpaces{}, cmd, cmd},
		{"NoTrailingWhitespace leaves the command", NoTrailingWhitespace{}, cmd, cmd},
		{"ESV skips backtick literals and what they nest", ExplicitStringVariable{},
			"<?php $a = `ls $b $c->d {$e}`;\n$f = `ls {$b[\"x $c\"]}` . \"$d\";\n$g = \"{$h[`ls $i`]} $j\";\n",
			"<?php $a = `ls $b $c->d {$e}`;\n$f = `ls {$b[\"x $c\"]}` . \"{$d}\";\n$g = \"{$h[`ls $i`]} {$j}\";\n"},
		{"YodaStyle: a shell-exec is not a constant", YodaStyle{}, "<?php if (`ls` === $x) {}", "<?php if (`ls` === $x) {}"},
		{"YodaStyle yoda: a shell-exec is not a constant", yoda, "<?php if ($y == `pwd`) {}", "<?php if ($y == `pwd`) {}"},
		{"LongToShorthand over a split literal", LongToShorthandOperator{},
			"<?php\n$x = $x . `e $a`;\n$y = $y . \"e $a\";\n$z = $z . `e`;\n",
			"<?php\n$x .= `e $a`;\n$y .= \"e $a\";\n$z .= `e`;\n"},
		{"Include spaces only a constant string", Include{}, "<?php\ninclude`x`;\ninclude\"x$a\";\ninclude\"x\";\n", "<?php\ninclude`x`;\ninclude\"x$a\";\ninclude \"x\";\n"},
		{"MethodArgumentSpace: no reflow on a line inside a backtick literal",
			MethodArgumentSpace{}.WithConfig(map[string]any{"on_multiline": "ensure_fully_multiline"}).(fixerRule),
			"<?php\n$x = `a\nb` . foo($a,\n    $b);\n", "<?php\n$x = `a\nb` . foo($a,\n    $b);\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, c.f, c.src); got != c.want {
			t.Errorf("%s:\n got=%q\nwant=%q", c.name, got, c.want)
		}
	}
}
