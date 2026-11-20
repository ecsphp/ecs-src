package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestMethodArgumentSpaceDestructuring(t *testing.T) {
	t.Parallel()
	f := configuredFromJSON(t, MethodArgumentSpace{}, `{"after_heredoc": true, "on_multiline": "ignore"}`)
	cases := []struct{ name, src, want string }{
		// a destructuring in an argument list
		{"argument", "foo([$a , $b] = $c);", "foo([$a, $b] = $c);"},
		{"missing space", "foo([$a,$b] = $c);", "foo([$a, $b] = $c);"},
		{"nested call", "foo(bar([$a , $b] = $c) , 1);", "foo(bar([$a, $b] = $c), 1);"},
		{"nested first", "foo([[$a , $b] , $c] = $d);", "foo([[$a, $b], $c] = $d);"},
		{"nested after comma", "foo([$a , [$b , $c]] = $d);", "foo([$a, [$b, $c]] = $d);"},
		{"in an offset", "foo($x[[$a , $b] = $c]);", "foo($x[[$a, $b] = $c]);"},
		{"in a dynamic property", "foo($o->{[$a , $b] = $c});", "foo($o->{[$a, $b] = $c});"},
		{"method call", "$o->m([$a , $b] = $c , [$d , $e]);", "$o->m([$a, $b] = $c, [$d , $e]);"},
		{"before an array", "foo([$a ,$b] = $c, [$d,$e]);", "foo([$a, $b] = $c, [$d,$e]);"},
		{"constructor", "new Foo([$a , $b] = $c);", "new Foo([$a, $b] = $c);"},
		{"anonymous class", "foo(new class ([$a , $b] = $c) {});", "foo(new class ([$a, $b] = $c) {});"},
		{"call in array", "$x = [foo([$a , $b] = $c)];", "$x = [foo([$a, $b] = $c)];"},

		// what the argument list skips
		{"array", "foo($a , [$b , $c]);", "foo($a, [$b , $c]);"},
		{"destructuring in an array", "foo([[$a , $b] = $c]);", "foo([[$a , $b] = $c]);"},
		{"keyed array value", "foo(['k' => [$a , $b]] = $c);", "foo(['k' => [$a , $b]] = $c);"},
		{"arrow function array", "foo(fn () => [$a , $b] = $c);", "foo(fn () => [$a , $b] = $c);"},
		{"unpacked array", "foo(...[$a , $b]);", "foo(...[$a , $b]);"},
		{"closure body", "foo(function () { [$a , $b] = $c; } , 1);", "foo(function () { [$a , $b] = $c; }, 1);"},
		{"empty elements", "foo([, , $c] = $x);", "foo([, , $c] = $x);"},
		{"foreach", "foreach ($rows as [$a , $b]) {}", "foreach ($rows as [$a , $b]) {}"},
		{"while", "while ([$a , $b] = foo()) {}", "while ([$a , $b] = foo()) {}"},
		{"statement", "[$a , $b] = $c;", "[$a , $b] = $c;"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertFix(t, f, "<?php\n\n"+c.src+"\n", "<?php\n\n"+c.want+"\n", c.src != c.want)
		})
	}
}
