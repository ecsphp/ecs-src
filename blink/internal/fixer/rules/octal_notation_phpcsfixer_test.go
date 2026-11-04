package rules

import "testing"

// Cases ported from PHP-CS-Fixer's OctalNotationFixerTest (PHP 8.1+).
func TestOctalNotationPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, OctalNotation{}, []aliasUpstreamCase{
		{"leave alone", nil,
			"<?php\n$a = 0;\n$b = '0';\n$foo = 0b01;\n$foo = 0x01;\n$foo = 0B01;\n$foo = 0X01;\n$foo = 0b0;\n$foo = 0x0;\n$foo = 0B0;\n$foo = 0X0;\n$foo = 1;\n$foo = 10;",
			"<?php\n$a = 0;\n$b = '0';\n$foo = 0b01;\n$foo = 0x01;\n$foo = 0B01;\n$foo = 0X01;\n$foo = 0b0;\n$foo = 0x0;\n$foo = 0B0;\n$foo = 0X0;\n$foo = 1;\n$foo = 10;"},
		{"zeros", nil, "<?php $foo = 00000;", "<?php $foo = 0o0000;"},
		{"two literals", nil, "<?php\n$foo = 0123;\n$foo = 01;", "<?php\n$foo = 0o123;\n$foo = 0o1;"},
		{"with separator", nil, "<?php $foo = 01_234;", "<?php $foo = 0o1_234;"},
		{"plus sign", nil, "<?php $foo = +01_234;", "<?php $foo = +0o1_234;"},
		{"minus sign separators", nil, "<?php $foo = -0_123_456;", "<?php $foo = -0o123_456;"},
	})
}
