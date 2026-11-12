package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NewExpressionParenthesesFixerTest (PHP 8.4);
// every expected output is what PHP-CS-Fixer produces for the input.
func TestNewExpressionParenthesesPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	wp := map[string]any{"use_parentheses": true}
	aliasRunUpstreamCases(t, NewExpressionParentheses{}, []aliasUpstreamCase{
		// anonymous class
		{"anon no call", nil, "<?php (new class {});", "<?php (new class {});"},
		{"anon double wrap", nil, "<?php ((new class {}))->bar;", "<?php ((new class {}))->bar;"},
		{"anon unwrap", nil, "<?php (new class {})->bar;", "<?php new class {}->bar;"},
		{"anon unwrap comments", nil, "<?php ( /**/ new /**/ class /**/ {} /**/ )->bar;", "<?php  /**/ new /**/ class /**/ {} /**/ ->bar;"},
		{"anon unwrap args", nil, "<?php (new class ($x, (1 + 2) / 3) {})->bar;", "<?php new class ($x, (1 + 2) / 3) {}->bar;"},
		{"anon unwrap nested anon", nil, "<?php (new class (new class {}) {})->bar;", "<?php new class (new class {}) {}->bar;"},
		{"anon unwrap extends implements", nil, "<?php (new class ($x, (1 + 2) / 3) extends stdClass implements Throwable {})->bar;", "<?php new class ($x, (1 + 2) / 3) extends stdClass implements Throwable {}->bar;"},
		{"anon unwrap nested call", nil, "<?php (new class { public function __construct() { (new class {})->bar; } })->bar;", "<?php new class { public function __construct() { new class {}->bar; } }->bar;"},
		{"anon wrap already", wp, "<?php (new class {})->bar;", "<?php (new class {})->bar;"},
		{"anon wrap", wp, "<?php new class {}->bar;", "<?php (new class {})->bar;"},

		// regular class name
		{"named no call", nil, "<?php (new stdClass());", "<?php (new stdClass());"},
		{"named double wrap", nil, "<?php ((new stdClass()));", "<?php ((new stdClass()));"},
		{"named unwrap", nil, "<?php (new stdClass())->bar;", "<?php new stdClass()->bar;"},
		{"named unwrap leading ns", nil, "<?php (new \\stdClass())->bar;", "<?php new \\stdClass()->bar;"},
		{"named unwrap ns operator", nil, "<?php (new namespace\\Foo())->bar;", "<?php new namespace\\Foo()->bar;"},
		{"named unwrap ns path", nil, "<?php (new Name\\Space\\Foo())->bar;", "<?php new Name\\Space\\Foo()->bar;"},
		{"named unwrap leading ns path", nil, "<?php (new \\Name\\Space\\Foo())->bar;", "<?php new \\Name\\Space\\Foo()->bar;"},
		{"named unwrap comments", nil, "<?php ( /**/ new /**/ \\Name\\Space\\Foo /**/ () /**/ )->bar;", "<?php  /**/ new /**/ \\Name\\Space\\Foo /**/ () /**/ ->bar;"},
		{"named unwrap args", nil, "<?php (new Foo($x, (1 + 2) / 3))->bar;", "<?php new Foo($x, (1 + 2) / 3)->bar;"},
		{"named unwrap nested new", nil, "<?php (new Foo((new Baz())->m()))->bar;", "<?php new Foo(new Baz()->m())->bar;"},
		{"named no args no change", nil, "<?php (new Foo)->bar;", "<?php (new Foo)->bar;"},
		{"named wrap already", wp, "<?php (new Foo())->bar;", "<?php (new Foo())->bar;"},
		{"named wrap", wp, "<?php new Foo()->bar;", "<?php (new Foo())->bar;"},

		// $variable class name
		{"var unwrap", nil, "<?php (new $class())->bar;", "<?php new $class()->bar;"},
		{"var unwrap comments", nil, "<?php ( /**/ new /**/ $class /**/ ( /**/ ) /**/ )->bar;", "<?php  /**/ new /**/ $class /**/ ( /**/ ) /**/ ->bar;"},
		{"var var unwrap", nil, "<?php (new $$var())->bar;", "<?php new $$var()->bar;"},
		{"var brace unwrap", nil, "<?php (new ${\"var\"}())->bar;", "<?php new ${\"var\"}()->bar;"},
		{"var prop unwrap", nil, "<?php (new $obj->prop())->bar;", "<?php new $obj->prop()->bar;"},
		{"var nullsafe prop unwrap", nil, "<?php (new $obj?->prop())->bar;", "<?php new $obj?->prop()->bar;"},
		{"var static prop unwrap", nil, "<?php (new $obj::$prop())->bar;", "<?php new $obj::$prop()->bar;"},
		{"name static prop unwrap", nil, "<?php (new SomeClass::$prop())->bar;", "<?php new SomeClass::$prop()->bar;"},
		{"var index unwrap", nil, "<?php (new $obj[\"x\"]())->bar;", "<?php new $obj[\"x\"]()->bar;"},
		{"var index2 unwrap", nil, "<?php (new $obj[1][2]())->bar;", "<?php new $obj[1][2]()->bar;"},
		{"var no args no change", wp, "<?php (new $class)->bar;", "<?php (new $class)->bar;"},
		{"var wrap", wp, "<?php new $class()->bar;", "<?php (new $class())->bar;"},

		// (expression) class name
		{"expr no call", nil, "<?php (new (foo()));", "<?php (new (foo()));"},
		{"expr double wrap", nil, "<?php ((new (foo())));", "<?php ((new (foo())));"},
		{"expr unwrap", nil, "<?php (new (foo())())->bar;", "<?php new (foo())()->bar;"},
		{"expr unwrap comments", nil, "<?php ( /**/ new /**/ ( /**/ foo() /**/ ) /**/ ( /**/ ) /**/ )->bar;", "<?php  /**/ new /**/ ( /**/ foo() /**/ ) /**/ ( /**/ ) /**/ ->bar;"},
		{"expr no args no change", nil, "<?php (new (foo()))->bar;", "<?php (new (foo()))->bar;"},
		{"expr no args no change wp", wp, "<?php (new (foo()))->bar;", "<?php (new (foo()))->bar;"},
		{"expr wrap", wp, "<?php new (foo())()->bar;", "<?php (new (foo())())->bar;"},

		// call argument parentheses are not instantiation parentheses
		{"chain keeps call args", nil, "<?php (new Foo())\n    ->do(new Bar())\n    ->do(new Bar());", "<?php new Foo()\n    ->do(new Bar())\n    ->do(new Bar());"},
		{"function call arg", nil, "<?php foo(new Bar())->baz();", "<?php foo(new Bar())->baz();"},
		{"variable call arg", nil, "<?php $f(new Bar())->baz();", "<?php $f(new Bar())->baz();"},
		{"index call arg", nil, "<?php $a[0](new Bar())->baz();", "<?php $a[0](new Bar())->baz();"},
		{"method call arg static", nil, "<?php $o->m(new Bar())::baz();", "<?php $o->m(new Bar())::baz();"},
		{"function call arg wp", wp, "<?php foo(new Bar())->baz();", "<?php foo(new Bar())->baz();"},
	})
}
