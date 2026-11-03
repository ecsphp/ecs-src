package rules

import "testing"

// phpUnitTestAnnotationSample covers the method shapes DocBlockAnnotationTrait::
// getDocBlockIndex walks over (visibility, final, abstract, comments,
// attributes; not static) and the @depends/@testdox handling.
const phpUnitTestAnnotationSample = `<?php

namespace Demo;

use PHPUnit\Framework\TestCase;

abstract class SampleTest extends TestCase
{
    /**
     * @test
     */
    public static function staticOne() {}

    /**
     * @test
     */
    #[Group('x')]
    public function withAttribute() {}

    /** @test */
    abstract protected function abstractOne();

    // a comment
    public function testAfterComment() {}

    /**
     * @depends first
     * @test
     */
    final public function second() {}

    /**
     * @testdox does things
     */
    public function documented() {}

    public function helper()
    {
        return function () {};
    }

    /** @test */
    public function snake_case_name() {}

    /**
     * @dataProvider provide
     * @test
     */
    public function provided($a) {}

    public function test_x_y() {}

    public function testFirst() {}
}
`

// TestPhpUnitTestAnnotationMethodShapes pins both styles on the sample to the
// output of the PHP engine.
func TestPhpUnitTestAnnotationMethodShapes(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpUnitTestAnnotation{}, []upstreamCase{
		{name: "prefix", expected: `<?php

namespace Demo;

use PHPUnit\Framework\TestCase;

abstract class SampleTest extends TestCase
{
    /**
     * @test
     */
    public static function staticOne() {}

    /**
     *
     */
    #[Group('x')]
    public function testWithAttribute() {}

    /** */
    abstract protected function testAbstractOne();

    // a comment
    public function testAfterComment() {}

    /**
     * @depends testFirst
     *
     */
    final public function testSecond() {}

    /**
     * @testdox does things
     */
    public function testDocumented() {}

    public function helper()
    {
        return function () {};
    }

    /** */
    public function testSnake_case_name() {}

    /**
     * @dataProvider provide
     *
     */
    public function testProvided($a) {}

    public function test_x_y() {}

    public function testFirst() {}
}
`, input: phpUnitTestAnnotationSample},
		{name: "annotation", cfg: map[string]any{"style": "annotation"}, expected: `<?php

namespace Demo;

use PHPUnit\Framework\TestCase;

abstract class SampleTest extends TestCase
{
    /**
     * @test
     */
    public static function staticOne() {}

    /**
     * @test
     */
    #[Group('x')]
    public function withAttribute() {}

    /** @test */
    abstract protected function abstractOne();

    // a comment
    /**
     * @test
     */
    public function afterComment() {}

    /**
     * @depends first
     * @test
     */
    final public function second() {}

    /**
     * @testdox does things
     *
     * @test
     */
    public function documented() {}

    public function helper()
    {
        return function () {};
    }

    /** @test */
    public function snake_case_name() {}

    /**
     * @dataProvider provide
     * @test
     */
    public function provided($a) {}

    /**
     * @test
     */
    public function x_y() {}

    /**
     * @test
     */
    public function first() {}
}
`, input: phpUnitTestAnnotationSample},
	})
}
