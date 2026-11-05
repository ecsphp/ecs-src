package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestFunctionDeclarationInnerEdge(t *testing.T) {
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
		got, changed := apply(t, withPHPDefaults(FunctionDeclaration{}).(fixerRule), "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
