package rules

import "testing"

// Cases ported from PHP-CS-Fixer's UseArrowFunctionsFixerTest
// (tests/Fixer/FunctionNotation); every expected output is what PHP-CS-Fixer
// produces for the input (equal to the input when nothing changes).
func TestUseArrowFunctionsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	same := func(code string) aliasUpstreamCase { return aliasUpstreamCase{"", nil, code, code} }
	fix := func(expected, input string) aliasUpstreamCase {
		return aliasUpstreamCase{"", nil, input, expected}
	}
	aliasRunUpstreamCases(t, UseArrowFunctions{}, []aliasUpstreamCase{
		same(`<?php foo(function () use ($a, &$b) { return 1; });`),
		same(`<?php foo(function () { bar(); return 1; });`),
		fix(`<?php foo(fn()=> 1);`, `<?php foo(function(){return 1;});`),
		fix(`<?php foo(fn()=>$a);`, `<?php foo(function()use($a){return$a;});`),
		fix(`<?php foo( fn () => 1 );`, `<?php foo( function () { return 1; } );`),
		fix(`<?php $func = static fn &(array &$a, string ...$b): ?int => 1;`, `<?php $func = static function &(array &$a, string ...$b): ?int { return 1; };`),
		fix("<?php\n    foo(1, fn (int $a, Foo $b) => bar($a, $c), 2);", "<?php\n    foo(1, function (int $a, Foo $b) use ($c, $d) {\n        return bar($a, $c);\n    }, 2);"),
		fix("<?php\n    foo(fn () => 1);", "<?php\n    foo(function () {\n\n\n        return 1;\n\n\n    });"),
		fix("<?php\n    foo(fn ($a) => fn () => $a + 1);", "<?php\n    foo(function ($a) {\n        return function () use ($a) {\n            return $a + 1;\n        };\n    });"),
		same("<?php\n    foo(function () {// comment\n        return 1;\n    });"),
		same("<?php\n    foo(function () {\n        // comment\n        return 1;\n    });"),
		same("<?php\n    foo(function () {\n        return 1; // comment\n    });"),
		same("<?php\n    foo(function () {\n        return 1;\n        // comment\n    });"),
		fix("<?php\n    foo(fn () =>\n            1);", "<?php\n    foo(function () {\n        return\n            1;\n    });"),
		fix("<?php\n    $func = fn (\n        $a,\n        $b\n    ) => 1;", "<?php\n    $func = function (\n        $a,\n        $b\n    ) {\n        return 1;\n    };"),
		same("<?php\n    $func = function () {\n        return function () {\n            foo();\n        };\n    };"),
		fix(`<?php $testDummy = fn () => null;`, `<?php $testDummy = function () { return; };`),
		fix(`<?php $testDummy = fn () => null ;`, `<?php $testDummy = function () { return ; };`),
		fix(`<?php $testDummy = fn () => null/* foo */;`, `<?php $testDummy = function () { return/* foo */; };`),
		fix("<?php return fn () => [\n        CONST_A,\n        CONST_B,\n    ];", "<?php return function () {\n    return [\n        CONST_A,\n        CONST_B,\n    ];\n};"),
		fix("<?php\n            foo(\n                fn () => 42\n                        "+"\n            );", "<?php\n            foo(\n                function () {\n                    return 42\n                        ;\n                }\n            );"),
		same("<?php\n$load = \\Closure::bind(static function ($path, $env) use ($container, $loader, $resource, $type) {\n    return include $path;\n}, null, null);"),
		same("<?php\n$load = function ($path) use ($config) {\n    return include_once $path;\n};"),
		same("<?php\n$load = function ($path) use ($data) {\n    return require $path;\n};"),
		same("<?php\n$load = function ($path) use ($settings) {\n    return require_once $path;\n};"),
		fix("<?php\n$load = fn ($path) => include $path;", "<?php\n$load = function ($path) {\n    return include $path;\n};"),
		fix("<?php\n$load = fn ($path) => $data[$path];", "<?php\n$load = function ($path) use ($data) {\n    return $data[$path];\n};"),
		// PHP 8.5: closures inside attributes are not converted
		fix("<?php\nclass Foo {\n    function f1() {\n        return fn (int $x): int => 100 - $x;\n    }\n\n    #[Bar(callback: static function () { return true; })]\n    #[Baz(callback: static function (int $i): int { return $i + 100; })]\n    function f2() {\n        return static fn (int $x, int $y): int => 2 * $x + 3 * $y;\n    }\n}", "<?php\nclass Foo {\n    function f1() {\n        return function (int $x): int { return 100 - $x; };\n    }\n\n    #[Bar(callback: static function () { return true; })]\n    #[Baz(callback: static function (int $i): int { return $i + 100; })]\n    function f2() {\n        return static function (int $x, int $y): int { return 2 * $x + 3 * $y; };\n    }\n}"),
	})
}
