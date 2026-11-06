package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_doc_to_param_type).
func TestPhpdocToParamTypePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpdocToParamType{}, []aliasUpstreamCase{
		{
			"string and nullable params", nil,
			"<?php\n/**\n * @param string $foo\n * @param string|null $bar\n */\nfunction f($foo, $bar)\n{}\n",
			"<?php\n/**\n * @param string $foo\n * @param string|null $bar\n */\nfunction f(string $foo, ?string $bar)\n{}\n",
		},
		{
			"class type", nil,
			"<?php\n/** @param Foo $foo */\nfunction foo($foo) {}\n",
			"<?php\n/** @param Foo $foo */\nfunction foo(Foo $foo) {}\n",
		},
		{
			"scalar_types false keeps scalar", map[string]any{"scalar_types": false},
			"<?php\n/** @param Foo $foo */\nfunction foo($foo) {}\n/** @param string $foo */\nfunction bar($foo) {}\n",
			"<?php\n/** @param Foo $foo */\nfunction foo(Foo $foo) {}\n/** @param string $foo */\nfunction bar($foo) {}\n",
		},
		{
			"union_types false keeps union", map[string]any{"union_types": false},
			"<?php\n/** @param Foo $foo */\nfunction foo($foo) {}\n/** @param int|string $foo */\nfunction bar($foo) {}\n",
			"<?php\n/** @param Foo $foo */\nfunction foo(Foo $foo) {}\n/** @param int|string $foo */\nfunction bar($foo) {}\n",
		},
		{
			"union default", nil,
			"<?php\n/** @param int|string $foo */\nfunction bar($foo) {}\n",
			"<?php\n/** @param int|string $foo */\nfunction bar(int|string $foo) {}\n",
		},
		{
			"no docblock noop", nil,
			"<?php\nfunction f($foo) {}\n",
			"<?php\nfunction f($foo) {}\n",
		},
		{
			"by reference", nil,
			"<?php\n/** @param int $foo */\nfunction f(&$foo) {}\n",
			"<?php\n/** @param int $foo */\nfunction f(int &$foo) {}\n",
		},
		{
			"variadic", nil,
			"<?php\n/** @param int ...$foo */\nfunction f(...$foo) {}\n",
			"<?php\n/** @param int ...$foo */\nfunction f(...$foo) {}\n",
		},
		{
			"already typed noop", nil,
			"<?php\n/** @param int $foo */\nfunction f(string $foo) {}\n",
			"<?php\n/** @param int $foo */\nfunction f(string $foo) {}\n",
		},
		{
			"void type skipped", nil,
			"<?php\n/** @param void $foo */\nfunction f($foo) {}\n",
			"<?php\n/** @param void $foo */\nfunction f($foo) {}\n",
		},
		{
			"fully qualified class", nil,
			"<?php\n/** @param \\My\\Bar $foo */\nfunction f($foo) {}\n",
			"<?php\n/** @param \\My\\Bar $foo */\nfunction f(\\My\\Bar $foo) {}\n",
		},
		{
			"mixed no question mark", nil,
			"<?php\n/** @param mixed $foo */\nfunction f($foo) {}\n",
			"<?php\n/** @param mixed $foo */\nfunction f(mixed $foo) {}\n",
		},
		{
			"arrow function", nil,
			"<?php\n/** @param int $x */\n$f = fn($x) => $x;\n",
			"<?php\n/** @param int $x */\n$f = fn($x) => $x;\n",
		},
		{
			"array type", nil,
			"<?php\n/** @param int[] $foo */\nfunction f($foo) {}\n",
			"<?php\n/** @param int[] $foo */\nfunction f(array $foo) {}\n",
		},
		{
			"arrow function reachable docblock", nil,
			"<?php\n/** @param int $x */\nfn($x) => $x;\n",
			"<?php\n/** @param int $x */\nfn(int $x) => $x;\n",
		},
	})
}
