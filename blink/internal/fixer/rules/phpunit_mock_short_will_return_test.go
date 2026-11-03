package rules

import "testing"

// TestPhpUnitMockShortWillReturnNullsafe ports upstream's testFix80 and
// testFix81: a nullsafe receiver is shortened, a first-class callable is not.
func TestPhpUnitMockShortWillReturnNullsafe(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpUnitMockShortWillReturn{}, []upstreamCase{
		{
			name:     "testFix80",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock?->method(\"someMethod\")?->willReturn(10);\n    }\n}",
			input:    "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $someMock?->method(\"someMethod\")?->will($this?->returnValue(10));\n    }\n}",
		},
		{
			name:     "testFix81",
			expected: "<?php\nclass FooTest extends TestCase {\n    public function testFoo() {\n        $a = $someMock?->method(\"someMethod\")->will($this?->returnValue(...));\n    }\n}",
		},
	})
}
