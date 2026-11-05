package rules

import "testing"

// dedWrap wraps a method body in a PHPUnit test class, matching the upstream
// fixer test's generateTest helper.
func dedWrap(body string) string {
	return "<?php\nfinal class FooTest extends \\PHPUnit\\Framework\\TestCase {\n    public function testSomething() {\n        " + body + "\n    }\n}\n"
}

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_dedicate_assert, default target).
func TestPhpUnitDedicateAssertPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitDedicateAssert{}, []aliasUpstreamCase{
		{
			"generic is_* to assertInternalType, positive and negative", nil,
			dedWrap("$this->assertTrue(is_array($var));\n        $this->assertFalse(is_array($var));\n        $this->assertTrue(is_float( $a), \"my message\");"),
			dedWrap("$this->assertInternalType('array', $var);\n        $this->assertNotInternalType('array', $var);\n        $this->assertInternalType('float', $a, \"my message\");"),
		},
		{
			"generic keeps an inner trailing comma case tidy", nil,
			dedWrap("$this->assertTrue(is_array($a,),);"),
			dedWrap("$this->assertInternalType('array', $a,);"),
		},
		{
			"is_null maps to assertNull / assertNotNull, namespaced too", nil,
			dedWrap("$this->assertTrue(is_null($a));\n        $this->assertFalse(is_null($a));\n        $this->assertTrue(\\is_null($a));"),
			dedWrap("$this->assertNull($a);\n        $this->assertNotNull($a);\n        $this->assertNull($a);"),
		},
		{
			"empty maps to assertEmpty / assertNotEmpty", nil,
			dedWrap("$this->assertTrue(empty($a));\n        $this->assertFalse(empty($a));"),
			dedWrap("$this->assertEmpty($a);\n        $this->assertNotEmpty($a);"),
		},
		{
			"file_exists and is_dir", nil,
			dedWrap("$this->assertTrue(file_exists($a));\n        $this->assertFalse(file_exists($a));\n        $this->assertTrue(is_dir($a));\n        $this->assertFalse(is_dir($a));"),
			dedWrap("$this->assertFileExists($a);\n        $this->assertFileNotExists($a);\n        $this->assertDirectoryExists($a);\n        $this->assertDirectoryNotExists($a);"),
		},
		{
			"is_infinite, is_nan (no negative), is_readable, is_writable", nil,
			dedWrap("$this->assertTrue(is_infinite($a));\n        $this->assertFalse(is_infinite($a));\n        $this->assertTrue(is_nan($a));\n        $this->assertTrue(is_readable($a));\n        $this->assertFalse(is_writable($a));"),
			dedWrap("$this->assertInfinite($a);\n        $this->assertFinite($a);\n        $this->assertNan($a);\n        $this->assertIsReadable($a);\n        $this->assertNotIsWritable($a);"),
		},
		{
			"array_key_exists takes two arguments", nil,
			dedWrap("$this->assertTrue(array_key_exists(\"test\", $a));\n        $this->assertFalse(array_key_exists($b, $a));"),
			dedWrap("$this->assertArrayHasKey(\"test\", $a);\n        $this->assertArrayNotHasKey($b, $a);"),
		},
		{
			"str_contains / str_starts_with / str_ends_with swap arguments", nil,
			dedWrap("self::assertTrue(str_contains($haystack, $needle));\n        self::assertFalse(str_contains($haystack, $needle));\n        self::assertTrue(str_starts_with($haystack, $needle));\n        self::assertTrue(str_ends_with($haystack, $needle));"),
			dedWrap("self::assertStringContainsString($needle, $haystack);\n        self::assertStringNotContainsString($needle, $haystack);\n        self::assertStringStartsWith($needle, $haystack);\n        self::assertStringEndsWith($needle, $haystack);"),
		},
		{
			"assertSame / assertEquals over count or sizeof become assertCount", nil,
			dedWrap("$this->assertSame(1, count($a));\n        $this->assertEquals(2, count($b));\n        $this->assertNotSame(3, count($c));\n        $this->assertNotEquals(4, sizeof($d));\n        $this->assertSame($b, \\count($a));"),
			dedWrap("$this->assertCount(1, $a);\n        $this->assertCount(2, $b);\n        $this->assertNotCount(3, $c);\n        $this->assertNotCount(4, $d);\n        $this->assertCount($b, $a);"),
		},
		{
			"instanceof becomes assertInstanceOf / assertNotInstanceOf", nil,
			dedWrap("$this->assertTrue($x instanceof SomeClass);\n        $this->assertTrue(!$x instanceof SomeClass);\n        $this->assertTrue($abc instanceof $tty);\n        self::assertFalse($a instanceof Foo);\n        self::assertFalse(!$b instanceof Foo);"),
			dedWrap("$this->assertInstanceOf(SomeClass::class, $x);\n        $this->assertNotInstanceOf(SomeClass::class, $x);\n        $this->assertInstanceOf($tty, $abc);\n        self::assertNotInstanceOf(Foo::class, $a);\n        self::assertInstanceOf(Foo::class, $b);"),
		},
		{
			"is_resource is left alone", nil,
			dedWrap("self::assertTrue(is_resource($resource));"),
			dedWrap("self::assertTrue(is_resource($resource));"),
		},
		{
			"wrong argument count, non-dedicated negative and non-same-class calls are left alone", nil,
			dedWrap("static::assertTrue(is_null($a, $b));\n        $this->assertFalse(is_nan($a));\n        $obj->assertTrue(is_array($var));\n        $this->assertSame(2, count($array) - 1);"),
			dedWrap("static::assertTrue(is_null($a, $b));\n        $this->assertFalse(is_nan($a));\n        $obj->assertTrue(is_array($var));\n        $this->assertSame(2, count($array) - 1);"),
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertTrue(is_array($var));\n    }\n}\n",
			"<?php\nclass Foo {\n    public function t() {\n        $this->assertTrue(is_array($var));\n    }\n}\n",
		},
	})
}
