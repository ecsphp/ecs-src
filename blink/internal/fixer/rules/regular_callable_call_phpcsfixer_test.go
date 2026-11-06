package rules

import "testing"

// Cases ported from PHP-CS-Fixer's RegularCallableCallFixerTest
// (tests/Fixer/FunctionNotation); every expected output is what PHP-CS-Fixer
// produces for the input. The pre-8.0 "$a{...}" index-brace syntax is left out,
// as it is removed from PHP 8.
func TestRegularCallableCallPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, RegularCallableCall{}, []aliasUpstreamCase{
		{
			"call by name - list", nil,
			"<?php\n                dont_touch_me(1, 2);\n                call_user_func('foo');\n                call_user_func(\"foo\");\n                call_user_func(\"foo\" . \"bar\"); // not (yet?) supported by Fixer, possible since PHP 7+\n                call_user_func(\"var_dump\", 1, 2);\n                call_user_func(\"Bar\\Baz::d\", 1, 2);\n                call_user_func(\"\\Bar\\Baz::d\", 1, 2);",
			"<?php\n                dont_touch_me(1, 2);\n                foo();\n                foo();\n                call_user_func(\"foo\" . \"bar\"); // not (yet?) supported by Fixer, possible since PHP 7+\n                var_dump(1, 2);\n                Bar\\Baz::d(1, 2);\n                \\Bar\\Baz::d(1, 2);",
		},
		{
			"call by name - array", nil,
			"<?php call_user_func_array(\"Bar\\Baz::d\", [1, 2]);",
			"<?php Bar\\Baz::d(...[1, 2]);",
		},
		{
			"call by array-as-name, not supported", nil,
			"<?php\n                call_user_func(array(\"Bar\\baz\", \"myCallbackMethod\"), 1, 2);\n                call_user_func([\"Bar\\baz\", \"myCallbackMethod\"], 1, 2);\n                call_user_func([$obj, \"myCallbackMethod\"], 1, 2);\n                call_user_func([$obj, $cb.\"Method\"], 1, 2);\n                call_user_func(array(__NAMESPACE__ .\"Foo\", \"test\"), 1, 2);\n                call_user_func(array(\"Foo\", \"parent::method\"), 1, 2); // no way to convert `parent::`\n            ",
			"<?php\n                call_user_func(array(\"Bar\\baz\", \"myCallbackMethod\"), 1, 2);\n                call_user_func([\"Bar\\baz\", \"myCallbackMethod\"], 1, 2);\n                call_user_func([$obj, \"myCallbackMethod\"], 1, 2);\n                call_user_func([$obj, $cb.\"Method\"], 1, 2);\n                call_user_func(array(__NAMESPACE__ .\"Foo\", \"test\"), 1, 2);\n                call_user_func(array(\"Foo\", \"parent::method\"), 1, 2); // no way to convert `parent::`\n            ",
		},
		{
			"call by variable", nil,
			"<?php\n                call_user_func($c, 1, 2);\n                call_user_func($a[\"b\"][\"c\"], 1, 2);\n            ",
			"<?php\n                $c(1, 2);\n                $a[\"b\"][\"c\"](1, 2);\n            ",
		},
		{
			"call with comments", nil,
			"<?php\n                dont_touch_me(/* a */1, 2/** b */);\n                call_user_func('foo');\n                call_user_func(\"foo\", /* a */1, 2/** b */);\n                call_user_func(\"foo\"/* a *//** b */, /** c */1/** d */,/** e */ 2);\n                call_user_func(\"foo\" . \"bar\"); // not (yet?) supported by Fixer, possible since PHP 7+\n                call_user_func(\"var_dump\", 1, /*\n                    aaa\n                    */ 2);\n                call_user_func(\"var_dump\", 3 /*\n                    aaa\n                    */, 4);\n                call_user_func(\"Bar\\Baz::d\", 1, 2);\n                call_user_func(\"\\Bar\\Baz::d\", 1, 2);",
			"<?php\n                dont_touch_me(/* a */1, 2/** b */);\n                foo();\n                foo(/* a */1, 2/** b */);\n                foo(/* a *//** b *//** c */1/** d */,/** e */ 2);\n                call_user_func(\"foo\" . \"bar\"); // not (yet?) supported by Fixer, possible since PHP 7+\n                var_dump(1, /*\n                    aaa\n                    */ 2);\n                var_dump(3 /*\n                    aaa\n                    */, 4);\n                Bar\\Baz::d(1, 2);\n                \\Bar\\Baz::d(1, 2);",
		},
		{
			"single var", nil,
			"<?php \\call_user_func($foo) ?>",
			"<?php $foo() ?>",
		},
		{
			"unsafe repeated variable", nil,
			"<?php call_user_func($foo, $foo = \"bar\");",
			"<?php call_user_func($foo, $foo = \"bar\");",
		},
		{
			"call by property", nil,
			"<?php\n                call_user_func($f->c, 1, 2);\n                call_user_func($f->{c}, 1, 2);\n                call_user_func($x[\"y\"]->c, 1, 2);\n                call_user_func($x[\"y\"]->{\"c\"}, 1, 2);\n            ",
			"<?php\n                ($f->c)(1, 2);\n                ($f->{c})(1, 2);\n                ($x[\"y\"]->c)(1, 2);\n                ($x[\"y\"]->{\"c\"})(1, 2);\n            ",
		},
		{
			"call by anon-function", nil,
			"<?php\n                call_user_func(function ($a, $b) { var_dump($a, $b); }, 1, 2);\n                call_user_func(static function ($a, $b) { var_dump($a, $b); }, 1, 2);\n            ",
			"<?php\n                (function ($a, $b) { var_dump($a, $b); })(1, 2);\n                (static function ($a, $b) { var_dump($a, $b); })(1, 2);\n            ",
		},
		{
			"complex cases", nil,
			"<?php\n                call_user_func('a'.$a.$b, 1, 2);\n                call_user_func($a/**/.$b, 1, 2);\n                \\call_user_func(function (){});\n                call_user_func($a[\"b\"][\"c\"]->a, 1, 2, 3, 4);\n                call_user_func($a::$b, 1, 2);\n                call_user_func($a[1]::$b[2][3], [&$c], array(&$d));\n            ",
			"<?php\n                call_user_func('a'.$a.$b, 1, 2);\n                ($a/**/.$b)(1, 2);\n                (function (){})();\n                ($a[\"b\"][\"c\"]->a)(1, 2, 3, 4);\n                ($a::$b)(1, 2);\n                ($a[1]::$b[2][3])([&$c], array(&$d));\n            ",
		},
		{
			"callback returning a callback", nil,
			"<?php call_user_func($a(1, 2), [&$x], array(&$z));",
			"<?php ($a(1, 2))([&$x], array(&$z));",
		},
		{
			"redeclare/override", nil,
			"<?php\n                if (!function_exists(\"call_user_func\")) {\n                    function call_user_func($foo){}\n                }\n            ",
			"<?php\n                if (!function_exists(\"call_user_func\")) {\n                    function call_user_func($foo){}\n                }\n            ",
		},
		{
			"function name with escaped slash", nil,
			"<?php call_user_func_array(\"\\\\pack\", $args);",
			"<?php \\pack(...$args);",
		},
		{
			"call_user_func_array with leading slash", nil,
			"<?php \\call_user_func_array(\"\\\\pack\", $args);",
			"<?php \\pack(...$args);",
		},
		{
			"call_user_func_array caps", nil,
			"<?php \\CALL_USER_FUNC_ARRAY(\"\\\\pack\", $args);",
			"<?php \\pack(...$args);",
		},
		{
			"trailing comma", nil,
			"<?php call_user_func(\"foo\", 1,);",
			"<?php foo(1,);",
		},
		{
			"empty string double quote", nil,
			"<?php call_user_func(\"\", 1,);",
			"<?php call_user_func(\"\", 1,);",
		},
		{
			"empty string single quote", nil,
			"<?php call_user_func('    ', 1,);",
			"<?php call_user_func('    ', 1,);",
		},
		{
			"string with padding", nil,
			"<?php call_user_func(\" padded  \", 1,);",
			"<?php call_user_func(\" padded  \", 1,);",
		},
		{
			"binary string lower double quote", nil,
			"<?php call_user_func(b\"foo\", 1,);",
			"<?php call_user_func(b\"foo\", 1,);",
		},
		{
			"binary string upper single quote", nil,
			"<?php call_user_func(B\"foo\", 1,);",
			"<?php call_user_func(B\"foo\", 1,);",
		},
		{
			"static property as first argument", nil,
			"<?php\nclass Foo {\n  public static $factory;\n  public static function createFromFactory(...$args) {\n    return call_user_func_array(static::$factory, $args);\n  }\n}",
			"<?php\nclass Foo {\n  public static $factory;\n  public static function createFromFactory(...$args) {\n    return call_user_func_array(static::$factory, $args);\n  }\n}",
		},
		{
			"first-class callable 8.1", nil,
			"<?php \\call_user_func(...) ?>",
			"<?php \\call_user_func(...) ?>",
		},
	})
}
