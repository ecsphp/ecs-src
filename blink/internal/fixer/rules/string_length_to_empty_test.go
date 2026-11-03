package rules

import "testing"

// Cases ported from PHP-CS-Fixer's StringLengthToEmptyFixerTest.
func TestStringLengthToEmptyUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, StringLengthToEmpty{}, []upstreamFixCase{
		{name: `#0`,
			expected: `<?php $a = '' === $b;`,
			input:    `<?php $a = 0 === strlen($b);`},
		{name: `casing`,
			expected: `<?php $a1 = '' === $b ?>`,
			input:    `<?php $a1 = 0 === \STRlen($b) ?>`},
		{name: `nested`,
			expected: `<?php $a2 = '' === foo('' === foo('' === foo('' === foo('' === foo('' === foo('' === $b))))));`,
			input:    `<?php $a2 = 0 === strlen(foo(0 === strlen(foo(0 === strlen(foo(0 === strlen(foo(0 === strlen(foo(0 === strlen(foo(0 === strlen($b)))))))))))));`},
		{name: `#3`,
			expected: `<?php $a3 = '' !== $b;`,
			input:    `<?php $a3 = 0 !== strlen($b);`},
		{name: `#4`, expected: `<?php $a4 = 0 <= strlen($b);`},
		{name: `#5`,
			expected: `<?php $a5 = '' === $b;`,
			input:    `<?php $a5 = 0 >= strlen($b);`},
		{name: `#6`,
			expected: `<?php $a6 = '' !== $b;`,
			input:    `<?php $a6 = 0 < strlen($b);`},
		{name: `#7`, expected: `<?php $a7 = 0 > strlen($b);`},
		{name: `#8`, expected: `<?php $a8 = 1 === strlen($b);`},
		{name: `#9`, expected: `<?php $a9 = 1 !== strlen($b);`},
		{name: `#10`,
			expected: `<?php $a10 = '' !== $b;`,
			input:    `<?php $a10 = 1 <= strlen($b);`},
		{name: `#11`, expected: `<?php $a11 = 1 >= strlen($b);`},
		{name: `#12`, expected: `<?php $a12 = 1 < strlen($b);`},
		{name: `#13`,
			expected: `<?php $a13 = '' === $b;`,
			input:    `<?php $a13 = 1 > strlen($b);`},
		{name: `#14`,
			expected: `<?php $a14 = $b === '';`,
			input:    `<?php $a14 = strlen($b) === 0;`},
		{name: `#15`,
			expected: `<?php $a15 = $b !== '';`,
			input:    `<?php $a15 = strlen($b) !== 0;`},
		{name: `#16`,
			expected: `<?php $a16 = $b === '';`,
			input:    `<?php $a16 = strlen($b) <= 0;`},
		{name: `#17`, expected: `<?php $a17 = strlen($b) >= 0;`},
		{name: `#18`, expected: `<?php $a18 = strlen($b) < 0;`},
		{name: `#19`,
			expected: `<?php $a19 = $b !== '';`,
			input:    `<?php $a19 = strlen($b) > 0;`},
		{name: `#20`, expected: `<?php $a20 = strlen($b) === 1;`},
		{name: `#21`, expected: `<?php $a21 = strlen($b) !== 1;`},
		{name: `#22`, expected: `<?php $a22 = strlen($b) <= 1;`},
		{name: `#23`,
			expected: `<?php $a23 = $b !== '';`,
			input:    `<?php $a23 = strlen($b) >= 1;`},
		{name: `#24`,
			expected: `<?php $a24 = $b === '';`,
			input:    `<?php $a24 = strlen($b) < 1;`},
		{name: `#25`, expected: `<?php $a25 = strlen($b) > 1;`},
		{name: `#26`,
			expected: `<?php $e = 0 === foo() ? -1 : '' === $a;`,
			input:    `<?php $e = 0 === foo() ? -1 : 0 === strlen($a);`},
		{name: `#27`,
			expected: `<?php $x = /* 1 */ $b /* 2 */ ->a !== '';`,
			input:    `<?php $x = strlen(/* 1 */ $b /* 2 */ ->a) >= 1;`},
		{name: `#28`,
			expected: `<?php $y = $b[0] === '';`,
			input:    `<?php $y = strlen($b[0]) < 1;`},
		{name: `#29`,
			expected: `<?php $y1 = $b[0]->$a[$a++](1) /* 1 */  === '';`,
			input:    `<?php $y1 = strlen($b[0]->$a[$a++](1) /* 1 */ ) < 1;`},
		{name: `#30`,
			expected: `<?php $z = '' === $b[1]->foo(++$i, static function () use ($z){ return $z + 1;});`,
			input:    `<?php $z = 0 === strlen($b[1]->foo(++$i, static function () use ($z){ return $z + 1;}));`},
		{name: `#31`,
			expected: `<?php if ((string) $node !== '') { echo 1; }`,
			input:    `<?php if (\strlen((string) $node) > 0) { echo 1; }`},
		{name: `do not fix`, expected: `<?php
//-----------------------------------
// operator precedence

$a01 = 0 === strlen($b) ** $c;
$a03 = 0 === strlen($b) % $c;
$a04 = 0 === strlen($b) / $c;
$a05 = 0 === strlen($b) * $c;
$a06 = 0 === strlen($b) + $c;
$a07 = 0 === strlen($b) - $c;
$a08 = 0 === strlen($b) . $c;
$a09 = 0 === strlen($b) >> $c;
$a10 = 0 === strlen($b) << $c;

$a01n = strlen($b) === 0 ** $c;
$a03n = strlen($b) === 0 % $c;
$a04n = strlen($b) === 0 / $c;
$a05n = strlen($b) === 0 * $c;
$a06n = strlen($b) === 0 + $c;
$a07n = strlen($b) === 0 - $c;
$a08n = strlen($b) === 0 . $c;
$a09n = strlen($b) === 0 >> $c;
$a10n = strlen($b) === 0 << $c;

$b = "a";

$c = 0 === strlen($b) - 1;
var_dump($c);

$c = "" === $b - 1;
var_dump($c);

//-----------------------------------
// type juggle

$d = false;

$e = 0 === strlen($d) ? -1 : 0;
var_dump($e);

$e = "" === $d ? -1 : 0;
var_dump($e);

//-----------------------------------
// wrong argument count

$f = strlen(1,2);
$g = \strlen(1,2,3);

//-----------------------------------
// others

$h = 0 === (string) strlen($b);
$i = 0 === @strlen($b);
$j = 0 === !strlen($b);

$jj = 2 === strlen($b);
$jk = __DIR__ === strlen($b);
$jl = 'X' !== strlen($b);

$jj = strlen($b) === 2;
$jk = strlen($b) === __DIR__;
$jl = strlen($b) !== 'X';

//-----------------------------------
// not global calls

$k = 0 === $a->strlen($b);
$l = 0 === Foo::strlen($b);

//-----------------------------------
// comments

// $a = 0 === strlen($b);
# $a = 0 === strlen($b);
/* $a = 0 === strlen($b); */
/** $a = 0 === strlen($b); */
`},
		{name: `#33`,
			expected: `<?php $e = ($a = trim($b)) !== '';`,
			input:    `<?php $e = \strlen($a = trim($b)) > 0;`},
		{name: `#34`,
			expected: `<?php if ('' === ($value = foo())) { echo 2; }`,
			input:    `<?php if (0 === \strlen($value = foo())) { echo 2; }`},
		{name: `#35`, expected: `<?php
                $a02 = 0 === strlen($b) instanceof stdClass;
                $a02n = strlen($b) === 0 instanceof stdClass;`},
	})
}

func TestStringLengthToEmpty(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, StringLengthToEmpty{}, []upstreamFixCase{
		{name: "function declared in the namespace",
			expected: "<?php namespace A;\nfunction strlen($a) { return 1; }\n$b = 0 === strlen($c);\n"},
		{name: "function declared in another namespace",
			expected: "<?php namespace A { function strlen($a) { return 1; } }\nnamespace B { $b = '' === $c; }\n",
			input:    "<?php namespace A { function strlen($a) { return 1; } }\nnamespace B { $b = 0 === strlen($c); }\n"},
		{name: "global namespace declaration",
			expected: "<?php function strlen($a) { return 1; }\n$b = '' === $c;\n",
			input:    "<?php function strlen($a) { return 1; }\n$b = 0 === strlen($c);\n"},
		{name: "method named strlen does not shadow",
			expected: "<?php namespace A;\nclass B { function strlen($a) { return '' === $a; } }\n",
			input:    "<?php namespace A;\nclass B { function strlen($a) { return 0 === strlen($a); } }\n"},
		{name: "imported function", expected: "<?php namespace A;\nuse function B\\strlen;\n$b = 0 === strlen($c);\n"},
		{name: "aliased import", expected: "<?php namespace A;\nuse function B\\foo as strlen;\n$b = 0 === strlen($c);\n"},
		{name: "global import",
			expected: "<?php namespace A;\nuse function strlen;\n$b = '' === $c;\n",
			input:    "<?php namespace A;\nuse function strlen;\n$b = 0 === strlen($c);\n"},
		{name: "fully qualified call ignores the namespace",
			expected: "<?php namespace A;\nfunction strlen($a) { return 1; }\n$b = '' === $c;\n",
			input:    "<?php namespace A;\nfunction strlen($a) { return 1; }\n$b = 0 === \\strlen($c);\n"},
		{name: "first-class callable", expected: "<?php $f = strlen(...);\n"},
		{name: "attribute", expected: "<?php #[strlen(1) > 0]\nfunction f() {}\n"},
		{name: "namespaced call", expected: "<?php $a = 0 === A\\strlen($b);\n"},
		{name: "keeps parentheses around a magic constant",
			expected: "<?php $a = (__DIR__) === '';\n",
			input:    "<?php $a = strlen(__DIR__) === 0;\n"},
		{name: "whitespace inside the call",
			expected: "<?php $a = '' ===  $b ;\n",
			input:    "<?php $a = 0 === strlen( $b );\n"},
		{name: "trailing comma keeps the parentheses",
			expected: "<?php $a = '' === ($b,);\n",
			input:    "<?php $a = 0 === strlen($b,);\n"},
	})
}
