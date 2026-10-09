package rules

import "testing"

// Cases ported from php-cs-fixer MethodArgumentSpaceFixerTest (4-space, LF only).
// A few upstream cases are intentionally omitted: #42 (php-cs-fixer collapses a
// blank line between reflowed arguments, but mautic parity needs them kept),
// #38/#54 (args delimited only by "#" line comments - pathological input no real
// code produces), and #97 (inline-HTML indent blink cannot read across the tag).
func TestMethodArgumentSpacePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  map[string]any
		src  string
		want string
	}{
		{"provideFixCases#0", map[string]any{"on_multiline": "ensure_fully_multiline"}, "<?php\n// space \n$var1 = $a->some_method(\n    $var2);\n\n// space \n$var2 = some_function(\n    $var2);\n\n    // space     \n    $var2a = $z[1](\n        $var2a\n    );\n    \n    $var3 = function(  $a , $b  ) { };\n", "<?php\n// space \n$var1 = $a->some_method(\n    $var2\n);\n\n// space \n$var2 = some_function(\n    $var2\n);\n\n    // space     \n    $var2a = $z[1](\n        $var2a\n    );\n    \n    $var3 = function(  $a, $b  ) { };\n"},
		{"provideFixCases#1", map[string]any{"on_multiline": "ensure_fully_multiline"}, "<?php\nfoo($x,\nbar($foo,\n$bar));\n", "<?php\nfoo(\n    $x,\n    bar(\n        $foo,\n        $bar\n    )\n);\n"},
		{"provideFixCases#2", map[string]any{"on_multiline": "ensure_fully_multiline"}, "<?php\nfoo($x,\nbar($foo,\nbaz($bar,\n$baz)));\n", "<?php\nfoo(\n    $x,\n    bar(\n        $foo,\n        baz(\n            $bar,\n            $baz\n        )\n    )\n);\n"},
		{"provideFixCases#3", nil, "<?php xyz(\"\",\"\",\"\",\"\");", "<?php xyz(\"\", \"\", \"\", \"\");"},
		{"provideFixCases#4", nil, "<?php function xyz($a=10,$b=20,$c=30) {}", "<?php function xyz($a=10, $b=20, $c=30) {}"},
		{"provideFixCases#5", nil, "<?php function xyz($a=10,         $b=20 , $c=30) {}", "<?php function xyz($a=10, $b=20, $c=30) {}"},
		{"provideFixCases#6", map[string]any{"keep_multiple_spaces_after_comma": true}, "<?php function xyz($a=10,         $b=20 , $c=30) {}", "<?php function xyz($a=10,         $b=20, $c=30) {}"},
		{"provideFixCases#7", nil, "<?php xyz($a=10 ,$b=20,$c=30);", "<?php xyz($a=10, $b=20, $c=30);"},
		{"provideFixCases#8", nil, "<?php xyz($a=10,$b=20 ,$this->foo() ,$c=30);", "<?php xyz($a=10, $b=20, $this->foo(), $c=30);"},
		{"provideFixCases#9", nil, "<?php xyz($a=10 , $b=20 ,          $c=30);", "<?php xyz($a=10, $b=20, $c=30);"},
		{"provideFixCases#10", map[string]any{"keep_multiple_spaces_after_comma": true}, "<?php xyz($a=10 , $b=20 ,          $c=30);", "<?php xyz($a=10, $b=20,          $c=30);"},
		{"provideFixCases#11", nil, "<?php xyz($a=10 , $b=20 ,\t $c=30);", "<?php xyz($a=10, $b=20, $c=30);"},
		{"provideFixCases#12", nil, "<?php xyz($a=10,$b=20 ,         $this->foo() ,$c=30);", "<?php xyz($a=10, $b=20, $this->foo(), $c=30);"},
		{"provideFixCases#13", map[string]any{"keep_multiple_spaces_after_comma": true}, "<?php xyz($a=10,$b=20 ,         $this->foo() ,$c=30);", "<?php xyz($a=10, $b=20,         $this->foo(), $c=30);"},
		{"provideFixCases#14", nil, "<?php new Foo($a=10,$b=20 ,$this->foo() ,$c=30);", "<?php new Foo($a=10, $b=20, $this->foo(), $c=30);"},
		{"provideFixCases#15", nil, "<?php new Foo($a=10 , $b=20 ,          $c=30);", "<?php new Foo($a=10, $b=20, $c=30);"},
		{"provideFixCases#16", map[string]any{"keep_multiple_spaces_after_comma": true}, "<?php new Foo($a=10 , $b=20 ,          $c=30);", "<?php new Foo($a=10, $b=20,          $c=30);"},
		{"provideFixCases#17", nil, "<?php new class ($a=10,$b=20 ,$this->foo() ,$c=30) {};", "<?php new class ($a=10, $b=20, $this->foo(), $c=30) {};"},
		{"provideFixCases#18", nil, "<?php new class ($a=10 , $b=20 ,          $c=30) extends Foo {};", "<?php new class ($a=10, $b=20, $c=30) extends Foo {};"},
		{"provideFixCases#19", map[string]any{"keep_multiple_spaces_after_comma": true}, "<?php new class ($a=10 , $b=20 ,          $c=30) {};", "<?php new class ($a=10, $b=20,          $c=30) {};"},
		{"provideFixCases#20", nil, "<?php list($a, $b,, ,$c) = foo();", "<?php list($a, $b, , , $c) = foo();"},
		{"provideFixCases#21", nil, "<?php list($a, $b,,    ,$c) = foo();", "<?php list($a, $b, , , $c) = foo();"},
		{"provideFixCases#22", map[string]any{"keep_multiple_spaces_after_comma": true}, "<?php list($a, $b,,    ,$c) = foo();", "<?php list($a, $b, ,    , $c) = foo();"},
		{"provideFixCases#23", nil, "<?php array(10 , 20 ,30); $foo = [ 10,50 , 60 ] ?>", "<?php array(10 , 20 ,30); $foo = [ 10,50 , 60 ] ?>"},
		{"provideFixCases#24", nil, "<?php list($path, $mode,) = foo();", "<?php list($path, $mode, ) = foo();"},
		{"provideFixCases#25", nil, "<?php\nlist(\n    $a   ,\n    $b  ,\n) = foo();\n", "<?php\nlist(\n    $a,\n    $b,\n) = foo();\n"},
		{"provideFixCases#26", nil, "<?php xyz($a=10,    /*comment1*/ $b=2000,/*comment2*/ $c=30);", "<?php xyz($a=10, /*comment1*/ $b=2000, /*comment2*/ $c=30);"},
		{"provideFixCases#27", map[string]any{"keep_multiple_spaces_after_comma": true}, "<?php xyz($a=10,    /*comment1*/ $b=2000,/*comment2*/ $c=30);", "<?php xyz($a=10,    /*comment1*/ $b=2000, /*comment2*/ $c=30);"},
		{"provideFixCases#28", nil, "<?php if (1) {\n                xyz(\n                    $a=10 ,\n                    $b=20,\n                    $c=30\n                );\n                }", "<?php if (1) {\n                xyz(\n                    $a=10,\n                    $b=20,\n                    $c=30\n                );\n                }"},
		{"provideFixCases#29", nil, "<?php if (1) {\n                new class (\n                    $a=10 ,\n                $b=20,$c=30) {};\n                }", "<?php if (1) {\n                new class (\n                    $a=10,\n                    $b=20,\n                    $c=30\n                ) {};\n                }"},
		{"provideFixCases#30", nil, "<?php fnc(1,array(2, func2(6,    7) ,4),    5);", "<?php fnc(1, array(2, func2(6, 7) ,4), 5);"},
		{"provideFixCases#31", map[string]any{"keep_multiple_spaces_after_comma": true}, "<?php fnc(1,array(2, func2(6,    7) ,4),    5);", "<?php fnc(1, array(2, func2(6,    7) ,4),    5);"},
		{"provideFixCases#32", nil, "<?php fnc(1, array(2, 3 ,4), 5);", "<?php fnc(1, array(2, 3 ,4), 5);"},
		{"provideFixCases#33", nil, "<?php\n                    array(\n                        10 ,\n                        20,\n                        30\n                    );", "<?php\n                    array(\n                        10 ,\n                        20,\n                        30\n                    );"},
		{"provideFixCases#34", nil, "<?php\n    $foo = [\"a\"=>\"apple\", \"b\"=>\"bed\" ,\"c\"=>\"car\"];\n    $bar = [\"a\" ,\"b\" ,\"c\"];\n    ", "<?php\n    $foo = [\"a\"=>\"apple\", \"b\"=>\"bed\" ,\"c\"=>\"car\"];\n    $bar = [\"a\" ,\"b\" ,\"c\"];\n    "},
		{"provideFixCases#35", nil, "<?php if (1) {\n    $this->foo(\n        <<<EOTXTa\n    heredoc\nEOTXTa\n        ,\n        <<<'EOTXTb'\n    nowdoc\nEOTXTb\n        ,\n        'foo'\n    );\n}", "<?php if (1) {\n    $this->foo(\n        <<<EOTXTa\n    heredoc\nEOTXTa\n        ,\n        <<<'EOTXTb'\n    nowdoc\nEOTXTb\n        ,\n        'foo'\n    );\n}"},
		{"provideFixCases#36", map[string]any{"on_multiline": "ignore"}, "<?php xyz#\n (#\n\"\"#\n,#\n$a#\n);", "<?php xyz#\n (#\n\"\"#\n,#\n$a#\n);"},
		{"provideFixCases#37", map[string]any{"on_multiline": "ensure_single_line"}, "<?php xyz#\n (#\n\"\"#\n,#\n$a#\n);", "<?php xyz#\n (#\n\"\"#\n,#\n$a#\n);"},
		{"provideFixCases#39", nil, "<?php\nfunctionCall(\n    'a', 'b',\n    'c'\n);", "<?php\nfunctionCall(\n    'a',\n    'b',\n    'c'\n);"},
		{"provideFixCases#40", nil, "<?php\nf(1,2,\n3);", "<?php\nf(\n    1,\n    2,\n    3\n);"},
		{"provideFixCases#41", nil, "<?php\nstr_replace(\"\\n\", PHP_EOL, <<<'TEXT'\n   1) someFile.php\n\nTEXT\n);", "<?php\nstr_replace(\n    \"\\n\",\n    PHP_EOL,\n    <<<'TEXT'\n   1) someFile.php\n\nTEXT\n);"},
		{"provideFixCases#43", nil, "<?php\nif (true) {\n    functionCall(\n        'a', 'b',\n        'c'\n    );\n}", "<?php\nif (true) {\n    functionCall(\n        'a',\n        'b',\n        'c'\n    );\n}"},
		{"provideFixCases#44", nil, "<?php\ndefraculate(1, array(\n    'a',\n    'b',\n    'c',\n), 42);", "<?php\ndefraculate(1, array(\n    'a',\n    'b',\n    'c',\n), 42);"},
		{"provideFixCases#45", nil, "<?php\ndefraculate(1, function () {\n    $a = 42;\n}, 42);", "<?php\ndefraculate(1, function () {\n    $a = 42;\n}, 42);"},
		{"provideFixCases#46", nil, "<?php\ndefraculate(\n    1, 2, 3);", "<?php\ndefraculate(\n    1,\n    2,\n    3\n);"},
		{"provideFixCases#47", nil, "<?php\ndefraculate(1, 2, 3\n);", "<?php\ndefraculate(\n    1,\n    2,\n    3\n);"},
		{"provideFixCases#48", nil, "<?php\ngetSchwifty('rick', defraculate(1, 2, 3\n), 'morty');", "<?php\ngetSchwifty('rick', defraculate(\n    1,\n    2,\n    3\n), 'morty');"},
		{"provideFixCases#49", nil, "<?php\nfunctionCall(\n    'a',/* comment */'b',\n    'c'\n);", "<?php\nfunctionCall(\n    'a', /* comment */\n    'b',\n    'c'\n);"},
		{"provideFixCases#50", nil, "<?php\nfoo('a',\n    'b',\n    [\n        'c',\n        'd', bar('e', 'f'),\n        baz('g',\n            ['h',\n                'i',\n            ]),\n    ]);", "<?php\nfoo(\n    'a',\n    'b',\n    [\n        'c',\n        'd', bar('e', 'f'),\n        baz(\n            'g',\n            ['h',\n                'i',\n            ]\n        ),\n    ]\n);"},
		{"provideFixCases#51", nil, "<?php\n$this->with('<?php\n%s\nclass FooClass\n{\n}', $comment, false);", "<?php\n$this->with('<?php\n%s\nclass FooClass\n{\n}', $comment, false);"},
		{"provideFixCases#52", nil, "<?php\n$a = array/**/(  1);\n$a = array/**/( 12,\n7);\n$a = array/***/(123,  7);\n$a = array (        1,\n2);", "<?php\n$a = array/**/(  1);\n$a = array/**/( 12,\n7);\n$a = array/***/(123,  7);\n$a = array (        1,\n2);"},
		{"provideFixCases#53", nil, "<?php\nif (true &&\n    true\n    ) {\n    // do whatever\n}", "<?php\nif (true &&\n    true\n    ) {\n    // do whatever\n}"},
		{"provideFixCases#55", nil, "<?php\n// no fix\nlist($a,\n    $b, $c) = $a;\nisset($a,\n$b, $c);\nunset($a,\n$b, $c);\narray(1,\n    2,3\n);", "<?php\n// no fix\nlist($a,\n    $b, $c) = $a;\nisset($a,\n$b, $c);\nunset($a,\n$b, $c);\narray(1,\n    2,3\n);"},
		{"provideFixCases#56", nil, "<?php\ncall_user_func(function ($arguments) {\n    echo 'a',\n      'b';\n}, $argv);", "<?php\ncall_user_func(function ($arguments) {\n    echo 'a',\n      'b';\n}, $argv);"},
		{"provideFixCases#57", nil, "<?php\ncall_user_func(function ($arguments) {\n    echo 'a', 'b';\n},\n$argv);", "<?php\ncall_user_func(\n    function ($arguments) {\n    echo 'a', 'b';\n},\n    $argv\n);"},
		{"provideFixCases#58", map[string]any{"on_multiline": "ensure_single_line"}, "<?php\nfunction foo(\n    $a,\n    $b\n) {\n    // foo\n}\nfoo(\n    $a,\n    $b\n);", "<?php\nfunction foo($a, $b) {\n    // foo\n}\nfoo($a, $b);"},
		{"provideFixCases#59", map[string]any{"on_multiline": "ensure_single_line"}, "<?php\nfunction foo(/* foo */// bar\n    $a, /* foo */// bar\n    $b#foo\n) {\n    // foo\n}\nfoo(/* foo */// bar\n    $a, /* foo */// bar\n    $b#foo\n);", "<?php\nfunction foo(/* foo */// bar\n    $a, /* foo */// bar\n    $b#foo\n) {\n    // foo\n}\nfoo(/* foo */// bar\n    $a, /* foo */// bar\n    $b#foo\n);"},
		{"provideFixCases#60", map[string]any{"on_multiline": "ensure_single_line"}, "<?php\nfunction foo(\n\n\n    $a,\n\n\n    $b\n\n\n) {\n    // foo\n}\nfoo(\n\n\n    $a,\n\n\n    $b\n\n\n);", "<?php\nfunction foo($a, $b) {\n    // foo\n}\nfoo($a, $b);"},
		{"provideFixCases#61", map[string]any{"on_multiline": "ensure_single_line"}, "<?php\nclass Foo {\n    public static function foo1(\n        $a,\n        $b,\n        $c\n    ) {}\n    private function foo2(\n        $a,\n        $b,\n        $c\n    ) {}\n}", "<?php\nclass Foo {\n    public static function foo1($a, $b, $c) {}\n    private function foo2($a, $b, $c) {}\n}"},
		{"provideFixCases#62", map[string]any{"on_multiline": "ensure_single_line"}, "<?php\nnew class {\n    public static function foo1(\n        $a,\n        $b,\n        $c\n    ) {}\n    private function foo2(\n        $a,\n        $b,\n        $c\n    ) {}\n};", "<?php\nnew class {\n    public static function foo1($a, $b, $c) {}\n    private function foo2($a, $b, $c) {}\n};"},
		{"provideFixCases#63", map[string]any{"on_multiline": "ensure_single_line", "keep_multiple_spaces_after_comma": true}, "<?php\nfunction foo(\n    $a,\n    $b\n) {\n    // foo\n}\nfoo(\n    $a,\n    $b\n);", "<?php\nfunction foo($a,    $b) {\n    // foo\n}\nfoo($a,    $b);"},
		{"provideFixCases#64", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfunction foo(\n    $a\n) {\n    // foo\n}\nfoo(\n    $a\n);", "<?php\nfunction foo($a) {\n    // foo\n}\nfoo($a);"},
		{"provideFixCases#65", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfunction foo(\n    $a, $b,\n    $c\n) {\n    // foo\n}\nfoo(\n    $a, $b,\n    $c\n);", "<?php\nfunction foo(\n    $a,\n    $b,\n    $c\n) {\n    // foo\n}\nfoo(\n    $a,\n    $b,\n    $c\n);"},
		{"provideFixCases#66", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo($a);\nfoo($a, $b);", "<?php\nfoo($a);\nfoo($a, $b);"},
		{"provideFixCases#67", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo();", "<?php\nfoo();"},
		{"provideFixCases#68", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    bar($a, $b)\n);", "<?php\nfoo(bar($a, $b));"},
		{"provideFixCases#69", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    bar(\n        $a,\n        $b\n    )\n);", "<?php\nfoo(bar(\n    $a,\n    $b\n));"},
		{"provideFixCases#70", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    [\n        'a',\n        'b',\n        'c',\n    ]\n);", "<?php\nfoo([\n        'a',\n        'b',\n        'c',\n    ]);"},
		{"provideFixCases#71", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    Foo::bar()->baz($quz)\n);", "<?php\nfoo(Foo::bar()->baz($quz));"},
		{"provideFixCases#72", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    Foo::bar()\n        ->baz($quz)\n);", "<?php\nfoo(\n    Foo::bar()\n        ->baz($quz)\n);"},
		{"provideFixCases#73", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    Foo::bar()\n        ->baz($quz)\n        ->qux(\n            $foo,\n            $bar\n        )\n);", "<?php\nfoo(\n    Foo::bar()\n        ->baz($quz)\n        ->qux(\n            $foo,\n            $bar\n        )\n);"},
		{"provideFixCases#74", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    'hello ' . 'world'\n);", "<?php\nfoo('hello ' . 'world');"},
		{"provideFixCases#75", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    'hello '\n        . 'world'\n);", "<?php\nfoo(\n    'hello '\n        . 'world'\n);"},
		{"provideFixCases#76", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    /* hello */ $a\n);", "<?php\nfoo(\n    /* hello */ $a\n);"},
		{"provideFixCases#77", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    $a /* hello */\n);", "<?php\nfoo(\n    $a /* hello */\n);"},
		{"provideFixCases#78", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    /* hello */\n    $a\n);", "<?php\nfoo(\n    /* hello */\n    $a\n);"},
		{"provideFixCases#79", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    /**\n     * hello\n     */\n    $a\n);", "<?php\nfoo(\n    /**\n     * hello\n     */\n    $a\n);"},
		{"provideFixCases#80", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    $a // hello\n);", "<?php\nfoo(\n    $a // hello\n);"},
		{"provideFixCases#81", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    $a # hello\n);", "<?php\nfoo(\n    $a # hello\n);"},
		{"provideFixCases#82", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    function () {\n        return true;\n    }\n);", "<?php\nfoo(function () {\n        return true;\n    });"},
		{"provideFixCases#83", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfoo(\n    ['a', 'b', 'c']\n);", "<?php\nfoo(['a', 'b', 'c']);"},
		{"provideFixCases#84", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\n$foo->bar(\n    $baz\n);", "<?php\n$foo->bar($baz);"},
		{"provideFixCases#85", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\n$foo->bar(\n    $baz, $qux\n);", "<?php\n$foo->bar(\n    $baz,\n    $qux\n);"},
		{"provideFixCases#86", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nFoo::bar(\n    $baz\n);", "<?php\nFoo::bar($baz);"},
		{"provideFixCases#87", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nFoo::bar(\n    $baz, $qux\n);", "<?php\nFoo::bar(\n    $baz,\n    $qux\n);"},
		{"provideFixCases#88", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nclass Foo {\n    public function foo1(\n        $a\n    ) {}\n    private function foo2(\n        $a, $b,\n        $c\n    ) {}\n}", "<?php\nclass Foo {\n    public function foo1($a) {}\n    private function foo2(\n        $a,\n        $b,\n        $c\n    ) {}\n}"},
		{"provideFixCases#89", map[string]any{"on_multiline": "ensure_fully_multiline"}, "<?php\nif (true) {\n    execute(\n        $foo,\n        $bar\n        );\n}", "<?php\nif (true) {\n    execute(\n        $foo,\n        $bar\n    );\n}"},
		{"provideFixCases#90", nil, "<?php\n$example = function () use ($message1,$message2) {\n};", "<?php\n$example = function () use ($message1, $message2) {\n};"},
		{"provideFixCases#91", nil, "<?php foo(\"aaa\n    bbb\",\n    $c, $d ,\n        $e,\n        $f);\n", "<?php foo(\n    \"aaa\n    bbb\",\n    $c,\n    $d,\n    $e,\n    $f\n);\n"},
		{"provideFixCases#92", nil, "<?php foo(\"aaa\n    bbb\", // comment1\n    $c, /** comment2 */$d ,\n        $e/* comment3 */,\n        $f);# comment4\n", "<?php foo(\n    \"aaa\n    bbb\", // comment1\n    $c, /** comment2 */\n    $d,\n    $e/* comment3 */,\n    $f\n);# comment4\n"},
		{"provideFixCases#93", nil, "<?php\nfoo(\n    /* bar */ \"baz\"\n);\n            ", "<?php\nfoo(\n    /* bar */\n    \"baz\"\n);\n            "},
		{"provideFixCases#94", nil, "<?php\nfunction f()\n{\n    echo <<<TEXT\n        some text\n        {$object->method(\n            42\n        )}\n        some text\n    TEXT;\n}", "<?php\nfunction f()\n{\n    echo <<<TEXT\n        some text\n        {$object->method(\n            42\n        )}\n        some text\n    TEXT;\n}"},
		{"provideFixCases#95", nil, "<?php\nfunction f()\n{\n    echo <<<'TEXT'\n        some text\n        {$object->method(\n            42\n        )}\n        some text\n    TEXT;\n}", "<?php\nfunction f()\n{\n    echo <<<'TEXT'\n        some text\n        {$object->method(\n            42\n        )}\n        some text\n    TEXT;\n}"},
		{"provideFixCases#96", nil, "<?php\nfunction f()\n{\n    echo <<<TEXT\n        some text\n        some value: {$object->method(\n            42\n        )}\n        some text\n    TEXT;\n}", "<?php\nfunction f()\n{\n    echo <<<TEXT\n        some text\n        some value: {$object->method(\n            42\n        )}\n        some text\n    TEXT;\n}"},
		{"provideFixCases#125", nil, "<?php function A($c ,...$a){}", "<?php function A($c, ...$a){}"},
		{"provideFixCases#126", map[string]any{"after_heredoc": true}, "<?php\nfoo(\n    <<<'EOD'\n        bar\n        EOD\n    ,\n    'baz'\n);", "<?php\nfoo(\n    <<<'EOD'\n        bar\n        EOD,\n    'baz'\n);"},
		{"provideFixCases#127", map[string]any{"on_multiline": "ensure_fully_multiline"}, "<?php\nfoo(\n    $bar,\n    $baz,\n);", "<?php\nfoo(\n    $bar,\n    $baz,\n);"},
		{"provideFixCases#128", map[string]any{"on_multiline": "ensure_fully_multiline"}, "<?php\nfunctionCall(\n    1, 2,\n    3,\n);", "<?php\nfunctionCall(\n    1,\n    2,\n    3,\n);"},
		{"provideFixCases#129", nil, "<?php foo(1,2,3,);", "<?php foo(1, 2, 3, );"},
		{"provideFixCases#130", map[string]any{"on_multiline": "ensure_fully_multiline"}, "<?php\n$fn = fn(\n    $test1, $test2\n) => null;", "<?php\n$fn = fn(\n    $test1,\n    $test2\n) => null;"},
		{"multiple attributes", nil, "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo] #[Bar] private ?string $name = null,\n    ) {}\n}", "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo]\n        #[Bar]\n        private ?string $name = null,\n    ) {}\n}"},
		{"keep attributes as-is", map[string]any{"attribute_placement": "ignore"}, "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo] #[Bar] private ?string $name = null,\n    ) {}\n}", "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo] #[Bar] private ?string $name = null,\n    ) {}\n}"},
		{"multiple attributes on the same line as argument", map[string]any{"attribute_placement": "same_line"}, "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo]\n        #[Bar]\n        private ?string $name = null,\n    ) {}\n}", "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo] #[Bar] private ?string $name = null,\n    ) {}\n}"},
		{"single attribute markup with comma separated list", nil, "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo, Bar] private ?string $name = null,\n    ) {}\n}", "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo, Bar]\n        private ?string $name = null,\n    ) {}\n}"},
		{"attributes with arguments", nil, "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo(value: 1234, otherValue: [1, 2, 3])] #[Bar(Bar::BAZ, array('[',']'))] private ?string $name = null,\n    ) {}\n}", "<?php\nclass MyClass\n{\n    public function __construct(\n        private string $id,\n        #[Foo(value: 1234, otherValue: [1, 2, 3])]\n        #[Bar(Bar::BAZ, array('[',']'))]\n        private ?string $name = null,\n    ) {}\n}"},
		{"fully qualified attributes", nil, "<?php\nfunction foo(\n    #[\\Foo\\Bar] $bar, #[\\Foo\\Baz] $baz, #[\\Foo\\Buzz] $buzz\n) {}", "<?php\nfunction foo(\n    #[\\Foo\\Bar]\n    $bar,\n    #[\\Foo\\Baz]\n    $baz,\n    #[\\Foo\\Buzz]\n    $buzz\n) {}"},
		{"multiline attributes", nil, "<?php\nfunction foo($foo, #[\n    Foo\\Bar,\n    Foo\\Baz,\n    Foo\\Buzz(a: 'astral', b: 1234),\n] $bar) {}", "<?php\nfunction foo(\n    $foo,\n    #[\n    Foo\\Bar,\n    Foo\\Baz,\n    Foo\\Buzz(a: 'astral', b: 1234),\n]\n    $bar\n) {}"},
		{"ensure_single_line_for_single_argument: collapses when parameter has inline attribute", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfunction foo(\n    #[Attr] $x\n) {}", "<?php\nfunction foo(#[Attr] $x) {}"},
		{"ensure_single_line_for_single_argument: collapses single-argument attribute invocation on function", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\n#[Attr(\n    'foo'\n)]\nfunction foo() {}", "<?php\n#[Attr('foo')]\nfunction foo() {}"},
		{"ensure_single_line_for_single_argument: collapses single-argument attribute invocation with named argument on function", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\n#[Attr(\n    value: 'foo'\n)]\nfunction foo() {}", "<?php\n#[Attr(value: 'foo')]\nfunction foo() {}"},
		{"ensure_single_line_for_single_argument: collapses inner multiline single-argument attribute when parameter has attribute on previous line", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfunction foo(\n    #[Attr(\n        value: 'something'\n    )]\n    $x\n) {}", "<?php\nfunction foo(\n    #[Attr(value: 'something')]\n    $x\n) {}"},
		{"ensure_single_line_for_single_argument: does not collapse when parameter has attribute on previous line", map[string]any{"on_multiline": "ensure_single_line_for_single_argument"}, "<?php\nfunction foo(\n    #[Attr]\n    $x\n) {}", "<?php\nfunction foo(\n    #[Attr]\n    $x\n) {}"},
		{"provideFix81Cases#143", nil, "<?php\n[Foo::class, 'method']( ...\n) ?>", "<?php\n[Foo::class, 'method'](\n    ...\n) ?>"},
	}
	pass, fail := 0, 0
	for _, c := range cases {
		f := fixerRule(MethodArgumentSpace{})
		if c.cfg != nil {
			f = MethodArgumentSpace{}.WithConfig(c.cfg).(fixerRule)
		}
		got, _ := apply(t, f, c.src)
		if got != c.want {
			fail++
			t.Errorf("%s:\n src=%q\n got=%q\nwant=%q", c.name, c.src, got, c.want)
		} else {
			pass++
		}
	}
	t.Logf("php-cs-fixer MAS cases: %d pass, %d fail of %d", pass, fail, pass+fail)
}
