package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_strict); every expected
// output is what the fixer actually produces for the input.
func TestPhpUnitStrictPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitStrict{}, []aliasUpstreamCase{
		{
			"two args, all four assertions", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function testFoo() {\n        $this->assertAttributeEquals(a(), b());\n        $this->assertAttributeNotEquals(a(), b());\n        $this->assertEquals(a(), b());\n        $this->assertNotEquals(a(), b());\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function testFoo() {\n        $this->assertAttributeSame(a(), b());\n        $this->assertAttributeNotSame(a(), b());\n        $this->assertSame(a(), b());\n        $this->assertNotSame(a(), b());\n    }\n}\n",
		},
		{
			"two and three args fixed, one and four left", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        $this->assertEquals($a);\n        $this->assertEquals($a, $b);\n        $this->assertEquals($a, $b, $c);\n        $this->assertEquals($a, $b, $c, $d);\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        $this->assertEquals($a);\n        $this->assertSame($a, $b);\n        $this->assertSame($a, $b, $c);\n        $this->assertEquals($a, $b, $c, $d);\n    }\n}\n",
		},
		{
			"self and static fixed, other receivers left", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        self::assertEquals(1, $a);\n        static::assertEquals(1, $a);\n        $obj->assertEquals(1, $a);\n        assertEquals(1, $a);\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        self::assertSame(1, $a);\n        static::assertSame(1, $a);\n        $obj->assertEquals(1, $a);\n        assertEquals(1, $a);\n    }\n}\n",
		},
		{
			"config limits to assertEquals", map[string]any{"assertions": []any{"assertEquals"}},
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        $this->assertEquals(1, $a);\n        $this->assertNotEquals(1, $a);\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit\\Framework\\TestCase {\n    public function t() {\n        $this->assertSame(1, $a);\n        $this->assertNotEquals(1, $a);\n    }\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertEquals(1, $a);\n    }\n}\n",
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertEquals(1, $a);\n    }\n}\n",
		},
	})
}
