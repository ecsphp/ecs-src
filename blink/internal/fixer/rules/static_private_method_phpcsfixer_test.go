package rules

import "testing"

// Cases ported from PHP-CS-Fixer's StaticPrivateMethodFixerTest.
func TestStaticPrivateMethodPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, StaticPrivateMethod{}, []aliasUpstreamCase{
		{"call from other method", nil,
			"<?php\nclass Foo\n{\n    public $baz;\n\n    public function bar()\n    {\n        $var = $this->baz;\n        $var = $this->baz();\n        if (true) {\n            $var = $this->baz();\n        }\n    }\n\n    private function baz()\n    {\n        if (true) {\n            return 1;\n        }\n    }\n}",
			"<?php\nclass Foo\n{\n    public $baz;\n\n    public function bar()\n    {\n        $var = $this->baz;\n        $var = self::baz();\n        if (true) {\n            $var = self::baz();\n        }\n    }\n\n    private static function baz()\n    {\n        if (true) {\n            return 1;\n        }\n    }\n}"},
		{"multiple classes", nil,
			"<?php\nabstract class Foo\n{\n    private function baz() { return 1; }\n\n    public function xyz()\n    {\n        return 1;\n    }\n}\nabstract class Bar\n{\n    public function baz() { return 1; }\n\n    abstract protected function xyz1();\n    protected abstract function xyz2();\n    abstract function xyz3();\n}",
			"<?php\nabstract class Foo\n{\n    private static function baz() { return 1; }\n\n    public function xyz()\n    {\n        return 1;\n    }\n}\nabstract class Bar\n{\n    public function baz() { return 1; }\n\n    abstract protected function xyz1();\n    protected abstract function xyz2();\n    abstract function xyz3();\n}"},
		{"already static", nil,
			"<?php\nclass Foo\n{\n    private static function foo1() { return 1; }\n    static private function foo2() { return 1; }\n}",
			"<?php\nclass Foo\n{\n    private static function foo1() { return 1; }\n    static private function foo2() { return 1; }\n}"},
		{"methods containing closures", nil,
			"<?php\nclass Foo\n{\n    private function bar()\n    {\n        return function() {};\n    }\n\n    private function baz()\n    {\n        return static function() {};\n    }\n}",
			"<?php\nclass Foo\n{\n    private function bar()\n    {\n        return function() {};\n    }\n\n    private function baz()\n    {\n        return static function() {};\n    }\n}"},
		{"instance reference", nil,
			"<?php\nclass Foo\n{\n    private function bar()\n    {\n        if (true) {\n            return $this;\n        }\n    }\n}",
			"<?php\nclass Foo\n{\n    private function bar()\n    {\n        if (true) {\n            return $this;\n        }\n    }\n}"},
		{"debug_backtrace", nil,
			"<?php\nclass Foo\n{\n    private function bar()\n    {\n        return debug_backtrace()[1]['object'];\n    }\n}",
			"<?php\nclass Foo\n{\n    private function bar()\n    {\n        return debug_backtrace()[1]['object'];\n    }\n}"},
		{"magic methods", nil,
			"<?php\nclass Foo\n{\n    private function __clone() {}\n    private function __construct() {}\n    private function __destruct() {}\n    private function __wakeup() {}\n}",
			"<?php\nclass Foo\n{\n    private function __clone() {}\n    private function __construct() {}\n    private function __destruct() {}\n    private function __wakeup() {}\n}"},
		{"method calling itself", nil,
			"<?php\nclass Foo\n{\n    private function baz()\n    {\n        return $this->baz();\n    }\n}",
			"<?php\nclass Foo\n{\n    private static function baz()\n    {\n        return self::baz();\n    }\n}"},
		{"trait", nil,
			"<?php\nclass Foo\n{\n    use A, B, C {\n        asd as lol;\n    }\n\n    private function bar() {}\n}",
			"<?php\nclass Foo\n{\n    use A, B, C {\n        asd as lol;\n    }\n\n    private static function bar() {}\n}"},
		{"final", nil,
			"<?php\nclass Foo\n{\n    final private function baz1() { return 1; }\n    private final function baz2() { return 1; }\n}",
			"<?php\nclass Foo\n{\n    final private static function baz1() { return 1; }\n    private final static function baz2() { return 1; }\n}"},
		{"nested calls", nil,
			"<?php class Foo {\n    public function the_function() { return $this->function_0(); }\n    private function function_2() { return $this->function_3(); }\n    private function function_1() { return $this->function_2(); }\n    private function function_3() { return $this->function_4(); }\n    private function function_7() { return $this->function_8(); }\n    private function function_9() { return null; }\n    private function function_5() { return $this->function_6(); }\n    private function function_8() { return $this->function_9(); }\n    private function function_0() { return $this->function_1(); }\n    private function function_4() { return $this->function_5(); }\n    private function function_6() { return $this->function_7(); }\n}",
			"<?php class Foo {\n    public function the_function() { return self::function_0(); }\n    private static function function_2() { return self::function_3(); }\n    private static function function_1() { return self::function_2(); }\n    private static function function_3() { return self::function_4(); }\n    private static function function_7() { return self::function_8(); }\n    private static function function_9() { return null; }\n    private static function function_5() { return self::function_6(); }\n    private static function function_8() { return self::function_9(); }\n    private static function function_0() { return self::function_1(); }\n    private static function function_4() { return self::function_5(); }\n    private static function function_6() { return self::function_7(); }\n}"},
		{"private constructor and abstract method", nil,
			"<?php\nabstract class AbstractClass\n{\n    private function __construct() {}\n    abstract protected static function abstractFunction();\n}\n",
			"<?php\nabstract class AbstractClass\n{\n    private function __construct() {}\n    abstract protected static function abstractFunction();\n}\n"},
	})
}
