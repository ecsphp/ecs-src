package rules

import "testing"

// Cases ported from PHP-CS-Fixer's YieldFromArrayToYieldsFixerTest; every
// expected output is what PHP-CS-Fixer produces for the input.
func TestYieldFromArrayToYieldsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, YieldFromArrayToYields{}, []aliasUpstreamCase{
		{"not an array, no change", nil, "<?php function f() { yield from foo(); }", "<?php function f() { yield from foo(); }"},
		{"array to yields", nil, "<?php function f() { yield from [1, 2, 3]; }", "<?php function f() {  yield 1; yield 2; yield 3; }"},
		{"array() syntax", nil, "<?php function f() { yield from array(11, 22, 33); }", "<?php function f() {  yield 11; yield 22; yield 33; }"},
		{"array with whitespace", nil, "<?php function f() { yield from array  (11, 22, 33); }", "<?php function f() {    yield 11; yield 22; yield 33; }"},
		{"block comment before parenthesis", nil, "<?php function f() { yield from array/* ugly comment */(11, 22, 33); }", "<?php function f() {  /* ugly comment */yield 11; yield 22; yield 33; }"},
		{"doc comment before parenthesis", nil, "<?php function f() { yield from array/** ugly doc */(11, 22, 33); }", "<?php function f() {  /** ugly doc */yield 11; yield 22; yield 33; }"},
		{"trailing comma", nil, "<?php function f() { yield from [111, 222, 333,]; }", "<?php function f() {  yield 111; yield 222; yield 333; }"},
		{"nested arrays", nil, "<?php function f() {\n    yield from [\n        [1, 2],\n        [3, 4],\n    ];\n}", "<?php function f() {\n     \n        yield [1, 2];\n        yield [3, 4];\n    \n}"},
		{"nested array() ", nil, "<?php function f() {\n    yield from [\n        array(1, 2),\n        array(3, 4),\n    ];\n}", "<?php function f() {\n     \n        yield array(1, 2);\n        yield array(3, 4);\n    \n}"},
		{"multiline trailing commas", nil, "<?php function f() {\n    yield from [\n        1,\n        2,\n        3,\n    ];\n}", "<?php function f() {\n     \n        yield 1;\n        yield 2;\n        yield 3;\n    \n}"},
		{"comments preserved", nil, "<?php function f() {\n    yield from [\n        // uno\n        1,\n        // dos\n        2,\n        // tres\n        3,\n    ];\n}", "<?php function f() {\n     \n        // uno\n        yield 1;\n        // dos\n        yield 2;\n        // tres\n        yield 3;\n    \n}"},
		{"keys and closures", nil, "<?php function f() {\n    yield from [\n        random_key() => true,\n        \"foo\" => foo(1, 2),\n        \"bar\" => function ($x, $y) { return max($x, $y); },\n        \"baz\" => function () { yield [1, 2]; },\n    ];\n}", "<?php function f() {\n     \n        yield random_key() => true;\n        yield \"foo\" => foo(1, 2);\n        yield \"bar\" => function ($x, $y) { return max($x, $y); };\n        yield \"baz\" => function () { yield [1, 2]; };\n    \n}"},
		{"multiple functions", nil, "<?php\n    function f1() { yield from [0, 1, 2]; }\n    function f2() { yield from [3, 4, 5]; }\n    function f3() { yield from [6, 7, 8]; }\n", "<?php\n    function f1() {  yield 0; yield 1; yield 2; }\n    function f2() {  yield 3; yield 4; yield 5; }\n    function f3() {  yield 6; yield 7; yield 8; }\n"},
		{"multiple functions mixed", nil, "<?php\n    function f1() { yield from array(0, 1); }\n    function f2() { yield from [2, 3]; }\n    function f3() { yield from array(4, 5); }\n    function f4() { yield from [6, 7]; }\n    function f5() { yield from array(8, 9); }\n", "<?php\n    function f1() {  yield 0; yield 1; }\n    function f2() {  yield 2; yield 3; }\n    function f3() {  yield 4; yield 5; }\n    function f4() {  yield 6; yield 7; }\n    function f5() {  yield 8; yield 9; }\n"},
		{"yield from in return array, no change", nil, "<?php\n    function foo() {\n        return [\n            1,\n            yield from [2, 3],\n            4,\n        ];\n    }\n", "<?php\n    function foo() {\n        return [\n            1,\n            yield from [2, 3],\n            4,\n        ];\n    }\n"},
		{"consecutive with nesting", nil, "<?php function foo() {\n    yield from [0, 1];\n    yield from [2, 3];\n    yield from [4, yield from [5, 6], 7];\n    yield from [8, 9];\n}", "<?php function foo() {\n     yield 0; yield 1;\n     yield 2; yield 3;\n    yield from [4, yield from [5, 6], 7];\n     yield 8; yield 9;\n}"},
		{"unrelated foreach before", nil, "<?php function foo()\n{\n    foreach ([] as $x) {}\n    yield from [1, 2, 3];\n}", "<?php function foo()\n{\n    foreach ([] as $x) {}\n     yield 1; yield 2; yield 3;\n}"},
		{"skip empty arrays", nil, "<?php\nfunction foo1()\n{\n    yield from [/*empty*/ ];\n}\nfunction bar()\n{\n    yield from [];\n    yield from array();\n}\nfunction baz()\n{\n    yield from [];\n    yield from [1, 2];\n    yield from [];\n}", "<?php\nfunction foo1()\n{\n    yield from [/*empty*/ ];\n}\nfunction bar()\n{\n    yield from [];\n    yield from array();\n}\nfunction baz()\n{\n    yield from [];\n     yield 1; yield 2;\n    yield from [];\n}"},
	})
}
