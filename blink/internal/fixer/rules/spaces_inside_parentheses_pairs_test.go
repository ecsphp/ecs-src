package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// The expected output comes from the PHP engine, with the two rules together.
func TestSpacesInsideParenthesesWithTrimArraySpaces(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"$x = array( 1, 2 );", "$x = array(1, 2);"},
		{"$x = array( );", "$x = array();"},
		{"$x = array( 1, );", "$x = array(1, );"},
		{"$x = ARRAY( 1 );", "$x = ARRAY(1);"},
		{"$x = array ( 1 );", "$x = array (1);"},
		{"$x = array( // c\n    1\n);", "$x = array( // c\n    1\n);"},
		{"$x = array( /* c */ 1 /* d */ );", "$x = array(/* c */ 1 /* d */);"},
		{"$x = [ 1 ];", "$x = [1];"},
		{"foo( array( foo( 1 ) ) );", "foo(array(foo(1)));"},
		{"$f = fn(array $a ) => array( $a );", "$f = fn(array $a) => array($a);"},
		{"$x = array(\n    1 );", "$x = array(\n    1);"},
		{"$a[ 1 ];", "$a[ 1 ];"},
		{"$x = array(1, 2);", "$x = array(1, 2);"},
	}
	for _, c := range cases {
		fixers := []fixer.Fixer{NoSpacesInsideParenthesis{}, TrimArraySpaces{}}
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

// The expected output comes from the PHP engine, with the two rules together.
func TestSpacesInsideParenthesesWithFunctionDeclaration(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"function g(\n    $a, ) {}", "function g(\n    $a,) {}"},
		{"function g($a,\n    $b, ) {}", "function g($a,\n    $b,) {}"},
		{"function g( $a ) {}", "function g($a) {}"},
		{"function g( $a, ) {}", "function g($a) {}"},
		{"function g(\t$a\t) {}", "function g($a) {}"},
		{"$f = function ( $a ) {};", "$f = function ($a) {};"},
		{"$f = function ( $a, ) use ( $b, ) {};", "$f = function ($a) use ($b) {};"},
		{"$f = fn ( $a ) => 1;", "$f = fn ($a) => 1;"},
		{"$f = fn ( $a, ) => 1;", "$f = fn ($a) => 1;"},
		{"function g(\n    $a\n) {}", "function g(\n    $a\n) {}"},
		{"function g( /* c */ $a /* d */ ) {}", "function g(/* c */ $a /* d */) {}"},
		{"abstract class X { abstract function f( $a ); }", "abstract class X { abstract function f($a); }"},
		{"function g( ) {}", "function g() {}"},
	}
	for _, c := range cases {
		fixers := []fixer.Fixer{NoSpacesInsideParenthesis{}, withPHPDefaults(FunctionDeclaration{})}
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
