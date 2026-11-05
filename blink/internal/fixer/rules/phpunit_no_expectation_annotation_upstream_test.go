package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_no_expectation_annotation), default config.
func TestPhpUnitNoExpectationAnnotationPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitNoExpectationAnnotation{}, []aliasUpstreamCase{
		{
			"expecting exception", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        public function testFnc()\n        {\n            $this->setExpectedException(\\FooException::class);\n\n            aaa();\n        }\n    }",
		},
		{
			"expecting rooted exception", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException \\FooException\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        public function testFnc()\n        {\n            $this->setExpectedException(\\FooException::class);\n\n            aaa();\n        }\n    }",
		},
		{
			"empty exception message", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         * @expectedExceptionMessage\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        public function testFnc()\n        {\n            $this->setExpectedException(\\FooException::class, '');\n\n            aaa();\n        }\n    }",
		},
		{
			"exception with message", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         * @expectedExceptionMessage foo@bar\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        public function testFnc()\n        {\n            $this->setExpectedException(\\FooException::class, 'foo@bar');\n\n            aaa();\n        }\n    }",
		},
		{
			"exception with code", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         * @expectedExceptionCode 123\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        public function testFnc()\n        {\n            $this->setExpectedException(\\FooException::class, null, 123);\n\n            aaa();\n        }\n    }",
		},
		{
			"exception with message and code", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         * @expectedExceptionMessage foo\n         * @expectedExceptionCode 123\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        public function testFnc()\n        {\n            $this->setExpectedException(\\FooException::class, 'foo', 123);\n\n            aaa();\n        }\n    }",
		},
		{
			"exception with message regexp", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         * @expectedExceptionMessageRegExp /foo.*$/\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        public function testFnc()\n        {\n            $this->setExpectedExceptionRegExp(\\FooException::class, '/foo.*$/');\n\n            aaa();\n        }\n    }",
		},
		{
			"keep rest of docblock", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * Summary.\n         *\n         * @param int $param\n         * @expectedException FooException\n         * @return void\n         */\n        public function testFnc($param)\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * Summary.\n         *\n         * @param int $param\n         * @return void\n         */\n        public function testFnc($param)\n        {\n            $this->setExpectedException(\\FooException::class);\n\n            aaa();\n        }\n    }",
		},
		{
			"method without visibility", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         */\n        function testFnc($param)\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        function testFnc($param)\n        {\n            $this->setExpectedException(\\FooException::class);\n\n            aaa();\n        }\n    }",
		},
		{
			"final method", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         */\n        final function testFnc($param)\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        final function testFnc($param)\n        {\n            $this->setExpectedException(\\FooException::class);\n\n            aaa();\n        }\n    }",
		},
		{
			"respect quotes in message", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * Summary.\n         *\n         * @expectedException FooException\n         * @expectedExceptionMessage Foo \\' bar \" baz\n         */\n        public function testFnc($param)\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * Summary.\n         *\n         */\n        public function testFnc($param)\n        {\n            $this->setExpectedException(\\FooException::class, 'Foo \\\\\\' bar \" baz');\n\n            aaa();\n        }\n    }",
		},
		{
			"docblock for property not method is untouched", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         * @expectedExceptionCode 123\n         */\n         public $foo;\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException FooException\n         * @expectedExceptionCode 123\n         */\n         public $foo;\n    }",
		},
		{
			"annotation with double at", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * Double \"@\" is/was below\n         * @@expectedException FooException\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * Double \"@\" is/was below\n         */\n        public function testFnc()\n        {\n            $this->setExpectedException(\\FooException::class);\n\n            aaa();\n        }\n    }",
		},
		{
			"special backslash handling", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedException \\Cake\\View\\Exception\\MissingElementException\n         * @expectedExceptionMessage A backslash at the end \\\n         */\n        public function testElementNonExistentOne()\n        {\n            $this->View->element('non_existent_element');\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         */\n        public function testElementNonExistentOne()\n        {\n            $this->setExpectedException(\\Cake\\View\\Exception\\MissingElementException::class, 'A backslash at the end \\\\');\n\n            $this->View->element('non_existent_element');\n        }\n    }",
		},
		{
			"no expectedException annotation is untouched", nil,
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedExceptionMessage foo\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
			"<?php\n    final class MyTest extends \\PHPUnit_Framework_TestCase\n    {\n        /**\n         * @expectedExceptionMessage foo\n         */\n        public function testFnc()\n        {\n            aaa();\n        }\n    }",
		},
	})
}
