package rules

import "testing"

// Cases ported from PHP-CS-Fixer's ImplodeCallFixerTest.
func TestImplodeCallPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, ImplodeCall{}, []nativeCallCase{
		{"provideFixCases#0", "<?php implode('', [1,2,3]);", ""},
		{"provideFixCases#1", "<?php implode(\"\", $foo);", ""},
		{"provideFixCases#2", "<?php implode($foo, $bar);", ""},
		{"provideFixCases#3", "<?php $arrayHelper->implode($foo);", ""},
		{"provideFixCases#4", "<?php ArrayHelper::implode($foo);", ""},
		{"provideFixCases#5", "<?php ArrayHelper\\implode($foo);", ""},
		{"provideFixCases#6", "<?php define(\"implode\", \"foo\"); implode; bar($baz);", ""},
		{"provideFixCases#7", "<?php implode(\"\", $foo);", "<?php implode($foo, \"\");"},
		{"provideFixCases#8", "<?php \\implode(\"\", $foo);", "<?php \\implode($foo, \"\");"},
		{"provideFixCases#9", "<?php implode(\"Lorem ipsum dolor sit amet\", $foo);", "<?php implode($foo, \"Lorem ipsum dolor sit amet\");"},
		{"provideFixCases#10", "<?php implode('', $foo);", "<?php implode($foo);"},
		{"provideFixCases#11", "<?php IMPlode(\"\", $foo);", "<?php IMPlode($foo, \"\");"},
		{"provideFixCases#12", "<?php implode(\"\",$foo);", "<?php implode($foo,\"\");"},
		{"provideFixCases#13", "<?php implode(\"\", $weirdStuff[mt_rand($min, getMax()) + 200]);", "<?php implode($weirdStuff[mt_rand($min, getMax()) + 200], \"\");"},
		{"provideFixCases#14", "<?php\n                implode(\n                    \"\",\n                    $foo\n                );", "<?php\n                implode(\n                    $foo,\n                    \"\"\n                );"},
		{"provideFixCases#15", "<?php\n                implode(\n                    '', $foo\n                );", "<?php\n                implode(\n                    $foo\n                );"},
		{"provideFixCases#16", "<?php\nimplode(# 1\n\"\"/* 2.1 */,# 2.2\n$foo# 3\n);", "<?php\nimplode(# 1\n$foo/* 2.1 */,# 2.2\n\"\"# 3\n);"},
		{"provideFixCases#17", "<?php\nimplode(# 1\n# 2\n'', $foo# 3\n# 4\n)# 5\n;", "<?php\nimplode(# 1\n# 2\n$foo# 3\n# 4\n)# 5\n;"},
		{"provideFixCases#18", "<?php\nimplode('', $a);implode('', $a);implode('', $a);implode('', $a);implode('', $a);implode('', $a);\n// comment for testing\nimplode('', $a);implode('', $a);implode('', $a);implode('', $a);implode('', $a);implode('', $a);\n", "<?php\nimplode($a);implode($a);implode($a);implode($a);implode($a);implode($a);\n// comment for testing\nimplode($a);implode($a);implode($a);implode($a);implode($a);implode($a);\n"},
		{"provideFixCases#19", "<?php implode(\"\", $foo, );", "<?php implode($foo, \"\", );"},
		{"provideFixCases#20", "<?php implode('', $foo, );", "<?php implode($foo, );"},
		{"provideFixCases#21", "<?php\n                implode(\n                    \"\",\n                    $foo,\n                );", "<?php\n                implode(\n                    $foo,\n                    \"\",\n                );"},
	})
}

func TestImplodeCallRegressions(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, ImplodeCall{}, []nativeCallCase{
		// upstream throws on an implode swapped inside the first argument of
		// another swap ("Must not add empty token"), and ECS leaves the file as is
		{"swap nested in a swapped first argument", "<?php implode(implode($b, \",\"), \";\");", ""},
		{"swap nested deeper in a swapped first argument", "<?php\nimplode($a . implode($b, \",\"), \";\");\nimplode($c);\n", ""},
		{"swap nested in a call in a swapped first argument", "<?php implode(foo(implode($b, \",\")), \";\");", ""},
		{"insert nested in a swapped first argument", "<?php implode(\";\", implode('', $b));", "<?php implode(implode($b), \";\");"},
		{"swap nested in a second argument", "<?php implode(\",\", implode(\",\", $b));", "<?php implode(\",\", implode($b, \",\"));"},
		{"swaps side by side", "<?php implode(\",\", $x) . implode(\";\", $y);", "<?php implode($x, \",\") . implode($y, \";\");"},
		// a binary string is one PHP string token
		{"binary string swapped", "<?php implode(b',', $a);", "<?php implode($a, b',');"},
		{"binary string first", "<?php implode(b',', $a);", ""},
		// an interpolated string is not a T_CONSTANT_ENCAPSED_STRING
		{"interpolated second argument", `<?php implode($a, "$sep");`, ""},
		{"heredoc second argument", "<?php implode($a, <<<X\n,\nX);\n", ""},
		// only the leading string token of the second argument moves
		{"concatenated second argument", "<?php implode(', ', $a . $b);", "<?php implode($a, ', ' . $b);"},
		{"function declared in the namespace", "<?php\nnamespace A;\nfunction implode($a) {}\nimplode($a);\n", ""},
		{"function imported from a namespace", "<?php\nnamespace A;\nuse function B\\implode;\nimplode($a);\n", ""},
		{"global function imported", "<?php\nnamespace A;\nuse function implode;\nimplode('', $a);\n", "<?php\nnamespace A;\nuse function implode;\nimplode($a);\n"},
		{"first-class callable", "<?php $f = implode(...);", ""},
		{"three arguments", "<?php implode($a, ',', $b);", ""},
		{"call in an interpolation", `<?php $a = "{$f(implode(',', $a))}";`, `<?php $a = "{$f(implode($a, ','))}";`},
		{"interpolated first argument", `<?php implode("{$s}", $a);`, ""},
	})
}
