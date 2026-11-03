package rules

import (
	"fmt"
	"strings"
	"testing"
)

// phpUnitConstructTest mirrors PhpUnitConstructFixerTest::generateTest.
func phpUnitConstructTest(content string) string {
	return "<?php final class FooTest extends \\PHPUnit_Framework_TestCase {\n    public function testSomething() {\n        " + content + "\n    }\n}\n"
}

// phpUnitConstructGeneratedCases mirrors PhpUnitConstructFixerTest::generateCases.
func phpUnitConstructGeneratedCases(expectedTemplate, inputTemplate string) []upstreamCase {
	methods := []struct {
		name     string
		positive bool
	}{{"Same", true}, {"NotSame", false}, {"Equals", true}, {"NotEquals", false}}
	var cases []upstreamCase
	for _, typ := range []string{"true", "false", "null"} {
		for _, m := range methods {
			not := "Not"
			if m.positive {
				not = ""
			}
			cases = append(cases, upstreamCase{
				name:     fmt.Sprintf("generated %s %s %s", inputTemplate, m.name, typ),
				expected: phpUnitConstructTest(fmt.Sprintf(expectedTemplate, not, strings.ToUpper(typ[:1])+typ[1:], m.name, typ)),
				input:    phpUnitConstructTest(fmt.Sprintf(inputTemplate, m.name, typ, m.name, typ)),
			})
		}
	}
	return cases
}

// TestPhpUnitConstructUpstream runs PHP-CS-Fixer's own
// PhpUnitConstructFixerTest cases.
func TestPhpUnitConstructUpstream(t *testing.T) {
	t.Parallel()
	wrapped := []struct{ expected, input string }{
		{"$sth->assertSame(true, $foo);", ""},
		{"$this->assertSame($b, null);", ""},
		{"$this->assertNull(/*bar*/ $a);", "$this->assertSame(null /*foo*/, /*bar*/ $a);"},
		{"$this->assertSame(null === $eventException ? $exception : $eventException, $event->getException());", ""},
		{"$this->assertSame(null /*comment*/ === $eventException ? $exception : $eventException, $event->getException());", ""},
		{"\n    $this->assertTrue(\n        $a,\n        \"foo\" . $bar\n    );", "\n    $this->assertSame(\n        true,\n        $a,\n        \"foo\" . $bar\n    );"},
		{"\n    $this->assertTrue(#\n        #\n        $a,#\n        \"foo\" . $bar#\n    );", "\n    $this->assertSame(#\n        true,#\n        $a,#\n        \"foo\" . $bar#\n    );"},
		{`$this->assertSame("a", $a); $this->assertTrue($b);`, `$this->assertSame("a", $a); $this->assertSame(true, $b);`},
		{`$this->assertSame(true || $a, $b); $this->assertTrue($c);`, `$this->assertSame(true || $a, $b); $this->assertSame(true, $c);`},
		{"$this->assertFalse($foo);", "$this->assertEquals(FALSE, $foo);"},
		{"$this->assertTrue($foo);", "$this->assertEquals(TruE, $foo);"},
		{"$this->assertNull($foo);", "$this->assertEquals(NULL, $foo);"},
		{"$this->assertTrue($a, );", "$this->assertSame(true, $a, );"},
		{"$this->assertTrue($a, $message , );", "$this->assertSame(true, $a, $message , );"},
		{"$this->assertEquals(...);", ""},
	}
	var cases []upstreamCase
	for i, w := range wrapped {
		c := upstreamCase{name: fmt.Sprintf("case %d", i), expected: phpUnitConstructTest(w.expected)}
		if w.input != "" {
			c.input = phpUnitConstructTest(w.input)
		}
		cases = append(cases, c)
	}
	cases = append(cases,
		upstreamCase{name: "not in a class", expected: "<?php $this->assertEquals(NULL, $foo);"},
		upstreamCase{name: "not phpunit class", expected: "<?php class Foo { public function testFoo(){ $this->assertEquals(NULL, $foo); }}"},
		upstreamCase{
			name: "multiple candidates in multiple classes",
			expected: `<?php
                        class FooTest1 extends PHPUnit_Framework_TestCase { public function testFoo(){ $this->assertNull($foo); }}
                        class FooTest2 extends PHPUnit_Framework_TestCase { public function testFoo(){ $this->assertNull($foo); }}
                        class FooTest3 extends PHPUnit_Framework_TestCase { public function testFoo(){ $this->assertNull($foo); }}
                    `,
			input: `<?php
                        class FooTest1 extends PHPUnit_Framework_TestCase { public function testFoo(){ $this->assertEquals(NULL, $foo); }}
                        class FooTest2 extends PHPUnit_Framework_TestCase { public function testFoo(){ $this->assertEquals(NULL, $foo); }}
                        class FooTest3 extends PHPUnit_Framework_TestCase { public function testFoo(){ $this->assertEquals(NULL, $foo); }}
                    `,
		},
		upstreamCase{name: "assertions option empty", cfg: map[string]any{"assertions": []any{}}, expected: phpUnitConstructTest("$this->assertSame(null, $a);")},
	)
	cases = append(cases, phpUnitConstructGeneratedCases(`$this->assert%s%s($a); //%s %s`, `$this->assert%s(%s, $a); //%s %s`)...)
	cases = append(cases, phpUnitConstructGeneratedCases(`$this->assert%s%s($a, "%s", "%s");`, `$this->assert%s(%s, $a, "%s", "%s");`)...)
	cases = append(cases, phpUnitConstructGeneratedCases(`static::assert%s%s($a); //%s %s`, `static::assert%s(%s, $a); //%s %s`)...)
	cases = append(cases, phpUnitConstructGeneratedCases(`STATIC::assert%s%s($a); //%s %s`, `STATIC::assert%s(%s, $a); //%s %s`)...)
	cases = append(cases, phpUnitConstructGeneratedCases(`self::assert%s%s($a); //%s %s`, `self::assert%s(%s, $a); //%s %s`)...)
	runUpstreamCases(t, PhpUnitConstruct{}, cases)
}
