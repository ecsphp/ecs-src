package rules

import "testing"

// Cases ported from PHP-CS-Fixer's FinalPublicMethodForAbstractClassFixerTest.
func TestFinalPublicMethodForAbstractClassPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, FinalPublicMethodForAbstractClass{}, []aliasUpstreamCase{
		{"regular-class", nil, "<?php class MyClass { public function f1(){} }", "<?php class MyClass { public function f1(){} }"},
		{"final-class", nil, "<?php final class MyClass { public function f1(){} }", "<?php final class MyClass { public function f1(){} }"},
		{"trait", nil, "<?php trait MyClass { public function f1(){} }", "<?php trait MyClass { public function f1(){} }"},
		{"interface", nil, "<?php interface MyClass { public function f1(); }", "<?php interface MyClass { public function f1(); }"},
		{"abstract public", nil,
			"<?php abstract class MyClass { public function f1(){} }",
			"<?php abstract class MyClass { final public function f1(){} }"},
		{"abstract public static", nil,
			"<?php abstract class MyClass { public static function f4(){} }",
			"<?php abstract class MyClass { final public static function f4(){} }"},
		{"abstract static public", nil,
			"<?php abstract class MyClass { static public function f7(){} }",
			"<?php abstract class MyClass { final static public function f7(){} }"},
		{"abstract protected/private untouched", nil,
			"<?php abstract class MyClass { protected function f2(){} private function f3(){} }",
			"<?php abstract class MyClass { protected function f2(){} private function f3(){} }"},
		{"magic methods", nil,
			"<?php abstract class MyClass { public function __construct() {} public function __toString() {} }",
			"<?php abstract class MyClass { public function __construct() {} public function __toString() {} }"},
		{"magic methods casing", nil,
			"<?php abstract class MyClass { public function __ToString() {} }",
			"<?php abstract class MyClass { public function __ToString() {} }"},
		{"non magic double underscore", nil,
			"<?php abstract class MyClass { public function __foo() {} public static function __bar($a, $b) {} }",
			"<?php abstract class MyClass { final public function __foo() {} final public static function __bar($a, $b) {} }"},
		{"already abstract/final methods", nil,
			"<?php abstract class MyClass { abstract public function foo(); final public function bar() {} }",
			"<?php abstract class MyClass { abstract public function foo(); final public function bar() {} }"},
		{"anonymous class", nil,
			"<?php abstract class MyClass { private function test() { $a = new class { public function f1(){} }; } }",
			"<?php abstract class MyClass { private function test() { $a = new class { public function f1(){} }; } }"},
		{"constant visibility", nil,
			"<?php abstract class MyClass { public const A = 1; protected const B = 2; private const C = 3; }",
			"<?php abstract class MyClass { public const A = 1; protected const B = 2; private const C = 3; }"},
		{"abstract readonly class", nil,
			"<?php abstract readonly class Foo {\n    public function bar() {}\n}",
			"<?php abstract readonly class Foo {\n    final public function bar() {}\n}"},
		{"readonly abstract class", nil,
			"<?php readonly abstract class Foo {\n    public function bar() {}\n}",
			"<?php readonly abstract class Foo {\n    final public function bar() {}\n}"},
	})
}
