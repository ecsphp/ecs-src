package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_data_provider_method_order).
func TestPhpUnitDataProviderMethodOrderPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitDataProviderMethodOrder{}, []aliasUpstreamCase{
		{
			"provider moved after the test (default placement)", nil,
			"<?php\nclass FooTest extends TestCase {\n    public function dataProvider() {}\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($expected, $actual) {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($expected, $actual) {}\n    public function dataProvider() {}\n}\n",
		},
		{
			"provider already after the test is left alone", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($expected, $actual) {}\n    public function dataProvider() {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($expected, $actual) {}\n    public function dataProvider() {}\n}\n",
		},
		{
			"placement before moves the provider ahead of the test", map[string]any{"placement": "before"},
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($expected, $actual) {}\n    public function dataProvider() {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    public function dataProvider() {}\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($expected, $actual) {}\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function dataProvider() {}\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($e, $a) {}\n}\n",
			"<?php\nclass Foo {\n    public function dataProvider() {}\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($e, $a) {}\n}\n",
		},
	})
}
