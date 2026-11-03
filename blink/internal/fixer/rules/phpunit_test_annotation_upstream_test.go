package rules

import "testing"

// TestPhpUnitTestAnnotationUpstream runs PHP-CS-Fixer's own PhpUnitTestAnnotationFixerTest cases.
func TestPhpUnitTestAnnotationUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpUnitTestAnnotation{}, []upstreamCase{
		{
			name:     "provideFixCases: Annotation is used, and it should not be",
			cfg:      map[string]any{"style": "prefix"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     *\n     */\n    public function testItDoesSomething() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function itDoesSomething() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation is not used, but should be",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function itDoesSomething() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function testItDoesSomething() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation is not used, but should be, class is extra indented",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nif (1) {\n    class Test extends \\PhpUnit\\FrameWork\\TestCase\n    {\n        /**\n         * @test\n         */\n        public function itDoesSomething() {}\n    }\n}",
			input:    "<?php\nif (1) {\n    class Test extends \\PhpUnit\\FrameWork\\TestCase\n    {\n        public function testItDoesSomething() {}\n    }\n}",
		},
		{
			name:     "provideFixCases: Annotation is not used, but should be, and there is already a docBlock",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @dataProvider blabla\n     *\n     * @test\n     */\n    public function itDoesSomething() {}\n    }",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @dataProvider blabla\n     */\n    public function testItDoesSomething() {}\n    }",
		},
		{
			name:     "provideFixCases: Annotation is used, but should not be, and it depends on other tests",
			cfg:      map[string]any{"style": "prefix"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     *\n     */\n    public function testAaa () {}\n\n    public function helperFunction() {}\n\n    /**\n     * @depends testAaa\n     *\n     *\n     */\n    public function testBbb () {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function aaa () {}\n\n    public function helperFunction() {}\n\n    /**\n     * @depends aaa\n     *\n     * @test\n     */\n    public function bbb () {}\n}",
		},
		{
			name:     "provideFixCases: Annotation is not used, but should be, and it depends on other tests",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function aaa () {}\n\n    /**\n     * @depends aaa\n     *\n     * @test\n     */\n    public function bbb () {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function testAaa () {}\n\n    /**\n     * @depends testAaa\n     */\n    public function testBbb () {}\n}",
		},
		{
			name:     "provideFixCases: Annotation is removed, the function is one word and we want it to use camel case",
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     *\n     */\n    public function testWorks() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function works() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation is added, and it is snake case",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function it_has_snake_case() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function test_it_has_snake_case() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation gets added, it has an @depends, and we use snake case",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function works_fine () {}\n\n    /**\n     * @depends works_fine\n     *\n     * @test\n     */\n    public function works_fine_too() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function test_works_fine () {}\n\n    /**\n     * @depends test_works_fine\n     */\n    public function test_works_fine_too() {}\n}",
		},
		{
			name:     "provideFixCases: Class has both camel and snake case, annotated functions and not, and wants to add annotations",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function snake_cased () {}\n\n    /**\n     * @test\n     */\n    public function camelCased () {}\n\n    /**\n     * Description.\n     *\n     * @depends camelCased\n     *\n     * @test\n     */\n    public function depends_on_someone () {}\n\n    //It even has a comment\n    public function a_helper_function () {}\n\n    /**\n     * @depends depends_on_someone\n     *\n     * @test\n     */\n    public function moreDepends() {}\n\n    /**\n     * @depends depends_on_someone\n     *\n     * @test\n     */\n    public function alreadyAnnotated() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function test_snake_cased () {}\n\n    public function testCamelCased () {}\n\n    /**\n     * Description.\n     *\n     * @depends testCamelCased\n     */\n    public function test_depends_on_someone () {}\n\n    //It even has a comment\n    public function a_helper_function () {}\n\n    /**\n     * @depends test_depends_on_someone\n     */\n    public function testMoreDepends() {}\n\n    /**\n     * @depends test_depends_on_someone\n     *\n     * @test\n     */\n    public function alreadyAnnotated() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation has to be added to multiple functions",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function itWorks() {}\n\n    /**\n     * @test\n     */\n    public function itDoesSomething() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function testItWorks() {}\n\n    public function testItDoesSomething() {}\n}",
		},
		{
			name:     "provideFixCases: Class with big doc blocks and multiple functions has to remove annotations",
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n\n    /**\n     * This test is part of the database group and has a provider.\n     *\n     * @param int $paramOne\n     * @param bool $paramTwo\n     *\n     *\n     * @dataProvider provides\n     * @group Database\n     */\n    public function testDatabase ($paramOne, $paramTwo) {}\n\n    /**\n     * Provider for the database test function\n     *\n     * @return array\n     */\n    public function provides() {}\n\n    /**\n     * I am just a helper function but I have test in my name.\n     * I also have a doc Block\n     *\n     * @return Foo\\Bar\n     */\n    public function help_test() {}\n\n\n    protected function setUp() {}\n\n    /**\n     * I depend on the database function, but I already\n     * had test in my name and a docblock\n     *\n     * @depends testDatabase\n     */\n    public function testDepends () {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n\n    /**\n     * This test is part of the database group and has a provider.\n     *\n     * @param int $paramOne\n     * @param bool $paramTwo\n     *\n     * @test\n     * @dataProvider provides\n     * @group Database\n     */\n    public function database ($paramOne, $paramTwo) {}\n\n    /**\n     * Provider for the database test function\n     *\n     * @return array\n     */\n    public function provides() {}\n\n    /**\n     * I am just a helper function but I have test in my name.\n     * I also have a doc Block\n     *\n     * @return Foo\\Bar\n     */\n    public function help_test() {}\n\n\n    protected function setUp() {}\n\n    /**\n     * I depend on the database function, but I already\n     * had test in my name and a docblock\n     *\n     * @depends database\n     */\n    public function testDepends () {}\n}",
		},
		{
			name:     "provideFixCases: Test Annotation has to be removed, but its just one line",
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /** */\n    public function testItWorks() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /** @test */\n    public function itWorks() {}\n}",
		},
		{
			name:     "provideFixCases: Test annotation has to be added, but there is already a one line doc block",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @group Database\n     *\n     * @test\n     */\n    public function itTestsDatabase() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /** @group Database */\n    public function testItTestsDatabase() {}\n}",
		},
		{
			name:     "provideFixCases: Test annotation has to be added, but there is already a one line doc block which is a sentence",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * I really like this test, it helps a lot\n     *\n     * @test\n     */\n    public function itTestsDatabase() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /** I really like this test, it helps a lot */\n    public function testItTestsDatabase() {}\n}",
		},
		{
			name:     "provideFixCases: Test annotation has to be added, but there is already a one line comment present",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    //I really like this test, it helps a lot\n    /**\n     * @test\n     */\n    public function itTestsDatabase() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    //I really like this test, it helps a lot\n    public function testItTestsDatabase() {}\n}",
		},
		{
			name:     "provideFixCases: Test annotation has to be added, there is a one line doc block which is an @depends tag",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function itTestsDatabase() {}\n\n    /**\n     * @depends itTestsDatabase\n     *\n     * @test\n     */\n    public function itDepends() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function testItTestsDatabase() {}\n\n    /** @depends testItTestsDatabase */\n    public function testItDepends() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation gets removed, but the function has a @testWith",
			expected: "<?php\nfinal class ProcessLinterProcessBuilderTest extends TestCase\n{\n    /**\n     *\n     * @param string $executable\n     * @param string $file\n     * @param string $expected\n     *\n     * @testWith [\"php\", \"foo.php\", \"\\\"php\\\" -l \\\"foo.php\\\"\"]\n     *           [\"C:\\Program Files\\php\\php.exe\", \"foo bar\\baz.php\", \"\\\"C:\\Program Files\\php\\php.exe\\\" -l \\\"foo bar\\baz.php\\\"\"]\n     * @requires OS Linux|Darwin\n     */\n    public function testPrepareCommandOnPhpOnLinuxOrMac($executable, $file, $expected)\n    {\n        $builder = new ProcessLinterProcessBuilder($executable);\n\n        $this->assertSame(\n            $expected,\n            $builder->build($file)->getCommandLine()\n        );\n    }\n}",
			input:    "<?php\nfinal class ProcessLinterProcessBuilderTest extends TestCase\n{\n    /**\n     * @test\n     * @param string $executable\n     * @param string $file\n     * @param string $expected\n     *\n     * @testWith [\"php\", \"foo.php\", \"\\\"php\\\" -l \\\"foo.php\\\"\"]\n     *           [\"C:\\Program Files\\php\\php.exe\", \"foo bar\\baz.php\", \"\\\"C:\\Program Files\\php\\php.exe\\\" -l \\\"foo bar\\baz.php\\\"\"]\n     * @requires OS Linux|Darwin\n     */\n    public function prepareCommandOnPhpOnLinuxOrMac($executable, $file, $expected)\n    {\n        $builder = new ProcessLinterProcessBuilder($executable);\n\n        $this->assertSame(\n            $expected,\n            $builder->build($file)->getCommandLine()\n        );\n    }\n}",
		},
		{
			name:     "provideFixCases: Annotation gets added, but there is already an @testWith in the doc block",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nfinal class ProcessLinterProcessBuilderTest extends TestCase\n{\n    /**\n     * @param string $executable\n     * @param string $file\n     * @param string $expected\n     *\n     * @testWith [\"php\", \"foo.php\", \"\\\"php\\\" -l \\\"foo.php\\\"\"]\n     *           [\"C:\\Program Files\\php\\php.exe\", \"foo bar\\baz.php\", \"\\\"C:\\Program Files\\php\\php.exe\\\" -l \\\"foo bar\\baz.php\\\"\"]\n     * @requires OS Linux|Darwin\n     *\n     * @test\n     */\n    public function prepareCommandOnPhpOnLinuxOrMac($executable, $file, $expected)\n    {\n        $builder = new ProcessLinterProcessBuilder($executable);\n\n        $this->assertSame(\n            $expected,\n            $builder->build($file)->getCommandLine()\n        );\n    }\n}",
			input:    "<?php\nfinal class ProcessLinterProcessBuilderTest extends TestCase\n{\n    /**\n     * @param string $executable\n     * @param string $file\n     * @param string $expected\n     *\n     * @testWith [\"php\", \"foo.php\", \"\\\"php\\\" -l \\\"foo.php\\\"\"]\n     *           [\"C:\\Program Files\\php\\php.exe\", \"foo bar\\baz.php\", \"\\\"C:\\Program Files\\php\\php.exe\\\" -l \\\"foo bar\\baz.php\\\"\"]\n     * @requires OS Linux|Darwin\n     */\n    public function testPrepareCommandOnPhpOnLinuxOrMac($executable, $file, $expected)\n    {\n        $builder = new ProcessLinterProcessBuilder($executable);\n\n        $this->assertSame(\n            $expected,\n            $builder->build($file)->getCommandLine()\n        );\n    }\n}",
		},
		{
			name:     "provideFixCases: Annotation gets properly removed, even when it is in a weird place",
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * I am a comment about the function\n     */\n    public function testIHateMyTestSuite() {}\n\n    /**\n     * I am another comment about a function\n     */\n    public function testThisMakesNoSense() {}\n\n    /**\n     * This comment has   more issues\n     */\n    public function testItUsesTabs() {}\n\n    /**\n     * @depends testItUsesTabs\n     */\n    public function testItDependsReally() {}\n\n    /**\n     * @depends testItUsesTabs\n     */\n    public function testItDependsSomeMore() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * I am a comment @test about the function\n     */\n    public function iHateMyTestSuite() {}\n\n    /**\n     * I am another comment about a function @test\n     */\n    public function thisMakesNoSense() {}\n\n    /**\n     * This comment has @test   more issues\n     */\n    public function itUsesTabs() {}\n\n    /**\n     * @depends itUsesTabs @test\n     */\n    public function itDependsReally() {}\n\n    /**\n     * @test @depends itUsesTabs\n     */\n    public function itDependsSomeMore() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation gets added when a single line has doc block has multiple tags already",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * There is some text here @group Database @group Integration\n     *\n     * @test\n     */\n    public function whyDoThis() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /** There is some text here @group Database @group Integration */\n    public function testWhyDoThis() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation gets removed when a single line doc block has the tag, but there are other things as well",
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /** There is some text here @group Database @group Integration */\n    public function testWhyDoThis() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /** There is some @test text here @group Database @group Integration */\n    public function testWhyDoThis() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation is used, and should be",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function itDoesSomething() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation is not used, and should not be",
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function testItDoesSomethingWithoutPhpDoc() {}\n    /**\n     * No annotation, just text\n     */\n    public function testItDoesSomethingWithPhpDoc() {}\n\n    public function testingItDoesSomethingWithoutPhpDoc() {}\n    /**\n     * No annotation, just text\n     */\n    public function testingItDoesSomethingWithPhpDoc() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation is added when it is already present in a weird place",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * I am a comment @test about the function\n     *\n     * @test\n     */\n    public function iHateMyTestSuite() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * I am a comment @test about the function\n     */\n    public function iHateMyTestSuite() {}\n}",
		},
		{
			name:     "provideFixCases: Docblock does not get converted to a multi line doc block if it already has @test annotation",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /** @test */\n    public function doesSomeThings() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation does not get added if class is not a test",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Waterloo\n{\n    public function testDoesSomeThings() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation does not get removed if class is not a test",
			expected: "<?php\nclass Waterloo\n{\n    /**\n     * @test\n     */\n    public function doesSomeThings() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation does not get added if there are no tests in the test class",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function setUp() {}\n\n    public function itHelpsSomeTests() {}\n\n    public function someMoreChanges() {}\n}",
		},
		{
			name:     "provideFixCases: Abstract test gets annotation removed",
			cfg:      map[string]any{"style": "prefix"},
			expected: "<?php\nabstract class Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     *\n     */\n    abstract function testFooBar();\n}",
			input:    "<?php\nabstract class Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    abstract function fooBar();\n}",
		},
		{
			name:     "provideFixCases: Annotation present, but method already have test prefix",
			cfg:      map[string]any{"style": "prefix"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     *\n     */\n    public function testarossaIsFromItaly() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function testarossaIsFromItaly() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation present, but method is test prefix",
			cfg:      map[string]any{"style": "prefix"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     *\n     */\n    public function test() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function test() {}\n}",
		},
		{
			name:     "provideFixCases: Abstract test gets annotation added",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nabstract class Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    abstract function fooBar();\n}",
			input:    "<?php\nabstract class Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    abstract function testFooBar();\n}",
		},
		{
			name:     "provideFixCases: Annotation gets added, but there is a number after the testprefix so it keeps the prefix",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function test123fooBar() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function test123fooBar() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation missing, but there is a lowercase character after the test prefix so it keeps the prefix",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function testarossaIsFromItaly() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function testarossaIsFromItaly() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation present, but there is a lowercase character after the test prefix so it keeps the prefix",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function testarossaIsFromItaly() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation missing, method qualifies as test, but test prefix cannot be removed",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function test() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function test() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation missing, method qualifies as test, but test_ prefix cannot be removed",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function test_() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function test_() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation present, method qualifies as test, but test_ prefix cannot be removed",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function test_() {}\n}",
		},
		{
			name:     "provideFixCases: Annotation missing, method after fix still has \"test\" prefix",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    public function test_foo() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    public function test_test_foo() {}\n}",
		},
		{
			name:     "provideFixCases: do not touch single line @depends annotation when already correct",
			expected: "<?php class FooTest extends \\PHPUnit\\Framework\\TestCase\n                {\n                    public function testOne() {}\n\n                    /** @depends testOne */\n                    public function testTwo() {}\n\n                    /**    @depends    testTwo    */\n                    public function testThree() {}\n                }",
		},
		{
			name:     "provideWithWhitespacesConfigCases: case 42",
			expected: "<?php\n\n                    class FooTest extends \\PHPUnit_Framework_TestCase {\n\n                    /**\n                     *\n                     */\n                    public function testFooTest() {}\n                    }\n                ",
			input:    "<?php\n\n                    class FooTest extends \\PHPUnit_Framework_TestCase {\n\n                    /**\n                     * @test\n                     */\n                    public function fooTest() {}\n                    }\n                ",
		},
		{
			name:     "provideFix80Cases: case 43",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    #[OneTest]\n    public function itWorks() {}\n\n    /**\n     * @test\n     */\n    #[TwoTest]\n    public function itDoesSomething() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    #[OneTest]\n    public function testItWorks() {}\n\n    #[TwoTest]\n    public function testItDoesSomething() {}\n}",
		},
		{
			name:     "provideFix80Cases: case 44",
			cfg:      map[string]any{"style": "annotation"},
			expected: "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    /**\n     * @test\n     */\n    #[OneTest]\n    #[Internal]\n    public function itWorks() {}\n\n    /**\n     * @test\n     */\n    #[TwoTest]\n    #[Internal]\n    public function itDoesSomething() {}\n}",
			input:    "<?php\nclass Test extends \\PhpUnit\\FrameWork\\TestCase\n{\n    #[OneTest]\n    #[Internal]\n    public function testItWorks() {}\n\n    #[TwoTest]\n    #[Internal]\n    public function testItDoesSomething() {}\n}",
		},
	})
}
