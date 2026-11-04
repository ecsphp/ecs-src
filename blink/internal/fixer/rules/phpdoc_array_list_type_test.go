package rules

import "testing"

// Cases ported from PHP-CS-Fixer's PhpdocArrayTypeFixerTest; every expected
// output is what PHP-CS-Fixer 3.95 produces for the input.
func TestPhpdocArrayTypeUpstreamCases(t *testing.T) {
	t.Parallel()

	expected32, input32 := "string", "string"
	for range 32 {
		expected32 = "array<" + expected32 + ">"
		input32 += "[]"
	}

	runUpstreamCases(t, PhpdocArrayType{}, []upstreamCase{
		{"idempotent, unsupported and literals", nil,
			"<?php\n/** @tagNotSupportingTypes string[] */\n/** @var array{'a', 'b[]', 'c'} */\n/** @var 'x|T1[][]|y' */\n/** @var \"x|T2[][]|y\" */", ""},
		{"simple", nil, "<?php /** @var array<int> */", "<?php /** @var int[] */"},
		{"nested", nil, "<?php /** @param array<array<array<array<int>>>> $x */", "<?php /** @param int[][][][] $x */"},
		{"nested with spaces", nil, "<?php /** @param array<array<array<array<int>>>> $x */", "<?php /** @param int    [][  ][]  [] $x */"},
		{"inside generic", nil, "<?php /** @return iterable<array<int>> */", "<?php /** @return iterable<int[]> */"},
		{"namespaced", nil, "<?php /** @var array<Foo\\Bar> */", "<?php /** @var Foo\\Bar[] */"},
		{"underscore", nil, "<?php /** @var array<Foo_Bar> */", "<?php /** @var Foo_Bar[] */"},
		{"union members", nil, "<?php /** @var array<bool>|array<float>|array<int>|array<string> */", "<?php /** @var array<bool>|float[]|array<int>|string[] */"},
		{"shape and callable", nil,
			"<?php\n/**\n * @param array{foo?: array<int>} $foo\n * @param callable(array<int>): array<int> $bar\n */",
			"<?php\n/**\n * @param array{foo?: int[]} $foo\n * @param callable(int[]): int[] $bar\n */"},
		{"literals and closure return", nil,
			"<?php\n/**\n * @var array<array{'a'}>\n * @var \\Closure(iterable<\\DateTime>): array<void>\n * @var array<Yes>|'No[][]'|array<array<Yes>>|'No[]'\n */",
			"<?php\n/**\n * @var array{'a'}[]\n * @var \\Closure(iterable<\\DateTime>): void[]\n * @var Yes[]|'No[][]'|Yes[][]|'No[]'\n */"},
		{"nullable", nil,
			"<?php\n/**\n * @param ?array<type> $a\n * @param ?array<array<type>> $b\n * @param ?'literal[]' $c\n */",
			"<?php\n/**\n * @param ?type[] $a\n * @param ?type[][] $b\n * @param ?'literal[]' $c\n */"},
		{"parenthesized union", nil, "<?php /** @var array<Foo|Bar> */", "<?php /** @var (Foo|Bar)[] */"},
		{"intersection then array", nil, "<?php /** @var (Foo&Bar)|array<Baz> */", "<?php /** @var (Foo&Bar)|Baz[] */"},
		{"deeply nested 32", nil, "<?php /** @var " + expected32 + " */", "<?php /** @var " + input32 + " */"},
		{"covariant generic", nil, "<?php /** @return array<Foo<covariant TEntity>> */", "<?php /** @return Foo<covariant TEntity>[] */"},
	})
}

// Cases ported from PHP-CS-Fixer's PhpdocListTypeFixerTest.
func TestPhpdocListTypeUpstreamCases(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpdocListType{}, []upstreamCase{
		{"unsupported tag", nil, "<?php /** @tagNotSupportingTypes string[] */", ""},
		{"keyed array", nil, "<?php /** @var array<string, int> */", ""},
		{"nested keyed array", nil, "<?php /** @var array<int, array<string, bool>> */", ""},
		{"class-string key", nil, "<?php /** @var array<class-string<Foo>, bool> */", ""},
		{"int range key", nil, "<?php /** @var array<int<1, 10>, bool> */", ""},
		{"const key", nil, "<?php /** @var array<Foo::BAR_*, bool> */", ""},
		{"literal union key", nil, "<?php /** @var array<'foo'|'bar', bool> */", ""},
		{"empty shape", nil, "<?php /** @var array{} */", ""},
		{"shape list", nil, "<?php /** @var array{string, string, string} */", ""},
		{"shape double quotes", nil, "<?php /** @var array{\"a\", \"b[]\", \"c\"} */", ""},
		{"shape single quotes", nil, "<?php /** @var array{'a', 'b[]', 'c'} */", ""},
		{"simple", nil, "<?php /** @var list<Foo> */", "<?php /** @var array<Foo> */"},
		{"uppercase", nil, "<?php /** @var list<Foo> */", "<?php /** @var ARRAY<Foo> */"},
		{"nullable", nil, "<?php /** @var ?list<Foo> */", "<?php /** @var ?array<Foo> */"},
		{"union members", nil, "<?php /** @var list<bool>|list<float>|list<int>|list<string> */", "<?php /** @var array<bool>|list<float>|array<int>|list<string> */"},
		{"non-empty", nil, "<?php /** @var non-empty-list<string> */", "<?php /** @var non-empty-array<string> */"},
		{"nested shapes", nil, "<?php /** @var array{string, list<array{Foo, list<int>, Bar}>} */", "<?php /** @var array{string, array<array{Foo, array<int>, Bar}>} */"},
		{"int range value", nil, "<?php /** @var list<int<1, 10>> */", "<?php /** @var array<int<1, 10>> */"},
		{"word boundary", nil,
			"<?php\n/** @var list<Foo> */\n/** @var \\SplFixedArray<Foo> */\n/** @var \\KůňArray<Foo> */\n/** @var \\My_Array<Foo> */\n/** @var \\My2Array<Foo> */",
			"<?php\n/** @var array<Foo> */\n/** @var \\SplFixedArray<Foo> */\n/** @var \\KůňArray<Foo> */\n/** @var \\My_Array<Foo> */\n/** @var \\My2Array<Foo> */"},
	})
}

// Cases ported from PHP-CS-Fixer's PhpdocTypesNoDuplicatesFixerTest. The fixer
// is a deprecated proxy for PhpdocNoDuplicateTypesFixer.
func TestPhpdocTypesNoDuplicatesUpstreamCases(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpdocTypesNoDuplicates{}, []upstreamCase{
		{"scalar union", nil, "<?php /** @var string|null|int */", "<?php /** @var string|string|null|int */"},
		{"generic member", nil, "<?php /** @var list<int> */", "<?php /** @var list<int|int> */"},
		{"nested generics", nil, "<?php /** @var string|array<string, string>|array<int, int> */", "<?php /** @var string|string|array<string, string|string>|array<int, int|int> */"},
		{"reordered generic members", nil, "<?php /** @var list<float|int> */", "<?php /** @var list<float|int>|list<int|float> */"},
		{"keyed and unkeyed", nil, "<?php /** @var array<int|float, int|string|float|null>|string */", "<?php /** @var array<int|float, int|int|string|float|null>|array<float|int, string|null|int|int|float>|string|string */"},
		{"shape", nil, "<?php /** @var array{foo: string} */", "<?php /** @var array{foo: string|string} */"},
		{"class union", nil, "<?php /** @var Bar|Baz */", "<?php /** @var Bar|Bar|Baz */"},
		{"distinct fqcn", nil, "<?php /** @var Bar|\\Bar */", ""},
		{"fqcn duplicates", nil, "<?php /** @var string|\\Foo\\Baz|\\Foo\\Bar|Foo\\Baz */", "<?php /** @var string|\\Foo\\Baz|\\Foo\\Bar|\\Foo\\Baz|Foo\\Baz */"},
	})
}
