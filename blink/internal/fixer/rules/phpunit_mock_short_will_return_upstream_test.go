package rules

import "testing"

// TestPhpUnitMockShortWillReturnUpstream runs PHP-CS-Fixer's own PhpUnitMockShortWillReturnFixerTest cases.
func TestPhpUnitMockShortWillReturnUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpUnitMockShortWillReturn{}, []upstreamCase{
		{
			name:     "provideFixCases: do not fix",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->will;\n        $someMock->method(\"someMethod\")->will(\"Smith\");\n        $someMock->method(\"someMethod\")->will($this->returnSelf);\n        $someMock->method(\"someMethod\")->will($this->doSomething(7));\n    }\n}",
		},
		{
			name:     "provideFixCases: will return simple scenarios",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->willReturn(10);\n        $someMock->method(\"someMethod\")->willReturn(20);\n        $someMock->method(\"someMethod\")->willReturn(30);\n        $someMock->method(\"someMethod\")->willReturn(40);\n        $someMock->method(\"someMethod\")->willReturn(50);\n        $someMock->method(\"someMethod\")->willReturn(60);\n        $someMock->method(\"someMethod\")->willReturn(-10);\n        $someMock->method(\"someMethod\")->willReturn(10.10);\n        $someMock->method(\"someMethod\")->willReturn(-10.10);\n        $someMock->method(\"someMethod\")->willReturn(\"myValue\");\n        $someMock->method(\"someMethod\")->willReturn($myValue);\n        $testMock->method(\"test_method\")->willReturn(DEFAULT_VALUE);\n        $testMock->method(\"test_method\")->willReturn(self::DEFAULT_VALUE);\n        $someMock->method(\"someMethod\")->willReturn([]);\n        $someMock->method(\"someMethod\")->willReturn([[]]);\n        $someMock->method(\"someMethod\")->willReturn(array());\n        $someMock->method(\"someMethod\")->willReturn(new stdClass());\n        $someMock->method(\"someMethod\")->willReturn(new \\DateTime());\n        $someMock->method(\"someMethod\")->willReturnSelf();\n        $someMock->method(\"someMethod\")->willReturnArgument(2);\n        $someMock->method(\"someMethod\")->willReturnCallback(\"str_rot13\");\n        $someMock->method(\"someMethod\")->willReturnMap([\"a\", \"b\", \"c\", \"d\"]);\n        $someMock->method(\"someMethod\")->willReturn(1);\n        $someMock->method(\"someMethod\")->willReturn(2);\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->will($this->returnValue(10));\n        $someMock->method(\"someMethod\")->will($THIS->returnValue(20));\n        $someMock->method(\"someMethod\")->will(self::returnValue(30));\n        $someMock->method(\"someMethod\")->will(Self::returnValue(40));\n        $someMock->method(\"someMethod\")->will(static::returnValue(50));\n        $someMock->method(\"someMethod\")->will(STATIC::returnValue(60));\n        $someMock->method(\"someMethod\")->will($this->returnValue(-10));\n        $someMock->method(\"someMethod\")->will($this->returnValue(10.10));\n        $someMock->method(\"someMethod\")->will($this->returnValue(-10.10));\n        $someMock->method(\"someMethod\")->will($this->returnValue(\"myValue\"));\n        $someMock->method(\"someMethod\")->will($this->returnValue($myValue));\n        $testMock->method(\"test_method\")->will($this->returnValue(DEFAULT_VALUE));\n        $testMock->method(\"test_method\")->will($this->returnValue(self::DEFAULT_VALUE));\n        $someMock->method(\"someMethod\")->will($this->returnValue([]));\n        $someMock->method(\"someMethod\")->will($this->returnValue([[]]));\n        $someMock->method(\"someMethod\")->will($this->returnValue(array()));\n        $someMock->method(\"someMethod\")->will($this->returnValue(new stdClass()));\n        $someMock->method(\"someMethod\")->will($this->returnValue(new \\DateTime()));\n        $someMock->method(\"someMethod\")->will($this->returnSelf());\n        $someMock->method(\"someMethod\")->will($this->returnArgument(2));\n        $someMock->method(\"someMethod\")->will($this->returnCallback(\"str_rot13\"));\n        $someMock->method(\"someMethod\")->will($this->returnValueMap([\"a\", \"b\", \"c\", \"d\"]));\n        $someMock->method(\"someMethod\")->WILL($this->returnValue(1));\n        $someMock->method(\"someMethod\")->will($this->ReturnVALUE(2));\n    }\n}",
		},
		{
			name:     "provideFixCases: will return with multi lines and messy indents",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock\n        ->method(\"someMethod\")\n            ->willReturn(\n                10\n            );\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock\n        ->method(\"someMethod\")\n            ->will(\n                $this->returnValue(10)\n            );\n    }\n}",
		},
		{
			name:     "provideFixCases: will return with multi lines, messy indents and comments inside",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock\n        ->method(\"someMethod\")\n            ->willReturn(\n                // foo\n                10\n                // bar\n            );\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock\n        ->method(\"someMethod\")\n            ->will(\n                // foo\n                $this->returnValue(10)\n                // bar\n            );\n    }\n}",
		},
		{
			name:     "provideFixCases: will return with block comments in weird places",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->/* a */willReturn/* b */(/* c */ 10 /* d */);\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->/* a */will/* b */(/* c */ $this->returnValue(10) /* d */);\n    }\n}",
		},
		{
			name:     "provideFixCases: will return with comments persisted not touched even if put in unexpected places",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")// a\n            ->/* b */willReturn/* c */(/* d */ /** e */\n             // f\n            \n            // g\n            \n            /* h */\n            10 /* i */);\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")// a\n            ->/* b */will/* c */(/* d */ $this/** e */\n            -> // f\n            returnValue\n            // g\n            (\n            /* h */\n            10) /* i */);\n    }\n}",
		},
		{
			name:     "provideFixCases: will return with multi lines, messy indents and comments in weird places",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock\n        ->method(\n        \"someMethod\"\n        )\n              ->\n\n              /* a */\n        willReturn\n                /*\n                b\n                c\n                d\n                e\n        */        (\n                    // f g h i\n                    /* j */ \n        \n            10\n             /* k */\n             /* l */);\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock\n        ->method(\n        \"someMethod\"\n        )\n              ->\n\n              /* a */\n        will\n                /*\n                b\n                c\n                d\n                e\n        */        (\n                    // f g h i\n                    /* j */ $this\n        ->returnValue\n            (10)\n             /* k */\n             /* l */);\n    }\n}",
		},
		{
			name:     "provideFixCases: case 8",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->willReturn( 10 , );\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->will($this->returnValue( 10 , ));\n    }\n}",
		},
		{
			name:     "provideFixCases: with trailing commas",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->willReturn( 10 ,   );\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock->method(\"someMethod\")->will($this->returnValue( 10 , ) , );\n    }\n}",
		},
	})
}
