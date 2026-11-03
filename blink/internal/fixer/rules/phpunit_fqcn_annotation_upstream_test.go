package rules

import "testing"

// TestPhpUnitFqcnAnnotationUpstream runs PHP-CS-Fixer's own PhpUnitFqcnAnnotationFixerTest cases.
func TestPhpUnitFqcnAnnotationUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpUnitFqcnAnnotation{}, []upstreamCase{
		{
			name:     "provideFixCases: case 1",
			expected: "<?php\n/**\n * @covers \\Foo\n * @covers ::fooMethod\n * @coversDefaultClass \\Bar\n */\nclass FooTest extends TestCase {\n    /**\n     * @ExpectedException Value\n     * @expectedException \\X\n     * @expectedException\n     * @expectedException \\Exception\n         * @expectedException \\Some\\Exception\\ClassName\n * @expectedExceptionCode 123\n     * @expectedExceptionMessage Foo bar\n     *\n     * @uses \\Baz\n     * @uses \\selfieGenerator\n     * @uses self::someFunction\n     * @uses static::someOtherFunction\n     */\n}",
			input:    "<?php\n/**\n * @covers Foo\n * @covers ::fooMethod\n * @coversDefaultClass Bar\n */\nclass FooTest extends TestCase {\n    /**\n     * @ExpectedException Value\n     * @expectedException X\n     * @expectedException\n     * @expectedException \\Exception\n         * @expectedException Some\\Exception\\ClassName\n * @expectedExceptionCode 123\n     * @expectedExceptionMessage Foo bar\n     *\n     * @uses Baz\n     * @uses selfieGenerator\n     * @uses self::someFunction\n     * @uses static::someOtherFunction\n     */\n}",
		},
		{
			name:     "provideFixCases: case 2",
			expected: "<?php\nclass Foo {\n    /**\n     * @expectedException Some\\Exception\\ClassName\n     * @covers Foo\n     * @uses Baz\n     * @uses self::someFunction\n     */\n}\n",
		},
	})
}
