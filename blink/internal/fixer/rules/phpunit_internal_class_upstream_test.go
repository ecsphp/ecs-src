package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_internal_class).
func TestPhpUnitInternalClassPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitInternalClass{}, []aliasUpstreamCase{
		{
			"normal test class gets @internal", nil,
			"<?php\nclass MyTest extends TestCase\n{\n}\n",
			"<?php\n\n/**\n * @internal\n */\nclass MyTest extends TestCase\n{\n}\n",
		},
		{
			"final test class gets @internal", nil,
			"<?php\nfinal class MyTest extends TestCase\n{\n}\n",
			"<?php\n\n/**\n * @internal\n */\nfinal class MyTest extends TestCase\n{\n}\n",
		},
		{
			"abstract test class is untouched by default", nil,
			"<?php\nabstract class MyTest extends TestCase\n{\n}\n",
			"<?php\nabstract class MyTest extends TestCase\n{\n}\n",
		},
		{
			"@internal added to an existing docblock", nil,
			"<?php\n/**\n * Something.\n */\nclass MyTest extends TestCase\n{\n}\n",
			"<?php\n/**\n * Something.\n * @internal\n */\nclass MyTest extends TestCase\n{\n}\n",
		},
		{
			"class already marked @internal is untouched", nil,
			"<?php\n/**\n * @internal\n */\nclass MyTest extends TestCase\n{\n}\n",
			"<?php\n/**\n * @internal\n */\nclass MyTest extends TestCase\n{\n}\n",
		},
		{
			"abstract marked only when configured", map[string]any{"types": []string{"abstract"}},
			"<?php\nabstract class MyTest extends TestCase\n{\n}\n",
			"<?php\n\n/**\n * @internal\n */\nabstract class MyTest extends TestCase\n{\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo extends Bar\n{\n}\n",
			"<?php\nclass Foo extends Bar\n{\n}\n",
		},
	})
}
