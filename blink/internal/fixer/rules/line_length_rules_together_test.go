package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// The expected output comes from the PHP engine, with the three rules together.
func TestMethodArgumentSpaceSpacesInsideAndLineLengthTogether(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"#[A(1,)]\nclass X {}", "#[A(1,)]\nclass X {}"},
		{"#[A(1, )]\nclass X {}", "#[A(1,)]\nclass X {}"},
		{"#[A(x: 1,)]\nclass X {}", "#[A(x: 1,)]\nclass X {}"},
		{"#[A(x: 1, )]\nclass X {}", "#[A(x: 1,)]\nclass X {}"},
		{"#[A(\n    1,\n)]\nclass X {}", "#[A(1,)]\nclass X {}"},
		{"#[A(\n    1, )]\nclass X {}", "#[A(1,)]\nclass X {}"},
		{"class X { #[A(1,)] public $a; }", "class X { #[A(1,)] public $a; }"},
		{"class X { #[A\\B(1,)] public $a; }", "class X { #[A\\B(1,)] public $a; }"},
		{"function f(#[A(1,)] $x) {}", "function f(#[A(1,)] $x) {}"},
		{"foo(1,);", "foo(1,);"},
		{"foo(1, );", "foo(1,);"},
		{"foo(\n    1, );", "foo(1,);"},
		{"foo(\n    1,\n);", "foo(1,);"},
		{"new A(1,);", "new A(1,);"},
		{"new A(1, );", "new A(1,);"},
		{"$a->b(1,);", "$a->b(1,);"},
		{"$a->b(1, );", "$a->b(1,);"},
		{"A::b(1,);", "A::b(1,);"},
		{"A::b(1, );", "A::b(1,);"},
		{"\\foo(1,);", "\\foo(1,);"},
		{"\\foo(1, );", "\\foo(1,);"},
		{"Foo(1,);", "Foo(1, );"},
		{"Foo(1, );", "Foo(1, );"},
		{"function g($a,) {}", "function g($a,) {}"},
		{"function g($a, ) {}", "function g($a,) {}"},
		{"$f = function ($a,) {};", "$f = function ($a,) {};"},
		{"$f = function ($a, ) {};", "$f = function ($a,) {};"},
		{"$f = fn ($a,) => 1;", "$f = fn ($a, ) => 1;"},
		{"$f = fn ($a, ) => 1;", "$f = fn ($a, ) => 1;"},
		{"$f = function () use ($a,) {};", "$f = function () use ($a,) {};"},
		{"$f = function () use ($a, ) {};", "$f = function () use ($a,) {};"},
		{"foo(new A([\n    1,\n], ), 2);", "foo(new A([1,],), 2);"},
		{"foo([\n    1,\n],);", "foo([1,],);"},
		{"foo([\n    1,\n], );", "foo([1,],);"},
		{"array(1, );", "array(1, );"},
		{"list($a, ) = $b;", "list($a, ) = $b;"},
		{"isset($a, );", "isset($a, );"},
		{"if (foo(1, )) {\n}", "if (foo(1,)) {\n}"},
		{"#[A( 1 )]\nclass X {}", "#[A(1)]\nclass X {}"},
		{"#[A(1, 2)]\nclass X {}", "#[A(1, 2)]\nclass X {}"},
		{"#[A(\n    1,\n    2\n)]\nclass X {}", "#[A(1, 2)]\nclass X {}"},
		{"#[A(\n    1,\n    2,\n)]\nclass X {}", "#[A(1, 2,)]\nclass X {}"},
		{"#[A(\n    x: 1,\n    y: 2,\n)]\nclass X {}", "#[A(x: 1, y: 2,)]\nclass X {}"},
		{"#[A(1, ), B(2, )]\nclass X {}", "#[A(1,), B(2, )]\nclass X {}"},
		{"#[A(\n    1\n), B(\n    2\n)]\nclass X {}", "#[A(1), B(\n    2\n)]\nclass X {}"},
		{"#[\n    A(1),\n    B(2),\n]\nclass X {}", "#[\n    A(1),\n    B(2),\n]\nclass X {}"},
		{"#[\n    A(\n        1\n    ),\n]\nclass X {}", "#[\n    A(1),\n]\nclass X {}"},
		{"#[\n    A,\n    B,\n]\nclass X {}", "#[\n    A,\n    B,\n]\nclass X {}"},
		{"#[\n    A\n]\nclass X {}", "#[\n    A\n]\nclass X {}"},
		{"#[A]\nclass X {}", "#[A]\nclass X {}"},
		{"#[A()]\nclass X {}", "#[A()]\nclass X {}"},
		{"#[A( )]\nclass X {}", "#[A()]\nclass X {}"},
		{"#[A(\n)]\nclass X {}", "#[A()]\nclass X {}"},
		{"#[A(\n    x: [1, 2],\n    y: 3\n)]\nclass X {}", "#[A(x: [1, 2], y: 3)]\nclass X {}"},
		{"#[A(\n    x: [\n        1,\n        2,\n    ]\n)]\nclass X {}", "#[A(x: [1, 2,])]\nclass X {}"},
		{"#[A(\n    new B(\n        1\n    )\n)]\nclass X {}", "#[A(new B(1))]\nclass X {}"},
		{"#[A(\n    ['k' => 1]\n)]\nclass X {}", "#[A(\n    ['k' => 1]\n)]\nclass X {}"},
		{"#[A(\n    k: ['k' => 1],\n    j: 2\n)]\nclass X {}", "#[A(\n    k: ['k' => 1],\n    j: 2\n)]\nclass X {}"},
		{"#[A(\n    // c\n    1\n)]\nclass X {}", "#[A(\n    // c\n    1\n)]\nclass X {}"},
		{"#[A(\n    /* c */\n    1\n)]\nclass X {}", "#[A(\n    /* c */\n    1\n)]\nclass X {}"},
		{"#[A(\n    1 // c\n)]\nclass X {}", "#[A(\n    1 // c\n)]\nclass X {}"},
		{"#[A(\n    <<<'T'\n    a\n    T\n)]\nclass X {}", "#[A(\n    <<<'T'\n    a\n    T\n)]\nclass X {}"},
		{"#[A(\n    'a\nb'\n)]\nclass X {}", "#[A('a\nb')]\nclass X {}"},
		{"class X {\n    #[A(\n        1,\n    )]\n    public $a;\n}", "class X {\n    #[A(1,)]\n    public $a;\n}"},
		{"class X {\n    #[A(1, )]\n    public function f() {}\n}", "class X {\n    #[A(1,)]\n    public function f() {}\n}"},
		{"function f(\n    #[A(\n        1\n    )]\n    $x\n) {}", "function f(#[A(1)] $x) {}"},
		{"function f(#[A(1, )] $x) {}", "function f(#[A(1,)] $x) {}"},
		{"$f = #[A(\n    1\n)] function () {};", "$f = #[A(1)] function () {};"},
		{"#[A\\B(\n    1\n)]\nclass X {}", "#[A\\B(1)]\nclass X {}"},
		{"#[\\A\\B(\n    1\n)]\nclass X {}", "#[\\A\\B(1)]\nclass X {}"},
		{"#[a(\n    1\n)]\nclass X {}", "#[a(1)]\nclass X {}"},
		{"#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}", "#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}"},
		{"#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}", "#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}"},
		{"#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}", "#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}"},
		{"#[A('xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx')]\nclass X {}", "#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}"},
		{"#[A('xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx')]\nclass X {}", "#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}"},
		{"#[A('xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx')]\nclass X {}", "#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n)]\nclass X {}"},
		{"#[A('xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', )]\nclass X {}", "#[A(\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n    'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n)]\nclass X {}"},
		{"class X {\n    #[A(\n        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n    )]\n    public $a;\n}", "class X {\n    #[A(\n        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',\n        'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'\n    )]\n    public $a;\n}"},
	}
	for _, c := range cases {
		fixers := []fixer.Fixer{LineLength{}, NoSpacesInsideParenthesis{}, MethodArgumentSpace{}}
		CanonicalOrder(fixers)
		got := "<?php\n" + c.src + "\n"
		for _, f := range fixers {
			got, _ = apply(t, f.(fixerRule), got)
		}
		if want := "<?php\n" + c.want + "\n"; got != want {
			t.Errorf("%s\n got=%q\nwant=%q", c.src, got, want)
		}
	}
}
