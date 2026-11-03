package rules

import "testing"

// Cases ported from PHP-CS-Fixer's LogicalOperatorsFixerTest (tests/Fixer/Operator), then
// invented ones (named "extra#N"); every expected output is what PHP-CS-Fixer
// 3.95 produces for the input.
func TestLogicalOperatorsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, LogicalOperators{}, []portConstructsCase{
		{"provideFixCases#0", nil, "<?php if ($a == \"foo\" and $b == \"bar\") {}", "<?php if ($a == \"foo\" && $b == \"bar\") {}"},
		{"provideFixCases#1", nil, "<?php if ($a == \"foo\" or $b == \"bar\") {}", "<?php if ($a == \"foo\" || $b == \"bar\") {}"},
		{"provideFixCases#2", nil, "<?php if ($a == \"foo\" and ($b == \"bar\" or $c == \"baz\")) {}", "<?php if ($a == \"foo\" && ($b == \"bar\" || $c == \"baz\")) {}"},
		{"provideFixCases#3", nil, "<?php if ($a == \"foo\" and ($b == \"bar\" || $c == \"baz\")) {}", "<?php if ($a == \"foo\" && ($b == \"bar\" || $c == \"baz\")) {}"},
		{"provideFixCases#4", nil, "<?php if ($a == \"foo\" && ($b == \"bar\" or $c == \"baz\")) {}", "<?php if ($a == \"foo\" && ($b == \"bar\" || $c == \"baz\")) {}"},
		{"provideFixCases#5", nil, "<?php if ($a == \"foo\" AND ($b == \"bar\" OR $c == \"baz\")) {}", "<?php if ($a == \"foo\" && ($b == \"bar\" || $c == \"baz\")) {}"},
		{"provideFixCases#6", nil, "<?php if ($a == \"foo\" and ($b == \"bar\" OR $c == \"baz\")) {}", "<?php if ($a == \"foo\" && ($b == \"bar\" || $c == \"baz\")) {}"},
		{"provideFixCases#7", nil, "<?php if ($a == \"foo\" AND ($b == \"bar\" or $c == \"baz\")) {}", "<?php if ($a == \"foo\" && ($b == \"bar\" || $c == \"baz\")) {}"},
		{"extra#0", nil, "<?php\n$x = $a and $b or $c;\n$y = $a AND $b OR $c;\n$z = $a and($b);\n", "<?php\n$x = $a && $b || $c;\n$y = $a && $b || $c;\n$z = $a &&($b);\n"},
		{"extra#1", nil, "<?php\nf(and: 1, or: 2);\nenum E { case And; case Or = 'or'; }\n$a->or(); A::AND; class B { const OR = 1; function and() {} }\n", "<?php\nf(and: 1, or: 2);\nenum E { case And; case Or = 'or'; }\n$a->or(); A::AND; class B { const OR = 1; function and() {} }\n"},
		{"extra#2", nil, "<?php\nclass A { use T { x as or; } }\n$x = $a xor $b;\n", "<?php\nclass A { use T { x as or; } }\n$x = $a xor $b;\n"},
		{"extra#3", nil, "<?php\nif ($a and\n    $b) {}\n$c = $d or throw new E();\n", "<?php\nif ($a &&\n    $b) {}\n$c = $d || throw new E();\n"},
		// reserved words PHP's tokenizer reads as names: trait aliases and
		// method references, typed and listed constants, by-reference methods
		{"extra#4", nil, "<?php\nclass C { use T { foo as protected and; and as protected or; bar as private OR; Or as public; } }\nclass D { const int AND = 3; const A = 1, OR = 2; public function &and() {} public static function & or () {} }\ninterface I { const ?string And = null, Or = 'x'; }\n$a?->and($b and $c); $d = $e or $f;\n", "<?php\nclass C { use T { foo as protected and; and as protected or; bar as private OR; Or as public; } }\nclass D { const int AND = 3; const A = 1, OR = 2; public function &and() {} public static function & or () {} }\ninterface I { const ?string And = null, Or = 'x'; }\n$a?->and($b && $c); $d = $e || $f;\n"},
	})
}
