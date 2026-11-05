package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestSpacesInsideParenthesesFromEngine(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"#[A(1,)]\nclass X {}", "#[A(1,)]\nclass X {}"},
		{"#[A(1, )]\nclass X {}", "#[A(1, )]\nclass X {}"},
		{"#[A(x: 1,)]\nclass X {}", "#[A(x: 1,)]\nclass X {}"},
		{"#[A(x: 1, )]\nclass X {}", "#[A(x: 1, )]\nclass X {}"},
		{"#[A(\n    1,\n)]\nclass X {}", "#[A(\n    1,\n)]\nclass X {}"},
		{"#[A(\n    1, )]\nclass X {}", "#[A(\n    1, )]\nclass X {}"},
		{"class X { #[A(1,)] public $a; }", "class X { #[A(1,)] public $a; }"},
		{"class X { #[A\\B(1,)] public $a; }", "class X { #[A\\B(1,)] public $a; }"},
		{"function f(#[A(1,)] $x) {}", "function f(#[A(1,)] $x) {}"},
		{"foo(1,);", "foo(1,);"},
		{"foo(1, );", "foo(1, );"},
		{"foo(\n    1, );", "foo(\n    1, );"},
		{"foo(\n    1,\n);", "foo(\n    1,\n);"},
		{"new A(1,);", "new A(1,);"},
		{"new A(1, );", "new A(1, );"},
		{"$a->b(1,);", "$a->b(1,);"},
		{"$a->b(1, );", "$a->b(1, );"},
		{"A::b(1,);", "A::b(1,);"},
		{"A::b(1, );", "A::b(1, );"},
		{"\\foo(1,);", "\\foo(1,);"},
		{"\\foo(1, );", "\\foo(1, );"},
		{"Foo(1,);", "Foo(1,);"},
		{"Foo(1, );", "Foo(1, );"},
		{"function g($a,) {}", "function g($a,) {}"},
		{"function g($a, ) {}", "function g($a, ) {}"},
		{"$f = function ($a,) {};", "$f = function ($a,) {};"},
		{"$f = function ($a, ) {};", "$f = function ($a, ) {};"},
		{"$f = fn ($a,) => 1;", "$f = fn ($a,) => 1;"},
		{"$f = fn ($a, ) => 1;", "$f = fn ($a, ) => 1;"},
		{"$f = function () use ($a,) {};", "$f = function () use ($a,) {};"},
		{"$f = function () use ($a, ) {};", "$f = function () use ($a, ) {};"},
		{"foo(new A([\n    1,\n], ), 2);", "foo(new A([\n    1,\n], ), 2);"},
		{"foo([\n    1,\n],);", "foo([\n    1,\n],);"},
		{"foo([\n    1,\n], );", "foo([\n    1,\n], );"},
		{"array(1, );", "array(1, );"},
		{"list($a, ) = $b;", "list($a, ) = $b;"},
		{"isset($a, );", "isset($a, );"},
		{"if (foo(1, )) {\n}", "if (foo(1, )) {\n}"},
		{"$a = array( 1 );", "$a = array( 1 );"},
		{"$a = array( 1, );", "$a = array( 1, );"},
		{"$a = ARRAY( 1 );", "$a = ARRAY( 1 );"},
		{"$a = array ( 1 );", "$a = array ( 1 );"},
		{"foo( /* c */ 1 );", "foo( /* c */ 1);"},
		{"foo( 1 /* c */ );", "foo(1 /* c */);"},
		{"$c = new A( 1 );", "$c = new A(1);"},
		{"list( $x ) = $y;", "list($x) = $y;"},
		{"foo( 1 );", "foo(1);"},
		{"foo(array( 1 ));", "foo(array( 1 ));"},
		{"foo( array(1) );", "foo(array(1));"},
		{"foo(\n    1 );", "foo(\n    1);"},
		{"foo( \n    1\n);", "foo( \n    1\n);"},
	}
	for _, c := range cases {
		got, changed := apply(t, NoSpacesInsideParenthesis{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
