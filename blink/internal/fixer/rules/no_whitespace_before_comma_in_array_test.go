package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestNoWhitespaceBeforeCommaInArrayDestructuring(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		// a destructuring is no array
		{"short list", "[$a, $b ,$c] = [1, 2, 3];", "[$a, $b ,$c] = [1, 2, 3];"},
		{"long list", "list($a ,$b) = $x;", "list($a ,$b) = $x;"},
		{"foreach value", "foreach ($rows as [$a ,$b]) {}", "foreach ($rows as [$a ,$b]) {}"},
		{"foreach key and value", "foreach ($rows as $k => [$a ,$b]) {}", "foreach ($rows as $k => [$a ,$b]) {}"},
		{"keyed", "['x' => $a , 'y' => $b] = $arr;", "['x' => $a , 'y' => $b] = $arr;"},
		{"nested after comma", "[$a , [$b , $c]] = $arr;", "[$a , [$b , $c]] = $arr;"},
		{"nested first", "[[$a , $b] , $c] = $arr;", "[[$a , $b] , $c] = $arr;"},
		{"nested first key", "[[1 , 2][0] => $a] = $arr;", "[[1 , 2][0] => $a] = $arr;"},
		{"chained", "[$a , $b] = [$c , $d] = $e;", "[$a , $b] = [$c , $d] = $e;"},
		{"condition", "while ([$a , $b] = foo()) {}", "while ([$a , $b] = foo()) {}"},
		{"after a closing parenthesis", "if ($x) [$a , $b] = $c;", "if ($x) [$a , $b] = $c;"},
		{"empty elements", "[, , $c] = $x;", "[, , $c] = $x;"},
		{"argument", "foo([$a , $b] = $c);", "foo([$a , $b] = $c);"},
		{"argument in array", "$x = [foo([$a , $b] = $c)];", "$x = [foo([$a , $b] = $c)];"},
		{"method named array", "$x = [$o->array([$a , $b] = $c)];", "$x = [$o->array([$a , $b] = $c)];"},
		{"argument in long array", "$x = array(foo([$a , $b] = $c));", "$x = array(foo([$a , $b] = $c));"},
		{"parentheses in long array", "$x = array(([$a , $b] = $c));", "$x = array(([$a , $b] = $c));"},
		{"closure in array", "$x = [function () { [$a , $b] = $c; }];", "$x = [function () { [$a , $b] = $c; }];"},

		// an array next to or inside a destructuring
		{"same line", "[$a , $b] = [1 , 2]; $c = [3 , 4];", "[$a , $b] = [1, 2]; $c = [3, 4];"},
		{"nested arrays", "[$x , $y] = [[1 , 2] , [3 , 4]];", "[$x , $y] = [[1, 2], [3, 4]];"},
		{"offset", "[$a[0] , $b->c] = [$d , $e];", "[$a[0] , $b->c] = [$d, $e];"},
		{"keyed value", "['k' => [$a , $b]] = $arr;", "['k' => [$a, $b]] = $arr;"},
		{"keyed value empty elements", "['k' => [, , $c]] = $x;", "['k' => [,, $c]] = $x;"},
		{"foreach keyed value", "foreach ($rows as ['id' => $id , 'tags' => [$t1 , $t2]]) {}", "foreach ($rows as ['id' => $id , 'tags' => [$t1, $t2]]) {}"},
		{"foreach property key", "foreach ($rows as $this->k => [$a , $b]) {}", "foreach ($rows as $this->k => [$a, $b]) {}"},
		{"array in key", "['k' . [1 , 2][0] => $a , 'm' => $b] = $arr;", "['k' . [1, 2][0] => $a , 'm' => $b] = $arr;"},
		{"long list element", "list($a , [$b , $c]) = $arr;", "list($a , [$b, $c]) = $arr;"},
		{"arrow function", "$f = fn ($x) => [$a , $b] = $x;", "$f = fn ($x) => [$a, $b] = $x;"},

		// a destructuring inside an array
		{"in array", "$x = [[$a , $b] = $c , 2];", "$x = [[$a, $b] = $c, 2];"},
		{"nested in array", "$x = [[$a , [$b , $c]] = $d];", "$x = [[$a, [$b, $c]] = $d];"},
		{"keyed in array", "$x = ['k' => [[$a , $b]] = foo()];", "$x = ['k' => [[$a, $b]] = foo()];"},
		{"empty elements in array", "$x = [[, , $c] = $y];", "$x = [[,, $c] = $y];"},
		{"parentheses in array", "$x = [([$a , $b] = $c)];", "$x = [([$a, $b] = $c)];"},
		{"dynamic property in array", "$x = [$o->{[$a , $b] = $c}];", "$x = [$o->{[$a, $b] = $c}];"},
		{"match arm in array", "$x = [match ($y) { 1 , 2 => [$a , $b] = $c }];", "$x = [match ($y) { 1 , 2 => [$a, $b] = $c }];"},
		{"in long array", "$x = array([$a , $b] = $c);", "$x = array([$a, $b] = $c);"},
		{"nested in long array", "$x = array([[$a , $b] , $c] = $d);", "$x = array([[$a, $b], $c] = $d);"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertFix(t, NoWhitespaceBeforeCommaInArray{}, "<?php\n\n"+c.src+"\n", "<?php\n\n"+c.want+"\n", c.src != c.want)
		})
	}
}
