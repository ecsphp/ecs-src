package rules

import (
	"slices"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// Cases ported from PHP-CS-Fixer's FunctionsAnalyzerTest::testIsGlobalFunctionCall: the
// byte offsets of the names PHP-CS-Fixer reports as global function calls.
func TestAliasIsGlobalFunctionCallPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code    string
		offsets []int
	}{
		{"<?php CONSTANT;", []int{}},
		{"<?php foo();", []int{6}},
		{"<?php foo(\"bar\");", []int{6}},
		{"<?php \\foo(\"bar\");", []int{7}},
		{"<?php foo\\bar(\"baz\");", []int{}},
		{"<?php foo::bar(\"baz\");", []int{}},
		{"<?php $foo->bar(\"baz\");", []int{}},
		{"<?php new bar(\"baz\");", []int{}},
		{"<?php function foo() {}", []int{}},
		{"<?php function & foo() {}", []int{}},
		{"<?php namespace\\foo(\"bar\");", []int{}},
		{"<?php\n                namespace A {\n                    use function A;\n                }\n                namespace B {\n                    use function D;\n                    A();\n                }\n            ", []int{176}},
		{"<?php\n                function A(){}\n                A();\n            ", []int{53}},
		{"<?php\n                function A(){}\n                a();\n            ", []int{53}},
		{"<?php\n                namespace {\n                    function A(){}\n                    A();\n                }\n            ", []int{89}},
		{"<?php\n                namespace Z {\n                    function A(){}\n                    A();\n                }\n            ", []int{}},
		{"<?php\n            namespace Z;\n\n            function A(){}\n            A();\n            ", []int{}},
		{"<?php\n                function & A(){}\n                A();\n            ", []int{55}},
		{"<?php\n                class Foo\n                {\n                    public function A(){}\n                }\n                A();\n            ", []int{126}},
		{"<?php\n                namespace A {\n                    function A(){}\n                }\n                namespace B {\n                    A();\n                }\n            ", []int{139}},
		{"<?php\n                use function X\\a;\n                A();\n            ", []int{}},
		{"<?php\n                use A;\n                A();\n            ", []int{45}},
		{"<?php\n                use const A;\n                A();\n            ", []int{51}},
		{"<?php\n                use function A;\n                str_repeat($a, $b);\n            ", []int{54}},
		{"<?php\n                namespace {\n                    function A(){}\n                    A();\n                    $b = function(){};\n                }\n            ", []int{89}},
		{"<?php implode($a);implode($a);implode($a);implode($a);implode($a);implode($a);", []int{6, 18, 30, 42, 54, 66}},
		{"<?php\n$z = new class(\n    new class(){ private function A(){} }\n){\n    public function A() {}\n};\n\nA();\n                ", []int{98}},
		{"<?php $foo = fn() => false;", []int{}},
		{"<?php foo(\"bar\"); class A { function Foo(){ foo(); } }", []int{6, 44}},
		{"<?php\n$array = [\n    1 => 2,\n    set() => 3,\n    $foo . set() => 3,\n];\n$x = set(set(), set());\nset();\nif (true) { set(); }\nset();", []int{33, 56, 76, 80, 87, 95, 114, 123}},
		{"<?php $a = new (foo());", []int{16}},
		{"<?php $b = $foo instanceof (foo());", []int{28}},
		{"<?php\n#[\\Attribute(\\Attribute::TARGET_CLASS)]\nclass Foo {}\n", []int{}},
		{"<?php $x?->count();", []int{}},
		{"<?php\n                #[Foo(), Bar(), Baz()]\n                class Foo {}\n            ", []int{}},
		{"<?php\nstrlen(...);\n\\strlen(...);\n$closure(...);\n$invokableObject(...);\n$obj->method(...);\n$obj->$methodStr(...);\n($obj->property)(...);\nFoo::method(...);\n$classStr::$methodStr(...);\nself::{$complex . $expression}(...);\n'strlen'(...);\n[$obj, 'method'](...);\n[Foo::class, 'method'](...);\n$c = new class{};\n$b = new class(){};\n$a = new #[foo]\nclass(){};\n", []int{}},
		{"<?php function g() { yield from foo(); }", []int{32}},
		{"<?php foo(\"bar\"); enum A { function Foo(){ foo(); } }", []int{6, 43}},
		{"<?php\nclass GetFirst\n{\n    public string $bothWithDoubleArrow = '' {\n        get => '';\n        set(string $x) => $x;\n    }\n    public string $getWithDoubleArrow = '' {\n        get => '';\n        set(string $x) { ''; }\n    }\n    public string $setWithDoubleArrow = '' {\n        get { ''; }\n        set(string $x) => $x;\n    }\n    public string $bothWithBraces = '' {\n        get { ''; }\n        set(string $x) { ''; }\n    }\n    public string $setUppercase = '' {\n        get { ''; }\n        SET(string $x) { ''; }\n    }\n}\nclass SetFirst\n{\n    public string $bothWithDoubleArrow = '' {\n        set(string $x) => $x;\n        get => '';\n    }\n    public string $getWithDoubleArrow = '' {\n        set(string $x) { ''; }\n        get => '';\n    }\n    public string $setWithDoubleArrow = '' {\n        set(string $x) => $x;\n        get { ''; }\n    }\n    public string $bothWithBraces = '' {\n        set(string $x) { ''; }\n        get { ''; }\n    }\n    public string $setUppercase = '' {\n        SET(string $x) { ''; }\n        get { ''; }\n    }\n}", []int{}},
	}
	for _, c := range cases {
		s := tokens.New(lexer.Lex(c.code))
		var got []int
		for i := range s.Len() {
			if aliasIsGlobalFunctionCall(s, i) {
				got = append(got, s.At(i).Pos)
			}
		}
		if !slices.Equal(got, c.offsets) && (len(got) != 0 || len(c.offsets) != 0) {
			t.Errorf("%q: global calls at %v, want %v", c.code, got, c.offsets)
		}
	}
}
