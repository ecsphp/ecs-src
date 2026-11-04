package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_dedicate_assert_internal_type).
func TestPhpUnitDedicateAssertInternalTypePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitDedicateAssertInternalType{}, []aliasUpstreamCase{
		{
			"mapped types, negation, unknown and malformed left", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function testMe() {\n        $this->assertInternalType(\"array\", $var);\n        $this->assertNotInternalType(\"array\", $var);\n        $this->assertInternalType('float', $var);\n        $this->assertInternalType(\"unknownType\", $var);\n        $obj->assertInternalType(\"array\", $var);\n        $this->assertInternalType(\"array\");\n        $this->assertInternalType($var, $x);\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function testMe() {\n        $this->assertIsArray($var);\n        $this->assertIsNotArray($var);\n        $this->assertIsFloat($var);\n        $this->assertInternalType(\"unknownType\", $var);\n        $obj->assertIsArray($var);\n        $this->assertInternalType(\"array\");\n        $this->assertInternalType($var, $x);\n    }\n}\n",
		},
		{
			"null maps to assertNull / assertNotNull", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function testMe() {\n        $this->assertInternalType(\"null\", $var);\n        $this->assertNotInternalType(\"null\", $var);\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function testMe() {\n        $this->assertNull($var);\n        $this->assertNotNull($var);\n    }\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertInternalType(\"array\", $var);\n    }\n}\n",
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertInternalType(\"array\", $var);\n    }\n}\n",
		},
	})
}
