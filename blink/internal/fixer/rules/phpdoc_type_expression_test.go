package rules

import "testing"

// TestPhpdocTypeExpressions checks that PhpdocTypes and PhpdocScalar rewrite
// every type name of a typed annotation, inside generics, shapes and callables
// too, and leave shape keys, quoted literals and the other annotations alone.
// The expected outputs come from PHP-CS-Fixer.
func TestPhpdocTypeExpressions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		rule     fixerRule
		in, want string
	}{
		{"types: composite types", PhpdocTypes{}.WithConfig(map[string]any{}).(fixerRule), "<?php\n/**\n * @param array<int, Bool> $o\n * @param array{a: Int, Bool: String, 'Int': Float} $p\n * @param Array<Int> $q\n * @param Int[] $r\n * @param non-empty-String|Null $s\n * @param callable(Int $x, Bool): Void $t\n * @param int | Bool $u\n * @param Foo::BAR|\\Bool $v\n * @param array<Int, array{x: Bool}> $w desc Bool here\n * @return ?Int\n * @var Closure(Int): Bool\n * @method Int foo(Bool $x)\n * @param 'Int'|\"Bool\" $z\n * @param list<Int>|iterable<Mixed> $y\n * @param Self|Static|$This $x\n * @param Integer|Boolean $aa\n * @param int<0, Max> $ab\n */\nfunction f($o, $p) {}\n", "<?php\n/**\n * @param array<int, bool> $o\n * @param array{a: int, Bool: string, 'Int': float} $p\n * @param array<int> $q\n * @param int[] $r\n * @param non-empty-String|null $s\n * @param callable(int $x, bool): void $t\n * @param int | bool $u\n * @param Foo::BAR|\\Bool $v\n * @param array<int, array{x: bool}> $w desc Bool here\n * @return ?int\n * @var Closure(int): bool\n * @method int foo(Bool $x)\n * @param 'Int'|\"Bool\" $z\n * @param list<int>|iterable<mixed> $y\n * @param self|static|$this $x\n * @param integer|boolean $aa\n * @param int<0, Max> $ab\n */\nfunction f($o, $p) {}\n"},
		{"types: typed tags", PhpdocTypes{}.WithConfig(map[string]any{}).(fixerRule), "<?php\n/**\n * @use Foo<Int>\n * @template-extends Foo<Int>\n * @extends Foo<Int>\n * @param-out Int $x\n * @phpstan-var Int\n * @psalm-var Int\n * @type Int\n * @phpstan-type Alias Int\n * @psalm-type Alias2 = Int\n * @phpstan-import-type Int from Foo\n * @throws Int\n * @template T of Int\n */\nclass F {}\n", "<?php\n/**\n * @use Foo<Int>\n * @template-extends Foo<Int>\n * @extends Foo<int>\n * @param-out int $x\n * @phpstan-var int\n * @psalm-var int\n * @type int\n * @phpstan-type Alias Int\n * @psalm-type Alias2 = Int\n * @phpstan-import-type int from Foo\n * @throws int\n * @template T of Int\n */\nclass F {}\n"},
		{"scalar: composite types", PhpdocScalar{}, "<?php\n/**\n * @param array<int, boolean> $o\n * @param array{a: Integer, boolean: String, 'Integer': double} $p\n * @param Array<Integer> $q\n * @param Integer[] $r\n * @param non-empty-String|Null $s\n * @param callable(Integer $x, boolean): Void $t\n * @param int | boolean $u\n * @param Foo::BAR|\\boolean $v\n * @param array<Integer, array{x: boolean}> $w desc boolean here\n * @return ?Integer\n * @var Closure(Integer): boolean\n * @method Integer foo(boolean $x)\n * @param 'Integer'|\"boolean\" $z\n * @param list<Integer>|iterable<Mixed> $y\n * @param Self|Static|$This $x\n * @param Integereger|booleanean $aa\n * @param int<0, Max> $ab\n */\nfunction f($o, $p) {}\n", "<?php\n/**\n * @param array<int, bool> $o\n * @param array{a: Integer, boolean: String, 'Integer': float} $p\n * @param Array<Integer> $q\n * @param Integer[] $r\n * @param non-empty-String|Null $s\n * @param callable(Integer $x, bool): Void $t\n * @param int | bool $u\n * @param Foo::BAR|\\boolean $v\n * @param array<Integer, array{x: bool}> $w desc boolean here\n * @return ?Integer\n * @var Closure(Integer): bool\n * @method Integer foo(boolean $x)\n * @param 'Integer'|\"boolean\" $z\n * @param list<Integer>|iterable<Mixed> $y\n * @param Self|Static|$This $x\n * @param Integereger|booleanean $aa\n * @param int<0, Max> $ab\n */\nfunction f($o, $p) {}\n"},
		{"scalar: default types", PhpdocScalar{}, "<?php\n/**\n * @return never-return|no-return\n * @param array<int, str> $a\n */\nfunction f($a) {}\n", "<?php\n/**\n * @return never-return|no-return\n * @param array<int, string> $a\n */\nfunction f($a) {}\n"},
		{"scalar: never types", PhpdocScalar{}.WithConfig(map[string]any{"types": []any{"never-return", "no-return", "str"}}).(fixerRule), "<?php\n/**\n * @return never-return|no-return\n * @param array<int, str> $a\n */\nfunction f($a) {}\n", "<?php\n/**\n * @return never|never\n * @param array<int, string> $a\n */\nfunction f($a) {}\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, c.rule, c.in); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
