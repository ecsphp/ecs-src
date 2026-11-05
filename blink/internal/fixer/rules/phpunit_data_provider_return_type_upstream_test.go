package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_data_provider_return_type).
func TestPhpUnitDataProviderReturnTypePhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitDataProviderReturnType{}, []aliasUpstreamCase{
		{
			"array return type becomes iterable", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases(): array {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases(): iterable {}\n}\n",
		},
		{
			"missing return type gets iterable", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases() {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases(): iterable {}\n}\n",
		},
		{
			"iterable return type is left alone", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases(): iterable {}\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases(): iterable {}\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function prov() {}\n}\n",
			"<?php\nclass Foo {\n    public function prov() {}\n}\n",
		},
	})
}
