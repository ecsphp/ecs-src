package rules

import "testing"

// Cases ported from PHP-CS-Fixer's PhpdocAddMissingParamAnnotationFixerTest.
// The one upstream case using a custom whitespace config (tab indent, CRLF) is
// omitted: blink has no whitespace-config concept.
func TestPhpdocAddMissingParamAnnotationUpstreamCases(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpdocAddMissingParamAnnotation{}, []upstreamCase{
		{"case 0", map[string]any{"only_untyped": false}, "<?php\n    /**\n     *\n     */", ""},
		{"case 1", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param int $foo\n     * @param mixed $bar\n     */\n    function f1($foo, $bar) {}", "<?php\n    /**\n     * @param int $foo\n     */\n    function f1($foo, $bar) {}"},
		{"case 2", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param int $bar\n     * @param mixed $foo\n     */\n    function f2($foo, $bar) {}", "<?php\n    /**\n     * @param int $bar\n     */\n    function f2($foo, $bar) {}"},
		{"case 3", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @return void\n     * @param mixed $foo\n     * @param mixed $bar\n     */\n    function f3($foo, $bar) {}", "<?php\n    /**\n     * @return void\n     */\n    function f3($foo, $bar) {}"},
		{"case 4", map[string]any{"only_untyped": false}, "<?php\n    abstract class Foo {\n        /**\n         * @param int $bar\n         * @param mixed $foo\n         */\n        abstract public function f4a($foo, $bar);\n    }", "<?php\n    abstract class Foo {\n        /**\n         * @param int $bar\n         */\n        abstract public function f4a($foo, $bar);\n    }"},
		{"case 5", map[string]any{"only_untyped": false}, "<?php\n    class Foo {\n        /**\n         * @param int $bar\n         * @param mixed $foo\n         */\n        static final public function f4b($foo, $bar) {}\n    }", "<?php\n    class Foo {\n        /**\n         * @param int $bar\n         */\n        static final public function f4b($foo, $bar) {}\n    }"},
		{"case 6", map[string]any{"only_untyped": false}, "<?php\n    class Foo {\n        /**\n         * @var int\n         */\n        private $foo;\n    }", ""},
		{"case 7", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param $bar No type !!\n     * @param mixed $foo\n     */\n    function f5($foo, $bar) {}", "<?php\n    /**\n     * @param $bar No type !!\n     */\n    function f5($foo, $bar) {}"},
		{"case 8", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param int\n     * @param int $bar\n     * @param Foo\\Bar $foo\n     */\n    function f6(Foo\\Bar $foo, $bar) {}", "<?php\n    /**\n     * @param int\n     * @param int $bar\n     */\n    function f6(Foo\\Bar $foo, $bar) {}"},
		{"case 9", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param int $bar\n     * @param null|string $foo\n     */\n    function f7(string $foo = nuLl, $bar) {}", "<?php\n    /**\n     * @param int $bar\n     */\n    function f7(string $foo = nuLl, $bar) {}"},
		{"case 10", map[string]any{"only_untyped": true}, "<?php\n    /**\n     * @param int $bar\n     * @param mixed $baz\n     *\n     * @return void\n     */\n    function f9(string $foo, $bar, $baz) {}", "<?php\n    /**\n     * @param int $bar\n     *\n     * @return void\n     */\n    function f9(string $foo, $bar, $baz) {}"},
		{"case 11", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param bool|bool[] $caseSensitive Line 1\n     *                                   Line 2\n     */\n     function f11($caseSensitive) {}\n", ""},
		{"case 12", map[string]any{"only_untyped": false}, "<?php\n    /** @return string */\n    function hello($string)\n    {\n        return $string;\n    }", ""},
		{"case 13", map[string]any{"only_untyped": false}, "<?php\n    /** @return string\n     * @param mixed $string\n     */\n    function hello($string)\n    {\n        return $string;\n    }", "<?php\n    /** @return string\n     */\n    function hello($string)\n    {\n        return $string;\n    }"},
		{"case 14", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param mixed $string\n     * @return string */\n    function hello($string)\n    {\n        return $string;\n    }", "<?php\n    /**\n     * @return string */\n    function hello($string)\n    {\n        return $string;\n    }"},
		{"case 15", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param int $bar\n     * @param string $foo\n     */\n    function f8(string $foo = \"null\", $bar) {}", "<?php\n    /**\n     * @param int $bar\n     */\n    function f8(string $foo = \"null\", $bar) {}"},
		{"case 16", nil, "<?php\n    /**\n     * @{inheritdoc}\n     */\n    function f10(string $foo = \"null\", $bar) {}", ""},
		{"case 17", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @inheritDoc\n     */\n    function f10(string $foo = \"null\", $bar) {}", ""},
		{"case 18", map[string]any{"only_untyped": false}, "<?php\n    /**\n     * @param int $bar\n     * @param ?array $foo\n     */\n    function p1(?array $foo = null, $bar) {}", "<?php\n    /**\n     * @param int $bar\n     */\n    function p1(?array $foo = null, $bar) {}"},
		{"case 19", map[string]any{"only_untyped": true}, "<?php\n    /**\n     * Foo\n     * @param mixed $bar\n     */\n    function p1(?int $foo = 0, $bar) {}", "<?php\n    /**\n     * Foo\n     */\n    function p1(?int $foo = 0, $bar) {}"},
		{"case 20", map[string]any{"only_untyped": true}, "<?php\n    /**\n     * Foo\n     * @return int\n     */\n    function p1(?int $foo = 0) {}", ""},
		// skipped (custom whitespace config, not modeled)
		{"case 22", map[string]any{"only_untyped": false}, "<?php\n                    /**\n                     * something\n                     * @param mixed $numbers\n                     */\n                    function add(&$numbers) {}\n                ", "<?php\n                    /**\n                     * something\n                     */\n                    function add(&$numbers) {}\n                "},
		{"case 23", map[string]any{"only_untyped": false}, "<?php\n                    /**\n                     * something\n                     * @param null|array $numbers\n                     */\n                    function add(array &$numbers = null) {}\n                ", "<?php\n                    /**\n                     * something\n                     */\n                    function add(array &$numbers = null) {}\n                "},
		{"case 24", map[string]any{"only_untyped": false}, "<?php\n                    /**\n                     * something\n                     * @param array $numbers\n                     */\n                    function sum(...$numbers) {}\n                ", "<?php\n                    /**\n                     * something\n                     */\n                    function sum(...$numbers) {}\n                "},
		{"case 25", map[string]any{"only_untyped": false}, "<?php\n                    /**\n                     * @param int $a\n                     * @param array $numbers\n                     */\n                    function sum($a, ...$numbers) {}\n                ", "<?php\n                    /**\n                     * @param int $a\n                     */\n                    function sum($a, ...$numbers) {}\n                "},
		{"case 26", map[string]any{"only_untyped": false}, "<?php\n                    /**\n                     * @param \\Date[] $numbers\n                     */\n                    function sum(\\Date ...$numbers) {}\n                ", "<?php\n                    /**\n                     */\n                    function sum(\\Date ...$numbers) {}\n                "},
	})
}
