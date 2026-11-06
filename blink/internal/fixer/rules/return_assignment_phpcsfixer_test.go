package rules

import "testing"

// Cases ported from PHP-CS-Fixer's ReturnAssignmentFixerTest; every expected
// output is what PHP-CS-Fixer produces for the input. This is a conservative
// port, so cases where PHP-CS-Fixer rewrites a "}" value expression (closure,
// anonymous class, match) are intentionally left out rather than asserted.
func TestReturnAssignmentPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	cases := []aliasUpstreamCase{
		{
			"simple",
			nil,
			"<?php\n                    function A()\n                    {\n                        $a = 15;\n                        return $a;\n                    }\n                ",
			"<?php\n                    function A()\n                    {\n                        return 15;\n                    }\n                ",
		},
		{
			"comments around assignment",
			nil,
			"<?php\n                    function A()\n                    {\n                        /*0*/$a/*1*/=/*2*/15;/*3*//*4*/ /*5*/ return/*6*/$a/*7*/;/*8*/\n                    }\n                ",
			"<?php\n                    function A()\n                    {\n                        /*0*/return /*1*//*2*/15;/*3*//*4*/ /*5*/ /*6*//*7*//*8*/\n                    }\n                ",
		},
		{
			"abstract method sibling",
			nil,
			"<?php\n                    abstract class B\n                    {\n                        abstract protected function Z();public function A()\n                        {\n                            $a = 16; return $a;\n                        }\n                    }\n                ",
			"<?php\n                    abstract class B\n                    {\n                        abstract protected function Z();public function A()\n                        {\n                            return 16;\n                        }\n                    }\n                ",
		},
		{
			"if block and function call",
			nil,
			"<?php\n                    function b() {\n                        if ($c) {\n                            $b = 0;\n                            return $b;\n                        }\n                        $a = testFunction(654+1);\n                        return $a;\n                    }\n                ",
			"<?php\n                    function b() {\n                        if ($c) {\n                            return 0;\n                        }\n                        return testFunction(654+1);\n                    }\n                ",
		},
		{
			"minimal notation",
			nil,
			"<?php $e=function(){$a=1;return$a;};$f=function(){$a=1;return$a;};$g=function(){$a=1;return$a;};",
			"<?php $e=function(){return 1;};$f=function(){return 1;};$g=function(){return 1;};",
		},
		{
			"leading line comment",
			nil,
			"<?php\nfunction A($b)\n{\n        // Comment\n        $value = a(\"2\", 4, $b);\n\n        return $value;\n}\n",
			"<?php\nfunction A($b)\n{\n        // Comment\n        return a(\"2\", 4, $b);\n}\n",
		},
		{
			"parenthesised expression",
			nil,
			"<?php function a($b,$c)  {if($c>1){echo 1;} $a= (1 + 2 + $b);return $a; }",
			"<?php function a($b,$c)  {if($c>1){echo 1;} return (1 + 2 + $b);  }",
		},
		{
			"close tag kept",
			nil,
			"<?php\n                    function a() {\n                        $a = 4563;\n                        return $a ?> <?php\n                    }\n                ",
			"<?php\n                    function a() {\n                        return 4563;\n                          ?> <?php\n                    }\n                ",
		},
		{
			"method chain value",
			nil,
			"<?php\n                    function A()\n                    {\n                        $a = $f[1]->a();\n                        return $a;\n                    }\n                ",
			"<?php\n                    function A()\n                    {\n                        return $f[1]->a();\n                    }\n                ",
		},
		{
			"value with nested closure",
			nil,
			"<?php\n                    function a($foos) {\n                        $bars = array_map(function ($foo) {\n                            return (string) $foo;\n                        }, $foos);\n\n                        return $bars;\n                    }",
			"<?php\n                    function a($foos) {\n                        return array_map(function ($foo) {\n                            return (string) $foo;\n                        }, $foos);\n                    }",
		},
		{
			"value is parenthesised assignment",
			nil,
			"<?php\n                    function a($foos) {\n                        $bars = ($foos = ['bar']);\n\n                        return $bars;\n                    }",
			"<?php\n                    function a($foos) {\n                        return ($foos = ['bar']);\n                    }",
		},
		{
			"iife value",
			nil,
			"<?php\n                    function a($foos) {\n                        $bars = (function ($foos) {\n                            return $foos;\n                        })($foos);\n\n                        return $bars;\n                    }",
			"<?php\n                    function a($foos) {\n                        return (function ($foos) {\n                            return $foos;\n                        })($foos);\n                    }",
		},
		{
			"arrow function value",
			nil,
			"<?php\n                    function Foo() {\n                        $fn1 = fn($x) => $x + $y;\n\n                        return $fn1;\n                    }\n                ",
			"<?php\n                    function Foo() {\n                        return fn($x) => $x + $y;\n                    }\n                ",
		},
		{
			"double reduction",
			nil,
			"<?php\n\nfunction foo() {\n    $a = bar();\n    $b = $a;\n\n    return $b;\n}\n",
			"<?php\n\nfunction foo() {\n    return bar();\n}\n",
		},

		// no-change cases proving conservativeness
		{"reference assignment", nil, "<?php\n                    function bar() {\n                        $foo = &foo();\n                        return $foo;\n                }", ""},
		{"static var", nil, "<?php\n                    function a() {\n                        static $a;\n                        $a = time();\n                        return $a;\n                    }\n                ", ""},
		{"global var", nil, "<?php\n                    function a() {\n                        global $a;\n                        $a = time();\n                        return $a;\n                    }\n                ", ""},
		{"param by reference", nil, "<?php\n                function foo(&$var)\n                    {\n                        $var = 1;\n                        return $var;\n                    }\n                ", ""},
		{"not in function scope", nil, "<?php\n                    $a = 1; // var might be global here\n                    return $a;\n                ", ""},
		{"return complex statement", nil, "<?php\n                    function a($c)\n                    {\n                        $a = 1;\n                        return $a + $c;\n                    }\n                ", ""},
		{"superglobal assign", nil, "<?php\n                    function a($c)\n                    {\n                        $_SERVER[\"abc\"] = 3;\n                        return $_SERVER;\n                    }\n                ", ""},
		{"if assign", nil, "<?php\n                    function foo ($bar)\n                    {\n                        $a = 123;\n                        if ($bar)\n                            $a = 12345;\n                        return $a;\n                    }\n                ", ""},
		{"changed after declaration", nil, "<?php\n                    function a($c)\n                    {\n                        $a = 1;\n                        $a += 1;\n                        return $a;\n                    }\n                ", ""},
		{"complex statement", nil, "<?php\n                    function a($c)\n                    {\n                        $d = $c && $a = 1;\n                        return $a;\n                    }\n                ", ""},
		{"require import", nil, "<?php\n                    function a()\n                    {\n                        require __DIR__.\"/test3.php\";\n                        $b = 1;\n                        return $b;\n                    }\n                ", ""},
		{"eval", nil, "<?php\n                    $c = function ($x) {\n                        $x = eval($x);\n                        $x = 1;\n                        return $x;\n                    };\n                ", ""},
		{"dynamic var brace", nil, "<?php\n                    function A($g)\n                    {\n                        $h = ${$g};\n\n                        return $h;\n                    }\n                ", ""},
		{"variable variable", nil, "<?php\n                    function B($c)\n                    {\n                        $b = $$c;\n\n                        return $b;\n                    }\n                ", ""},
		{"return by reference function", nil, "<?php\n                function &foo() {\n                    $var = 1;\n                    return $var;\n                }", ""},
		{"if ($a = N)", nil, "<?php\n                    function a($c)\n                    {\n                        if ($a = 1)\n                            return $a;\n                    }\n                ", ""},

		// try/catch/finally
		{
			"try catch do not fix inside try",
			nil,
			"<?php\n                function foo()\n                {\n                    if (isSomeCondition()) {\n                        $result = getSomeResult();\n\n                        return $result;\n                    }\n\n                    try {\n                        $result = getResult();\n\n                        return $result;\n                    } catch (\\Throwable $exception) {\n                        baz($result ?? null);\n                    }\n                }\n                ",
			"<?php\n                function foo()\n                {\n                    if (isSomeCondition()) {\n                        return getSomeResult();\n                    }\n\n                    try {\n                        $result = getResult();\n\n                        return $result;\n                    } catch (\\Throwable $exception) {\n                        baz($result ?? null);\n                    }\n                }\n                ",
		},
		{
			"try catch finally not fixed",
			nil,
			"<?php\n                function add($a, $b): mixed\n                {\n                    try {\n                        $result = $a + $b;\n\n                        return $result;\n                    } catch (\\Throwable $th) {\n                        throw $th;\n                    } finally {\n                        echo 'result:', $result, \\PHP_EOL;\n                    }\n                }\n                ",
			"",
		},

		// doc comments
		{
			"doc comment var without name is fixed",
			nil,
			"<?php\n\nfunction foo() {\n    /** @var int[] */\n    $a = doSomething();\n\n    return $a;\n}\n",
			"<?php\n\nfunction foo() {\n    /** @var int[] */\n    return doSomething();\n}\n",
		},
		{
			"doc comment var with name, default config is fixed",
			nil,
			"<?php\n\nfunction foo() {\n    /** @var int[] $a */\n    $a = doSomething();\n\n    return $a;\n}\n",
			"<?php\n\nfunction foo() {\n    /** @var int[] $a */\n    return doSomething();\n}\n",
		},
		{
			"skip_named_var_tags keeps named var tag",
			map[string]any{"skip_named_var_tags": true},
			"<?php\n\nfunction foo() {\n    /** @var int[] $a */\n    $a = doSomething();\n\n    return $a;\n}\n",
			"",
		},
		{
			"skip_named_var_tags without name is fixed",
			map[string]any{"skip_named_var_tags": true},
			"<?php\n\nfunction foo() {\n    /** @var int[] */\n    $a = doSomething();\n\n    return $a;\n}\n",
			"<?php\n\nfunction foo() {\n    /** @var int[] */\n    return doSomething();\n}\n",
		},
		{
			"line comment before assignment",
			nil,
			"<?php\n\nfunction foo() {\n    // some comment\n    $a = doSomething();\n\n    return $a;\n}\n",
			"<?php\n\nfunction foo() {\n    // some comment\n    return doSomething();\n}\n",
		},
	}
	// a "" expected marks a no-change case: the output must equal the input
	for i := range cases {
		if cases[i].expected == "" {
			cases[i].expected = cases[i].input
		}
	}
	aliasRunUpstreamCases(t, ReturnAssignment{}, cases)
}
