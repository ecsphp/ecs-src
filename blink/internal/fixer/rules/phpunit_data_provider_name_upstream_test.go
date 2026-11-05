package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_data_provider_name).
func TestPhpUnitDataProviderNamePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitDataProviderName{}, []aliasUpstreamCase{
		{
			"provider renamed to match the test (default prefix/suffix)", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($expected, $actual) {}\n    public function dataProvider() {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases() {}\n}\n",
		},
		{
			"attribute reference is renamed too", nil,
			"<?php\nclass FooTest extends TestCase {\n    #[DataProvider('dataProvider')]\n    public function testSomething($expected, $actual) {}\n    public function dataProvider() {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    #[DataProvider('provideSomethingCases')]\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases() {}\n}\n",
		},
		{
			"custom prefix ending in underscore keeps lower case", map[string]any{"prefix": "data_", "suffix": ""},
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider dtProvider\n     */\n    public function test_feature($expected, $actual) {}\n    public function dtProvider() {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider data_feature\n     */\n    public function test_feature($expected, $actual) {}\n    public function data_feature() {}\n}\n",
		},
		{
			"provider used by multiple tests is left alone", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider prov\n     */\n    public function testA($e, $a) {}\n    /**\n     * @dataProvider prov\n     */\n    public function testB($e, $a) {}\n    public function prov() {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider prov\n     */\n    public function testA($e, $a) {}\n    /**\n     * @dataProvider prov\n     */\n    public function testB($e, $a) {}\n    public function prov() {}\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($e, $a) {}\n    public function dataProvider() {}\n}\n",
			"<?php\nclass Foo {\n    /**\n     * @dataProvider dataProvider\n     */\n    public function testSomething($e, $a) {}\n    public function dataProvider() {}\n}\n",
		},
	})
}
