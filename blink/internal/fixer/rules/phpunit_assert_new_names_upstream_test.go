package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_assert_new_names).
func TestPhpUnitAssertNewNamesPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitAssertNewNames{}, []aliasUpstreamCase{
		{
			"all renames", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        $this->assertFileNotExists(\"a\");\n        $this->assertNotIsWritable(\"b\");\n        $this->assertRegExp(\"/x/\", $s);\n        $this->assertNotRegExp(\"/x/\", $s);\n        $this->assertDirectoryNotExists(\"d\");\n        $this->assertSame(1, 2);\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        $this->assertFileDoesNotExist(\"a\");\n        $this->assertIsNotWritable(\"b\");\n        $this->assertMatchesRegularExpression(\"/x/\", $s);\n        $this->assertDoesNotMatchRegularExpression(\"/x/\", $s);\n        $this->assertDirectoryDoesNotExist(\"d\");\n        $this->assertSame(1, 2);\n    }\n}\n",
		},
		{
			"self and static receivers", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        self::assertFileNotExists(\"a\");\n        static::assertRegExp(\"/x/\", $s);\n        $obj->assertFileNotExists(\"c\");\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        self::assertFileDoesNotExist(\"a\");\n        static::assertMatchesRegularExpression(\"/x/\", $s);\n        $obj->assertFileNotExists(\"c\");\n    }\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertFileNotExists(\"a\");\n    }\n}\n",
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertFileNotExists(\"a\");\n    }\n}\n",
		},
	})
}
