package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_mock).
func TestPhpUnitMockPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitMock{}, []aliasUpstreamCase{
		{
			"newest target", nil,
			"<?php\nfinal class MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testFoo() {\n        $a = $this->getMockWithoutInvokingTheOriginalConstructor(\"Foo\");\n        $b = $this->getMock(\"Foo\");\n        $c = $this->getMock(\"Bar\", [\"x\"]);\n        $d = $this->getMock(\"Baz\", [\"x\"], [\"y\"]);\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testFoo() {\n        $a = $this->createMock(\"Foo\");\n        $b = $this->createMock(\"Foo\");\n        $c = $this->createPartialMock(\"Bar\", [\"x\"]);\n        $d = $this->getMock(\"Baz\", [\"x\"], [\"y\"]);\n    }\n}\n",
		},
		{
			"target 5.4 keeps two-arg getMock", map[string]any{"target": "5.4"},
			"<?php\nfinal class MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testFoo() {\n        $b = $this->getMock(\"Foo\");\n        $c = $this->getMock(\"Bar\", [\"x\"]);\n    }\n}\n",
			"<?php\nfinal class MyTest extends \\PHPUnit_Framework_TestCase {\n    public function testFoo() {\n        $b = $this->createMock(\"Foo\");\n        $c = $this->getMock(\"Bar\", [\"x\"]);\n    }\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function t() {\n        $b = $this->getMock(\"Foo\");\n    }\n}\n",
			"<?php\nclass Foo {\n    public function t() {\n        $b = $this->getMock(\"Foo\");\n    }\n}\n",
		},
	})
}
