package rules

import "testing"

// Cases ported from PHP-CS-Fixer's FopenFlagOrderFixerTest.
func TestFopenFlagOrderPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, FopenFlagOrder{}, []nativeCallCase{
		{"provideFixCases#most simple fix case", "<?php\n                    $a = fopen($foo, 'rw+b');\n                ", "<?php\n                    $a = fopen($foo, 'brw+');\n                "},
		{"provideFixCases#\"fopen\" casing insensitive", "<?php\n                    $a = \\FOPen($foo, \"cr+w+b\");\n                    $a = \\FOPEN($foo, \"crw+b\");\n                ", "<?php\n                    $a = \\FOPen($foo, \"bw+r+c\");\n                    $a = \\FOPEN($foo, \"bw+rc\");\n                "},
		{"provideFixCases#comments around flags", "<?php\n                    $a = fopen($foo,/*0*/'rb'/*1*/);\n                ", "<?php\n                    $a = fopen($foo,/*0*/'br'/*1*/);\n                "},
		{"provideFixCases#binary string", "<?php\n                    $a = \\fopen($foo, b\"cr+w+b\");\n                    $b = \\fopen($foo, B\"crw+b\");\n                    $c = \\fopen($foo, b'cr+w+b');\n                    $d = \\fopen($foo, B'crw+b');\n                ", "<?php\n                    $a = \\fopen($foo, b\"bw+r+c\");\n                    $b = \\fopen($foo, B\"bw+rc\");\n                    $c = \\fopen($foo, b'bw+r+c');\n                    $d = \\fopen($foo, B'bw+rc');\n                "},
		{"provideFixCases#common typos", "<?php\n                     $a = fopen($a, \"b+r\");\n                     $b = fopen($b, 'b+w');\n                ", ""},
		{"provideFixCases#5", "<?php\n                    $a = fopen($foo, 'rw+t');\n                ", "<?php\n                    $a = fopen($foo, 'trw+');\n                "},
		{"provideFixCases#6", "<?php\n                    $a = \\fopen($foo, 'rw+tb');\n                ", "<?php\n                    $a = \\fopen($foo, 'btrw+');\n                "},
		{"provideFixCases#single flag", "<?php\n                    $a = fopen($foo, \"r\");\n                    $a = fopen($foo, \"r+\");\n                ", ""},
		{"provideFixCases#not simple flags", "<?php\n                    $a = fopen($foo, \"br+\".$a);\n                ", ""},
		{"provideFixCases#wrong # of arguments", "<?php\n                    $b = \\fopen(\"br+\");\n                    $c = fopen($foo, \"bw+\", 1, 2 , 3);\n                ", ""},
		{"provideFixCases#\"flags\" is too long (must be overridden)", "<?php\n                    $d = fopen($foo, \"r+w+a+x+c+etbX\");\n                ", ""},
		{"provideFixCases#static method call", "<?php\n                    $e = A::fopen($foo, \"bw+\");\n                ", ""},
		{"provideFixCases#method call", "<?php\n                    $f = $b->fopen($foo, \"br+\");\n                ", ""},
		{"provideFixCases#comments, PHPDoc and literal", "<?php\n                    // fopen($foo, \"brw\");\n                    /* fopen($foo, \"brw\"); */\n                    echo(\"fopen($foo, \\\"brw\\\")\");\n                ", ""},
		{"provideFixCases#invalid flag values", "<?php\n                    $a = fopen($foo, '');\n                    $a = fopen($foo, 'x'); // ok but should not mark collection as changed\n                    $a = fopen($foo, 'k');\n                    $a = fopen($foo, 'kz');\n                    $a = fopen($foo, 'k+');\n                    $a = fopen($foo, '+k');\n                    $a = fopen($foo, 'xc++');\n                    $a = fopen($foo, 'w+r+r+');\n                    $a = fopen($foo, '+brw+');\n                    $a = fopen($foo, 'b+rw');\n                    $a = fopen($foo, 'bbrw+');\n                    $a = fopen($foo, 'brw++');\n                    $a = fopen($foo, '++brw');\n                    $a = fopen($foo, 'ybrw+');\n                    $a = fopen($foo, 'rr');\n                    $a = fopen($foo, 'ロ');\n                    $a = fopen($foo, 'ロ+');\n                    $a = \\fopen($foo, 'rロ');\n                    $a = \\fopen($foo, 'w+ロ');\n                ", ""},
	})
}

func TestFopenFlagOrderRegressions(t *testing.T) {
	t.Parallel()
	runNativeCallCases(t, FopenFlagOrder{}, []nativeCallCase{
		{"binary string mode", "<?php fopen($f, b'r+b');", "<?php fopen($f, b'br+');"},
		{"interpolated mode", `<?php fopen($f, "b$m");`, ""},
		{"concatenated mode", "<?php fopen($f, 'b' . 'r');", ""},
		{"commented mode", "<?php fopen($f, /* m */ 'rb');", "<?php fopen($f, /* m */ 'br');"},
		{"function imported from a namespace", "<?php\nnamespace A;\nuse function B\\fopen;\nfopen($f, 'br');\n", ""},
	})
}
