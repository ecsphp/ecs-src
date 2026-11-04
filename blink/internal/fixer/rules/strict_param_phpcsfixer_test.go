package rules

import "testing"

// Cases ported from PHP-CS-Fixer's StrictParamFixerTest.
func TestStrictParamPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, StrictParam{}, []aliasUpstreamCase{
		{"already strict", nil,
			"<?php\n    in_array(1, $a, true);\n    in_array(1, $a, false);\n    in_array(1, $a, $useStrict);",
			"<?php\n    in_array(1, $a, true);\n    in_array(1, $a, false);\n    in_array(1, $a, $useStrict);"},
		{"method not global", nil,
			"<?php class Foo\n                {\n                    public function in_array($needle, $haystack) {}\n                }",
			"<?php class Foo\n                {\n                    public function in_array($needle, $haystack) {}\n                }"},
		{"in_array add", nil, "<?php\n    in_array(1, $a);", "<?php\n    in_array(1, $a, true);"},
		{"in_array call arg", nil, "<?php\n    in_array(1, foo());", "<?php\n    in_array(1, foo(), true);"},
		{"in_array array()", nil, "<?php\n    in_array(1, array(1, 2, 3));", "<?php\n    in_array(1, array(1, 2, 3), true);"},
		{"in_array short array", nil, "<?php\n    in_array(1, [1, 2, 3]);", "<?php\n    in_array(1, [1, 2, 3], true);"},
		{"nested in_array", nil,
			"<?php\n    in_array(in_array(1, [1, in_array(1, [1, 2, 3]) ? 21 : 22, 3]) ? 111 : 222, [1, in_array(1, [1, 2, 3]) ? 21 : 22, 3]);",
			"<?php\n    in_array(in_array(1, [1, in_array(1, [1, 2, 3], true) ? 21 : 22, 3], true) ? 111 : 222, [1, in_array(1, [1, 2, 3], true) ? 21 : 22, 3], true);"},
		{"case insensitive", nil, "<?php\n    in_Array(1, $a);", "<?php\n    in_Array(1, $a, true);"},
		{"base64_decode already", nil,
			"<?php\n    base64_decode($foo, true);\n    base64_decode($foo, false);\n    base64_decode($foo, $useStrict);",
			"<?php\n    base64_decode($foo, true);\n    base64_decode($foo, false);\n    base64_decode($foo, $useStrict);"},
		{"base64_decode add", nil, "<?php\n    base64_decode($foo);", "<?php\n    base64_decode($foo, true);"},
		{"array_search already", nil,
			"<?php\n    array_search($foo, $bar, true);\n    array_search($foo, $bar, false);\n    array_search($foo, $bar, $useStrict);",
			"<?php\n    array_search($foo, $bar, true);\n    array_search($foo, $bar, false);\n    array_search($foo, $bar, $useStrict);"},
		{"array_search add", nil, "<?php\n    array_search($foo, $bar);", "<?php\n    array_search($foo, $bar, true);"},
		{"array_keys missing earlier", nil,
			"<?php\n    array_keys($foo);\n    array_keys($foo, $bar, true);\n    array_keys($foo, $bar, false);\n    array_keys($foo, $bar, $useStrict);",
			"<?php\n    array_keys($foo);\n    array_keys($foo, $bar, true);\n    array_keys($foo, $bar, false);\n    array_keys($foo, $bar, $useStrict);"},
		{"array_keys add", nil, "<?php\n    array_keys($foo, $bar);", "<?php\n    array_keys($foo, $bar, true);"},
		{"mb_detect_encoding already", nil,
			"<?php\n    mb_detect_encoding($foo, $bar, true);\n    mb_detect_encoding($foo, $bar, false);\n    mb_detect_encoding($foo, $bar, $useStrict);",
			"<?php\n    mb_detect_encoding($foo, $bar, true);\n    mb_detect_encoding($foo, $bar, false);\n    mb_detect_encoding($foo, $bar, $useStrict);"},
		{"mb_detect_encoding add both", nil, "<?php\n    mb_detect_encoding($foo);", "<?php\n    mb_detect_encoding($foo, mb_detect_order(), true);"},
		{"namespaced", nil,
			"<?php\n    namespace Foo {\n        array_keys($foo, $bar);\n    }\n    namespace Bar {\n        use function Foo\\LoremIpsum;\n        array_keys($foo, $bar);\n    }",
			"<?php\n    namespace Foo {\n        array_keys($foo, $bar, true);\n    }\n    namespace Bar {\n        use function Foo\\LoremIpsum;\n        array_keys($foo, $bar, true);\n    }"},
		{"trailing comma", nil, "<?php in_array($b, $c, );", "<?php in_array($b, $c, true, );"},
		{"comment after arg", nil, "<?php\n    in_array(1, foo() /* 1 *//* 2 *//* 3 */);", "<?php\n    in_array(1, foo(), true /* 1 *//* 2 *//* 3 */);"},
		{"trailing comma comments", nil, "<?php in_array($b, $c/* 0 *//* 1 */,/* 2 *//* 3 */);", "<?php in_array($b, $c/* 0 *//* 1 */, true,/* 2 *//* 3 */);"},
	})
}
