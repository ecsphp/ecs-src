package rules

import "testing"

// smcWrap wraps a method body in a PHPUnit test class.
func smcWrap(body string) string {
	return "<?php\nclass MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testMe() {\n        " + body + "\n    }\n}\n"
}

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_test_case_static_method_calls, default call_type: static).
func TestPhpUnitTestCaseStaticMethodCallsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitTestCaseStaticMethodCalls{}, []aliasUpstreamCase{
		{
			"default normalizes this/self to static, leaves static alone", nil,
			smcWrap("$this->assertSame(1, 2);\n        self::markTestIncomplete('foo');\n        static::fail('foo');"),
			smcWrap("static::assertSame(1, 2);\n        static::markTestIncomplete('foo');\n        static::fail('foo');"),
		},
		{
			"$this:: with a double colon also becomes static", nil,
			smcWrap("$this::assertTrue(true);"),
			smcWrap("static::assertTrue(true);"),
		},
		{
			"non-PHPUnit methods, property access and other receivers are left alone", nil,
			smcWrap("$this->createMock(Foo::class);\n        $this->assertSame = 42;\n        $obj->assertSame(1, 2);\n        Other::assertSame(1, 2);"),
			smcWrap("$this->createMock(Foo::class);\n        $this->assertSame = 42;\n        $obj->assertSame(1, 2);\n        Other::assertSame(1, 2);"),
		},
		{
			"a first-class callable reference is left alone", nil,
			smcWrap("$this->assertTrue(...);"),
			smcWrap("$this->assertTrue(...);"),
		},
		{
			"calls inside a lambda body are left alone", nil,
			smcWrap("$this->assertSame(1, 2);\n        $cb = function () {\n            $this->assertSame(1, 2);\n            self::assertSame(1, 2);\n        };"),
			smcWrap("static::assertSame(1, 2);\n        $cb = function () {\n            $this->assertSame(1, 2);\n            self::assertSame(1, 2);\n        };"),
		},
		{
			"calls inside an anonymous class are left alone", nil,
			"<?php\nclass MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testMe() {\n        $this->assertSame(1, 2);\n        $foo = new class() {\n            public function assertSame($a, $b) {\n                $this->assertSame(1, 2);\n            }\n        };\n    }\n}\n",
			"<?php\nclass MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testMe() {\n        static::assertSame(1, 2);\n        $foo = new class() {\n            public function assertSame($a, $b) {\n                $this->assertSame(1, 2);\n            }\n        };\n    }\n}\n",
		},
		{
			"call_type self rewrites this/static to self", map[string]any{"call_type": "self"},
			smcWrap("$this->assertSame(1, 2);\n        static::markTestIncomplete('foo');\n        self::fail('foo');"),
			smcWrap("self::assertSame(1, 2);\n        self::markTestIncomplete('foo');\n        self::fail('foo');"),
		},
		{
			"call_type this skips static methods but rewrites instance methods", map[string]any{"call_type": "this"},
			"<?php\nclass MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testMe() {\n        self::assertSame(1, 2);\n    }\n    public static function helper() {\n        static::assertSame(1, 2);\n    }\n}\n",
			"<?php\nclass MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testMe() {\n        $this->assertSame(1, 2);\n    }\n    public static function helper() {\n        static::assertSame(1, 2);\n    }\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertSame(1, 2);\n    }\n}\n",
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertSame(1, 2);\n    }\n}\n",
		},
	})
}
