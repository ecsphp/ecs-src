package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NoTrailingWhitespaceInStringFixerTest
// (tests/Fixer/StringNotation); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestNoTrailingWhitespaceInStringPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, NoTrailingWhitespaceInString{}, []aliasUpstreamCase{
		{"single quoted", nil,
			"<?php $a = ' foo  \r bar \r\n  \nbaz  \n  ';",
			"<?php $a = ' foo\r bar\r\n\nbaz\n  ';"},
		{"double quoted", nil,
			"<?php $a = \" foo  \r bar \r\n  \nbaz  \n  \";",
			"<?php $a = \" foo\r bar\r\n\nbaz\n  \";"},
		{"double quoted with variable", nil,
			"<?php $a = \"  $foo  \n\";",
			"<?php $a = \"  $foo\n\";"},
		{"inline html only", nil,
			" foo  \r bar \r\nbaz  \n  ",
			" foo\r bar\r\nbaz\n"},
		{"inline html around php", nil,
			"  \n<?php foo() ?>  \n  foo",
			"\n<?php foo() ?>\n  foo"},
		{"two php blocks", nil,
			"<?php foo() ?>  \n<?php foo() ?>  \n",
			"<?php foo() ?>\n<?php foo() ?>\n"},
		{"blank line after close tag", nil,
			"<?php foo() ?>\n  \nfoo",
			"<?php foo() ?>\n\nfoo"},
		{"text after close tag", nil,
			"<?php foo() ?>foo  \n",
			"<?php foo() ?>foo\n"},
		{"whitespace only", nil, " ", ""},
		{"trailing space after close tag", nil, "<?php echo 1; ?> ", "<?php echo 1; ?>"},
		{"heredoc", nil,
			"\n<?php\n$a = <<<EOD\n  foo  \nbar\n  $a \n$b\n    \n   baz  \nEOD;\n                ",
			"\n<?php\n$a = <<<EOD\n  foo\nbar\n  $a\n$b\n\n   baz\nEOD;\n                "},
		{"nowdoc", nil,
			"\n<?php\n$a = <<<'EOD'\n  foo  \nbar\n    \n   baz  \nEOD;\n                ",
			"\n<?php\n$a = <<<'EOD'\n  foo\nbar\n\n   baz\nEOD;\n                "},
		{"nowdoc indented marker", nil,
			"\n<?php\n    $a = <<<'EOD'\n      foo  \n    bar\n        \n       baz  \n    EOD;\n",
			"\n<?php\n    $a = <<<'EOD'\n      foo\n    bar\n\n       baz\n    EOD;\n"},
		{"binary string", nil,
			"<?php $a = b\"  $foo  \n\";",
			"<?php $a = b\"  $foo\n\";"},
	})
}
