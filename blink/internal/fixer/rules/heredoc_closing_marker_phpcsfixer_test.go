package rules

import "testing"

// Cases ported from PHP-CS-Fixer's HeredocClosingMarkerFixerTest
// (tests/Fixer/StringNotation); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestHeredocClosingMarkerPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, HeredocClosingMarker{}, []aliasUpstreamCase{
		{"heredoc", nil,
			"<?php $a = <<<TEST\nxxx EOD xxx\nTEST;",
			"<?php $a = <<<EOD\nxxx EOD xxx\nEOD;"},
		{"nowdoc", nil,
			"<?php $a = <<<'TEST'\nxxx EOD xxx\nTEST;",
			"<?php $a = <<<'EOD'\nxxx EOD xxx\nEOD;"},
		{"custom preferred closing marker", map[string]any{"closing_marker": "EOF"},
			"<?php $a = <<<TEST\nxxx\nTEST;",
			"<?php $a = <<<EOF\nxxx\nEOF;"},
		{"explicit style", map[string]any{"explicit_heredoc_style": true},
			"<?php $a = <<<TEST\nxxx\nTEST;\n$b = <<<\"TEST\"\nxxx2\nTEST;\n$b = <<<'TEST'\nxxx3\nTEST;",
			"<?php $a = <<<\"EOD\"\nxxx\nEOD;\n$b = <<<\"EOD\"\nxxx2\nEOD;\n$b = <<<'EOD'\nxxx3\nEOD;"},
		{"b prefix", nil,
			"<?php $a = b<<<TEST\nxxx EOD xxx\nTEST;",
			"<?php $a = b<<<EOD\nxxx EOD xxx\nEOD;"},
		{"B prefix", nil,
			"<?php $a = B<<<TEST\nxxx EOD xxx\nTEST;",
			"<?php $a = B<<<EOD\nxxx EOD xxx\nEOD;"},
		{"reserved marker kept", nil,
			"<?php $a = <<<PHP\nxxx\nPHP;",
			"<?php $a = <<<PHP\nxxx\nPHP;"},
		{"reserved marker different case", nil,
			"<?php $a = <<<php\nxxx\nphp;\n$a = <<<Php\nPhp;",
			"<?php $a = <<<PHP\nxxx\nPHP;\n$a = <<<PHP\nPHP;"},
		{"reserved custom marker", map[string]any{"reserved_closing_markers": []any{"Žlutý"}},
			"<?php $a = <<<Žlutý\nxxx\nŽlutý;\n$aNormCase = <<<ŽluTý\nxxx\nŽluTý;\n$aNormCase = <<<ŽLUTÝ\nxxx\nŽLUTÝ;\n$b = <<<Žlutý2\nxxx2\nŽlutý2;\n$c = <<<PHP\nxxx3\nPHP;",
			"<?php $a = <<<Žlutý\nxxx\nŽlutý;\n$aNormCase = <<<Žlutý\nxxx\nŽlutý;\n$aNormCase = <<<Žlutý\nxxx\nŽlutý;\n$b = <<<EOD\nxxx2\nEOD;\n$c = <<<EOD\nxxx3\nEOD;"},
		{"colliding reserved recovery", nil,
			"<?php\n$a = <<<CSS_\n    CSS_;\n$a = <<<CSS__\n    CSS__;\n$a = <<<CSS__\n    CSS\n    CSS__;\n$a = <<<CSS__\n    CSS_\n    CSS__;",
			"<?php\n$a = <<<CSS\n    CSS;\n$a = <<<CSS\n    CSS;\n$a = <<<CSS_\n    CSS\n    CSS_;\n$a = <<<CSS\n    CSS_\n    CSS;"},
		{"content starts with preferred marker", nil,
			"<?php $a = <<<TEST\nEOD xxx\nTEST;",
			"<?php $a = <<<EOD_\nEOD xxx\nEOD_;"},
		{"content starts with whitespace and preferred marker", nil,
			"<?php $a = <<<TEST\n EOD xxx\nTEST;",
			"<?php $a = <<<EOD_\n EOD xxx\nEOD_;"},
		{"content starts with preferred marker and single quote", nil,
			"<?php $a = <<<TEST\nEOD'\nTEST;",
			"<?php $a = <<<EOD_\nEOD'\nEOD_;"},
		{"content starts with preferred marker and semicolon", nil,
			"<?php $a = <<<TEST\nEOD;\nTEST;",
			"<?php $a = <<<EOD_\nEOD;\nEOD_;"},
		{"content ends with preferred marker", nil,
			"<?php $a = <<<TEST\nxxx EOD\nTEST;",
			"<?php $a = <<<EOD\nxxx EOD\nEOD;"},
		{"interpolated heredoc", nil,
			"<?php $a = <<<TEST\nline $x end\nTEST;",
			"<?php $a = <<<EOD\nline $x end\nEOD;"},
	})
}
