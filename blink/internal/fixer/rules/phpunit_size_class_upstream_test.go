package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_size_class).
func TestPhpUnitSizeClassPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitSizeClass{}, []aliasUpstreamCase{
		{
			"test class gets @small", nil,
			"<?php\nclass MyTest extends TestCase\n{\n}\n",
			"<?php\n\n/**\n * @small\n */\nclass MyTest extends TestCase\n{\n}\n",
		},
		{
			"@small added to a single-line docblock", nil,
			"<?php\n/** Foo */\nclass MyTest extends TestCase\n{\n}\n",
			"<?php\n/**\n * Foo\n * @small\n */\nclass MyTest extends TestCase\n{\n}\n",
		},
		{
			"group option changes the annotation", map[string]any{"group": "medium"},
			"<?php\nclass MyTest extends TestCase\n{\n}\n",
			"<?php\n\n/**\n * @medium\n */\nclass MyTest extends TestCase\n{\n}\n",
		},
		{
			"existing size annotation is untouched", nil,
			"<?php\n/**\n * @medium\n */\nclass MyTest extends TestCase\n{\n}\n",
			"<?php\n/**\n * @medium\n */\nclass MyTest extends TestCase\n{\n}\n",
		},
		{
			"abstract test class is untouched", nil,
			"<?php\nabstract class MyTest extends TestCase\n{\n}\n",
			"<?php\nabstract class MyTest extends TestCase\n{\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo extends Bar\n{\n}\n",
			"<?php\nclass Foo extends Bar\n{\n}\n",
		},
	})
}
