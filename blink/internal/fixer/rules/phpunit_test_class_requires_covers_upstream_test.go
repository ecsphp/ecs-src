package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_test_class_requires_covers).
func TestPhpUnitTestClassRequiresCoversPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitTestClassRequiresCovers{}, []aliasUpstreamCase{
		{
			"test class without @covers gets @coversNothing", nil,
			"<?php\nfinal class MyTest extends TestCase\n{\n}\n",
			"<?php\n\n/**\n * @coversNothing\n */\nfinal class MyTest extends TestCase\n{\n}\n",
		},
		{
			"@coversNothing added to an existing docblock", nil,
			"<?php\n/**\n * Something.\n */\nfinal class MyTest extends TestCase\n{\n}\n",
			"<?php\n/**\n * Something.\n * @coversNothing\n */\nfinal class MyTest extends TestCase\n{\n}\n",
		},
		{
			"class with @covers is untouched", nil,
			"<?php\n/**\n * @covers \\Foo\n */\nfinal class MyTest extends TestCase\n{\n}\n",
			"<?php\n/**\n * @covers \\Foo\n */\nfinal class MyTest extends TestCase\n{\n}\n",
		},
		{
			"class already with @coversNothing is untouched", nil,
			"<?php\n/**\n * @coversNothing\n */\nfinal class MyTest extends TestCase\n{\n}\n",
			"<?php\n/**\n * @coversNothing\n */\nfinal class MyTest extends TestCase\n{\n}\n",
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
