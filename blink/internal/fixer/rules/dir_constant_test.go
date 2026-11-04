package rules

import "testing"

// Cases ported from PHP-CS-Fixer's DirConstantFixerTest.
func TestDirConstantPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, DirConstant{}, []nativeCallCase{
		{"provideFixCases#0", "<?php $x = \"dirname\";", ""},
		{"provideFixCases#1", "<?php $x = dirname(__FILE__.\".dist\");", ""},
		{"provideFixCases#2", "<?php $x = ClassA::dirname(__FILE__);", ""},
		{"provideFixCases#3", "<?php $x = ScopeA\\dirname(__FILE__);", ""},
		{"provideFixCases#4", "<?php $x = namespace\\dirname(__FILE__);", ""},
		{"provideFixCases#5", "<?php $x = $object->dirname(__FILE__);", ""},
		{"provideFixCases#6", "<?php $x = new \\dirname(__FILE__);", ""},
		{"provideFixCases#7", "<?php $x = new dirname(__FILE__);", ""},
		{"provideFixCases#8", "<?php $x = new ScopeB\\dirname(__FILE__);", ""},
		{"provideFixCases#9", "<?php dirnameSmth(__FILE__);", ""},
		{"provideFixCases#10", "<?php smth_dirname(__FILE__);", ""},
		{"provideFixCases#11", "<?php \"SELECT ... dirname(__FILE__) ...\";", ""},
		{"provideFixCases#12", "<?php \"SELECT ... DIRNAME(__FILE__) ...\";", ""},
		{"provideFixCases#13", "<?php \"test\" . \"dirname\" . \"in concatenation\";", ""},
		{"provideFixCases#14", "<?php $x = dirname(__DIR__);", "<?php $x = dirname(dirname(__FILE__));"},
		{"provideFixCases#15", "<?php $x = __DIR__;", "<?php $x = dirname(__FILE__);"},
		{"provideFixCases#16", "<?php $x =   /* A */ __DIR__     /* B */;", "<?php $x = dirname  (  /* A */ __FILE__  )   /* B */;"},
		{"provideFixCases#17", "<?php $x = __DIR__;", "<?php $x = \\dirname(__FILE__);"},
		{"provideFixCases#18", "<?php $x = __DIR__.\".dist\";", "<?php $x = dirname(__FILE__).\".dist\";"},
		{"provideFixCases#19", "<?php $x = __DIR__.\".dist\";", "<?php $x = \\dirname(__FILE__).\".dist\";"},
		{"provideFixCases#20", "<?php $x = /* 0 *//* 1 */ /** x2*//*3*//** 4*/__DIR__/**5*//*xx*/;", "<?php $x = /* 0 */dirname/* 1 */ /** x2*/(/*3*//** 4*/__FILE__/**5*/)/*xx*/;"},
		{"provideFixCases#21", "<?php\n                interface Test\n                {\n                    public function dirname($a);\n                }", ""},
		{"provideFixCases#22", "<?php\n                interface Test\n                {\n                    public function &dirname($a);\n                }", ""},
		{"provideFixCases#23", "<?php echo __DIR__\n?>", "<?php echo dirname\n(\n__FILE__\n)\n?>"},
		{"provideFixCases#24", "<?php echo __DIR__/*1*/\n?>", "<?php echo dirname\n(\n__FILE__/*1*/\n)\n?>"},
		{"provideFixCases#25", "<?php $x =\n__DIR__\n\n;", "<?php $x =\ndirname\n\n(\n\n    __FILE__\n\n)\n\n;"},
		{"provideFixCases#26", "<?php $x = __DIR__;", "<?php $x = \\dirname(\n                    __FILE__                     \n                );"},
		{"provideFixCases#27", "<?php\n                    $x = dirname(dirname(\"a\".__FILE__));\n                    $x = dirname(dirname(__FILE__.\"a\"));\n                    $x = dirname(dirname(\"a\".__FILE__.\"a\"));\n                ", ""},
		{"provideFixCases#28", "<?php $x = __DIR__.\".dist\";", "<?php $x = dirname(__FILE__,   ).\".dist\";"},
		{"provideFixCases#29", "<?php $x = __DIR__/* a */  /* b */  .\".dist\";", "<?php $x = \\dirname(__FILE__/* a */,  /* b */)  .\".dist\";"},
		{"provideFixCases#30", "<?php $x = __DIR__;", "<?php $x = \\dirname(\n                    __FILE__   ,                     \n                );"},
	})
}

func TestDirConstantRegressions(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, DirConstant{}, []nativeCallCase{
		{"lowercase magic constant", "<?php $a = __DIR__;", "<?php $a = dirname(__file__);"},
		{"levels argument", "<?php $a = dirname(__FILE__, 2);", ""},
		{"call in an interpolation", `<?php $a = "{$f(__DIR__)}";`, `<?php $a = "{$f(dirname(__FILE__))}";`},
		{"function imported from a namespace", "<?php\nnamespace A;\nuse function B\\dirname;\n$a = dirname(__FILE__);\n", ""},
		{"in a namespace", "<?php\nnamespace A;\n$a = __DIR__;\n", "<?php\nnamespace A;\n$a = \\dirname(__FILE__);\n"},
	})
}
