package rules

import "testing"

// Cases ported from PHP-CS-Fixer's CombineNestedDirnameFixerTest.
func TestCombineNestedDirnamePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, CombineNestedDirname{}, []nativeCallCase{
		{"provideFixCases#0", "<?php dirname();", ""},
		{"provideFixCases#1", "<?php dirname($path);", ""},
		{"provideFixCases#2", "<?php dirname($path, 3);", ""},
		{"provideFixCases#3", "<?php dirname($path, 2);", "<?php dirname(dirname($path));"},
		{"provideFixCases#4", "<?php dirname /* a */ ( /* b */ /* c */ $path /* d */, 2);", "<?php dirname /* a */ ( /* b */ dirname( /* c */ $path) /* d */);"},
		{"provideFixCases#5", "<?php dirname($path, 3);", "<?php dirname(\\dirname(dirname($path)));"},
		{"provideFixCases#6", "<?php dirname($path, 4);", "<?php dirname(dirname($path, 3));"},
		{"provideFixCases#7", "<?php dirname($path, 4);", "<?php dirname(dirname($path), 3);"},
		{"provideFixCases#8", "<?php dirname($path, 5);", "<?php dirname(dirname($path, 2), 3);"},
		{"provideFixCases#9", "<?php dirname($path, 5);", "<?php dirname(dirname(dirname($path), 3));"},
		{"provideFixCases#10", "<?php dirname(dirname($path, $level));", ""},
		{"provideFixCases#11", "<?php dirname(\"foo/\".dirname($path));", ""},
		{"provideFixCases#12", "<?php dirname(dirname($path).$foo);", ""},
		{"provideFixCases#13", "<?php foo\\dirname(dirname($path));", ""},
		{"provideFixCases#14", "<?php dirname(foo(dirname($path, 2)), 2);", "<?php dirname(dirname(foo(dirname(dirname($path)))));"},
		{"provideFixCases#15", "<?php new dirname(dirname($path, 2));", "<?php new dirname(dirname(dirname($path)));"},
		{"provideFixCases#16", "<?php dirname($path, 3);", "<?php dirname(dirname(dirname($path, ), ));"},
		{"provideFixCases#17", "<?php dirname($path, 3);", "<?php dirname(dirname(dirname($path, ), ), );"},
		{"provideFix81Cases#0", "<?php $a = dirname(dirname(...));", ""},
	})
}

func TestCombineNestedDirnameRegressions(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, CombineNestedDirname{}, []nativeCallCase{
		{"float levels", "<?php dirname(dirname($a), 1.0);", ""},
		{"hex levels cast as PHP does", "<?php dirname($a, 1);", "<?php dirname(dirname($a), 0x2);"},
		// an integer literal too large for an int is a float (T_DNUMBER)
		{"overflowing decimal levels", "<?php dirname(dirname($b, 99999999999999999999));", ""},
		{"overflowing hex levels", "<?php dirname(dirname($b, 0xFFFFFFFFFFFFFFFF));", ""},
		{"overflowing octal levels", "<?php dirname(dirname($b, 01777777777777777777777));", ""},
		{"largest decimal levels", "<?php dirname($b, 9.2233720368548E+18);", "<?php dirname(dirname($b, 9223372036854775807));"},
		{"octal levels cast as decimal, saturating", "<?php dirname($b, 9.2233720368548E+18);", "<?php dirname(dirname($b, 0777777777777777777777));"},
		{"legacy octal levels cast as decimal", "<?php dirname($b, 18);", "<?php dirname(dirname($b, 017));"},
		{"binary levels", "<?php dirname($b, 1);", "<?php dirname(dirname($b, 0b11));"},
		{"separated levels", "<?php dirname($b, 2);", "<?php dirname(dirname($b, 1_000));"},
		{"argument with nested calls", "<?php dirname(f($a, [1, 2]), 2);", "<?php dirname(dirname(f($a, [1, 2])));"},
		{"function imported from a namespace", "<?php\nnamespace A;\nuse function B\\dirname;\ndirname(dirname($a));\n", ""},
		{"inner call namespaced", "<?php dirname(B\\dirname($a));", ""},
	})
}
