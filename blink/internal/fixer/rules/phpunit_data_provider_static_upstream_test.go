package rules

import "testing"

// Cases checked against PHP-CS-Fixer 3.95 (php_unit_data_provider_static).
func TestPhpUnitDataProviderStaticPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, PhpUnitDataProviderStatic{}, []aliasUpstreamCase{
		{
			"provider without $this becomes static", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases() { return []; }\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public static function provideSomethingCases() { return []; }\n}\n",
		},
		{
			"provider using $this is left alone by default", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases() { return $this->data(); }\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases() { return $this->data(); }\n}\n",
		},
		{
			"force makes a $this provider static", map[string]any{"force": true},
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public function provideSomethingCases() { self::data(); }\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public static function provideSomethingCases() { self::data(); }\n}\n",
		},
		{
			"already static provider is left alone", nil,
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public static function provideSomethingCases() { return []; }\n}\n",
			"<?php\nclass FooTest extends TestCase {\n    /**\n     * @dataProvider provideSomethingCases\n     */\n    public function testSomething($expected, $actual) {}\n    public static function provideSomethingCases() { return []; }\n}\n",
		},
		{
			"not a PHPUnit class is untouched", nil,
			"<?php\nclass Foo {\n    public function prov() { return []; }\n}\n",
			"<?php\nclass Foo {\n    public function prov() { return []; }\n}\n",
		},
	})
}
