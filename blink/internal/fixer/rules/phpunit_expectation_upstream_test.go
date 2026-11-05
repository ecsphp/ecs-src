package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_expectation), default config.
func TestPhpUnitExpectationPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitExpectation{}, []aliasUpstreamCase{
		{
			"inline comments B and C", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this->setExpectedException('RuntimeException', 'msg'/*B*/, /*C*/123);\n            zzz();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this->expectException('RuntimeException');\n            $this->expectExceptionMessage('msg'/*B*/);\n            $this->expectExceptionCode(/*C*/123);\n            zzz();\n        }\n    }",
		},
		{
			"null message with surrounding comments", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this->setExpectedException('RuntimeException',/*B*/ null /*B2*/,/*C*/123);\n            zzz();\n        }\n        function testFnc2()\n        {\n            aaa();\n            $this->setExpectedException('RuntimeException',/*B*/ null /*B2*/ + 1,/*C*/123);\n            zzz();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this->expectException('RuntimeException'/*B*/  /*B2*/);\n            $this->expectExceptionCode(/*C*/123);\n            zzz();\n        }\n        function testFnc2()\n        {\n            aaa();\n            $this->expectException('RuntimeException');\n            $this->expectExceptionMessage(/*B*/ null /*B2*/ + 1);\n            $this->expectExceptionCode(/*C*/123);\n            zzz();\n        }\n    }",
		},
		{
			"single multiline argument", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            $this->setExpectedException(\n                \\Exception::class\n            );\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            $this->expectException(\n                \\Exception::class\n            );\n        }\n    }",
		},
		{
			"multiline three arguments", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            $this->setExpectedException(\n                \\Exception::class,\n                \"foo\",\n                123\n            );\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            $this->expectException(\n                \\Exception::class\n            );\n            $this->expectExceptionMessage(\n                \"foo\"\n            );\n            $this->expectExceptionCode(\n                123\n            );\n        }\n    }",
		},
		{
			"newest default message code", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit_Framework_TestCase\n{\n    public function testFoo()\n    {\n        $this->setExpectedException(\"RuntimeException\", \"Msg\", 123);\n        foo();\n    }\n\n    public function testBar()\n    {\n        $this->setExpectedExceptionRegExp(\"RuntimeException\", \"/Msg.*/\", 123);\n        bar();\n    }\n}",
			"<?php\nfinal class MyTest extends \\PHPUnit_Framework_TestCase\n{\n    public function testFoo()\n    {\n        $this->expectException(\"RuntimeException\");\n        $this->expectExceptionMessage(\"Msg\");\n        $this->expectExceptionCode(123);\n        foo();\n    }\n\n    public function testBar()\n    {\n        $this->expectException(\"RuntimeException\");\n        $this->expectExceptionMessageMatches(\"/Msg.*/\");\n        $this->expectExceptionCode(123);\n        bar();\n    }\n}",
		},
		{
			"trailing comma", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this->setExpectedException(\"RuntimeException\", \"msg\"/*B*/, /*C*/123, );\n            zzz();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this->expectException(\"RuntimeException\");\n            $this->expectExceptionMessage(\"msg\"/*B*/);\n            $this->expectExceptionCode(/*C*/123);\n            zzz();\n        }\n    }",
		},
		{
			"trailing comma with comment", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this->setExpectedException(\"RuntimeException\", \"msg\"/*B*/, /*C*/123/*D*/);\n            zzz();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this->expectException(\"RuntimeException\");\n            $this->expectExceptionMessage(\"msg\"/*B*/);\n            $this->expectExceptionCode(/*C*/123/*D*/);\n            zzz();\n        }\n    }",
		},
		{
			"regexp method renamed to matches", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit_Framework_TestCase\n{\n    public function testFoo()\n    {\n        $this->expectExceptionMessageRegExp(\"/Msg.*/\");\n        foo();\n    }\n}",
			"<?php\nfinal class MyTest extends \\PHPUnit_Framework_TestCase\n{\n    public function testFoo()\n    {\n        $this->expectExceptionMessageMatches(\"/Msg.*/\");\n        foo();\n    }\n}",
		},
		{
			"nullsafe object operator", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this?->setExpectedException(\"RuntimeException\", \"message\", 123);\n            zzz();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        function testFnc()\n        {\n            aaa();\n            $this?->expectException(\"RuntimeException\");\n            $this->expectExceptionMessage(\"message\");\n            $this->expectExceptionCode(123);\n            zzz();\n        }\n    }",
		},
		{
			"not a phpunit class is untouched", nil,
			"<?php\nclass Foo\n{\n    public function t()\n    {\n        $this->setExpectedException(\"RuntimeException\", \"Msg\", 123);\n    }\n}",
			"<?php\nclass Foo\n{\n    public function t()\n    {\n        $this->setExpectedException(\"RuntimeException\", \"Msg\", 123);\n    }\n}",
		},
	})
}
