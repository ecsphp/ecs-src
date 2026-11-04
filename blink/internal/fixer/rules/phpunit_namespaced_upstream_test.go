package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_namespaced).
func TestPhpUnitNamespacedPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitNamespaced{}, []aliasUpstreamCase{
		{
			"extends, static call, new and mapped class", nil,
			`<?php
final class MyTest extends \PHPUnit_Framework_TestCase
{
    public function testSomething()
    {
        PHPUnit_Framework_Assert::assertTrue(true);
        $x = new PHPUnit_Framework_TestCase();
        $y = PHPUnit_Util_XML::load($z);
    }
}
`,
			`<?php
final class MyTest extends \PHPUnit\Framework\TestCase
{
    public function testSomething()
    {
        PHPUnit\Framework\Assert::assertTrue(true);
        $x = new PHPUnit\Framework\TestCase();
        $y = PHPUnit\Util\Xml::load($z);
    }
}
`,
		},
		{
			"import makes later usages short; const and method call left", nil,
			`<?php
use PHPUnit_Framework_TestCase;

class MyTest extends PHPUnit_Framework_TestCase
{
    const PHPUnit_Framework_TestCase = 1;
    public function foo()
    {
        $this->PHPUnit_Framework_Assert();
        return PHPUnit_Framework_TestCase::class;
    }
}
`,
			`<?php
use PHPUnit\Framework\TestCase;

class MyTest extends TestCase
{
    const PHPUnit_Framework_TestCase = 1;
    public function foo()
    {
        $this->PHPUnit_Framework_Assert();
        return TestCase::class;
    }
}
`,
		},
		{
			"target 5.7 only converts the forward-compat classes", map[string]any{"target": "5.7"},
			`<?php
class MyTest extends PHPUnit_Framework_TestCase
{
    public function foo()
    {
        PHPUnit_Framework_Assert::assertTrue(true);
        $x = PHPUnit_Util_XML::load($z);
    }
}
`,
			`<?php
class MyTest extends PHPUnit\Framework\TestCase
{
    public function foo()
    {
        PHPUnit\Framework\Assert::assertTrue(true);
        $x = PHPUnit_Util_XML::load($z);
    }
}
`,
		},
	})
}
