package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NoUselessSprintfFixerTest.
func TestNoUselessSprintfPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, NoUselessSprintf{}, []nativeCallCase{
		{"provideFixCases#simple", "<?php echo \"bar\";", "<?php echo sprintf(\"bar\");"},
		{"provideFixCases#simple II", "<?php echo 'bar' ?>", "<?php echo sprintf('bar') ?>"},
		{"provideFixCases#simple III", "<?php echo $bar;", "<?php echo sprintf($bar);"},
		{"provideFixCases#simple IV", "<?php echo 1;", "<?php echo sprintf(1);"},
		{"provideFixCases#minimal", "<?php $foo;", "<?php sprintf($foo);"},
		{"provideFixCases#case insensitive", "<?php echo \"bar\";", "<?php echo SPRINTF(\"bar\");"},
		{"provideFixCases#nested", "<?php echo /* test */\"bar\";", "<?php echo sprintf(sprintf(sprintf(/* test */sprintf(sprintf(sprintf(\"bar\"))))));"},
		{"provideFixCases#7", "<?php namespace Foo {\n                function sprintf($foo) {\n                    echo $foo;\n                }\n            }", ""},
		{"provideFixCases#8", "<?php namespace Foo;\n                function sprintf($foo) {\n                    echo $foo;\n                }\n            ", ""},
		{"provideFixCases#9", "<?php\n                echo \\Foo\\sprintf(\"abc\");\n                echo $bar->sprintf($foo);\n                echo Bar::sprintf($foo);\n                echo sprint(...$foo);\n                echo sprint(\"%d\", 1);\n                echo sprint(\"%d%d%d\", 1, 2, 3);\n                echo sprint();\n            ", ""},
		{"provideFixCases#10", "<?php echo sprint[2](\"foo\");", ""},
		{"provideFixCases#trailing comma", "<?php echo \"bar\";", "<?php echo sprintf(\"bar\",);"},
	})
}

func TestNoUselessSprintfRegressions(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, NoUselessSprintf{}, []nativeCallCase{
		{"first-class callable", "<?php $f = sprintf(...);", ""},
		{"function declared in the namespace", "<?php\nnamespace A;\nfunction sprintf($f) {}\necho sprintf('a');\n", ""},
		{"function imported from a namespace", "<?php\nnamespace A;\nuse function B\\sprintf;\necho sprintf('a');\n", ""},
		{"in a namespace", "<?php\nnamespace A;\necho 'a';\n", "<?php\nnamespace A;\necho sprintf('a');\n"},
		{"interpolated format", `<?php echo "$a";`, `<?php echo sprintf("$a");`},
		{"heredoc format", "<?php echo <<<X\nx $y\nX;\n", "<?php echo sprintf(<<<X\nx $y\nX);\n"},
		{"call in an interpolation", `<?php echo "{$f("y")}";`, `<?php echo "{$f(sprintf("y"))}";`},
	})
}
