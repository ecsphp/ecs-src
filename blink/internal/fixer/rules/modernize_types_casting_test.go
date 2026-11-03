package rules

import "testing"

// Cases ported from PHP-CS-Fixer's ModernizeTypesCastingFixerTest.
func TestModernizeTypesCastingPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, ModernizeTypesCasting{}, []nativeCallCase{
		{"provideFixCases#0", "<?php $x = \"intval\";", ""},
		{"provideFixCases#1", "<?php $x = ClassA::intval(mt_rand(0, 100));", ""},
		{"provideFixCases#2", "<?php $x = ScopeA\\intval(mt_rand(0, 100));", ""},
		{"provideFixCases#3", "<?php $x = namespace\\intval(mt_rand(0, 100));", ""},
		{"provideFixCases#4", "<?php $x = $object->intval(mt_rand(0, 100));", ""},
		{"provideFixCases#5", "<?php $x = new \\intval(mt_rand(0, 100));", ""},
		{"provideFixCases#6", "<?php $x = new intval(mt_rand(0, 100));", ""},
		{"provideFixCases#7", "<?php $x = new ScopeB\\intval(mt_rand(0, 100));", ""},
		{"provideFixCases#8", "<?php intvalSmth(mt_rand(0, 100));", ""},
		{"provideFixCases#9", "<?php smth_intval(mt_rand(0, 100));", ""},
		{"provideFixCases#10", "<?php \"SELECT ... intval(mt_rand(0, 100)) ...\";", ""},
		{"provideFixCases#11", "<?php \"test\" . \"intval\" . \"in concatenation\";", ""},
		{"provideFixCases#12", "<?php $x = intval($x, 16);", ""},
		{"provideFixCases#13", "<?php $x = intval($x, $options[\"base\"]);", ""},
		{"provideFixCases#14", "<?php $x = intval($x, $options->get(\"base\", 16));", ""},
		{"provideFixCases#15", "<?php $x = (int) $x;", "<?php $x = intval($x);"},
		{"provideFixCases#16", "<?php $x = (float) $x;", "<?php $x = floatval($x);"},
		{"provideFixCases#17", "<?php $x = (float) $x;", "<?php $x = doubleval($x);"},
		{"provideFixCases#18", "<?php $x = (string) $x;", "<?php $x = strval($x);"},
		{"provideFixCases#19", "<?php $x = (bool) $x;", "<?php $x = boolval   (  $x  );"},
		{"provideFixCases#20", "<?php $x = (int) (mt_rand(0, 100));", "<?php $x = intval(mt_rand(0, 100));"},
		{"provideFixCases#21", "<?php $x = (int) (mt_rand(0, 100));", "<?php $x = \\intval(mt_rand(0, 100));"},
		{"provideFixCases#22", "<?php $x = (int) (mt_rand(0, 100)).\".dist\";", "<?php $x = intval(mt_rand(0, 100)).\".dist\";"},
		{"provideFixCases#23", "<?php $x = (int) (mt_rand(0, 100)).\".dist\";", "<?php $x = \\intval(mt_rand(0, 100)).\".dist\";"},
		{"provideFixCases#24", "<?php $x =\n(int) (\n    mt_rand\n    (\n        0, 100\n    )\n\n)\n\n;", "<?php $x =\nintval\n\n(\n    mt_rand\n    (\n        0, 100\n    )\n\n)\n\n;"},
		{"provideFixCases#25", "<?php\n\nclass overridesIntval\n{\n    public function intval($x)\n    {\n        return (int) $x;\n    }\n\n    public function usesInval()\n    {\n        // that's why it is risky\n        return (int) (mt_rand(0, 100));\n    }\n}", "<?php\n\nclass overridesIntval\n{\n    public function intval($x)\n    {\n        return \\intval($x);\n    }\n\n    public function usesInval()\n    {\n        // that's why it is risky\n        return intval(mt_rand(0, 100));\n    }\n}"},
		{"provideFixCases#26", "<?php $a = (string) ($b . $c);", "<?php $a = strval($b . $c);"},
		{"provideFixCases#27", "<?php $x = /**/(int) /**/ /** x*/(/**//** */mt_rand(0, 100)/***/)/*xx*/;", "<?php $x = /**/intval/**/ /** x*/(/**//** */mt_rand(0, 100)/***/)/*xx*/;"},
		{"provideFixCases#28", "<?php $x = (string) ((int) ((int) $x + (float) $x));", "<?php $x = strval(intval(intval($x) + floatval($x)));"},
		{"provideFixCases#29", "<?php intval();intval(1,2,3);", ""},
		{"provideFixCases#30", "<?php\n                interface Test\n                {\n                    public function floatval($a);\n                    public function &doubleval($a);\n                }", ""},
		{"provideFixCases#31", "<?php $foo = ((int) $x)**2;", "<?php $foo = intval($x)**2;"},
		{"provideFixCases#32", "<?php $foo = ((string) $x)[0];", "<?php $foo = strval($x)[0];"},
		{"provideFixCases#33", "<?php $foo = ((string) ($x + $y))[0];", "<?php $foo = strval($x + $y)[0];"},
		{"provideFixCases#34", "<?php $a = (int) $b;", "<?php $a = intval($b, );"},
		{"provideFixCases#35", "<?php $a = (int) $b;", "<?php $a = intval($b , );"},
		{"provideFixCases#36", "<?php $a = (string) ($b . $c);", "<?php $a = strval($b . $c, );"},
		{"provideFix81Cases#0", "<?php $x = intval(...);", ""},
	})
}

func TestModernizeTypesCastingRegressions(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, ModernizeTypesCasting{}, []nativeCallCase{
		// one PHP token each: no parentheses kept
		{"binary string", "<?php $a = (int) b'1';", "<?php $a = intval(b'1');"},
		{"magic constant", "<?php $a = (string) __LINE__;", "<?php $a = strval(__LINE__);"},
		// several PHP tokens each: parentheses kept
		{"interpolated string", `<?php $a = (int) ("$b");`, `<?php $a = intval("$b");`},
		{"heredoc", "<?php $a = (int) (<<<X\n1\nX);\n", "<?php $a = intval(<<<X\n1\nX);\n"},
		{"argument with a comment", "<?php $a = (int) ($b /* c */);", "<?php $a = intval($b /* c */);"},
		{"nested", "<?php $a = (int) ((string) $b);", "<?php $a = intval(strval($b));"},
		{"function imported from a namespace", "<?php\nnamespace A;\nuse function B\\intval;\n$a = intval($b);\n", ""},
		{"first-class callable", "<?php $f = intval(...);", ""},
	})
}
