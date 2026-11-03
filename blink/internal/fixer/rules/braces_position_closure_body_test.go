package rules

import "testing"

// TestBracesPositionClosureBodyOnOwnLines pins how BracesPosition puts the
// body of an anonymous function on lines of its own, with outputs taken from
// PHP-CS-Fixer's BracesPositionFixer: both the body and the "}" go at the
// indentation of the line holding the "{", not at the brace nesting depth
// (StatementIndentation indents the body afterwards). With the default
// allow_single_line_anonymous_functions, only a body that already spans lines
// is expanded.
func TestBracesPositionClosureBodyOnOwnLines(t *testing.T) {
	t.Parallel()
	configs := map[string]fixerRule{
		"def": BracesPosition{},
		"off": BracesPosition{}.WithConfig(map[string]any{"allow_single_line_anonymous_functions": false}).(fixerRule),
		"offnl": BracesPosition{}.WithConfig(map[string]any{
			"allow_single_line_anonymous_functions": false,
			"anonymous_functions_opening_brace":     "next_line_unless_newline_at_signature_end",
		}).(fixerRule),
	}
	cases := []struct{ name, config, in, want string }{
		{"off/c1", "off", "<?php\n\nfunction f()\n{\n    foo(function () { return 1; });\n}\n", "<?php\n\nfunction f()\n{\n    foo(function () {\n    return 1;\n    });\n}\n"},
		{"off/c2", "off", "<?php\n\n$x = [\n    \"a\" => function ($v) { return $v; },\n    \"b\" => 2,\n];\n", "<?php\n\n$x = [\n    \"a\" => function ($v) {\n    return $v;\n    },\n    \"b\" => 2,\n];\n"},
		{"off/c4", "off", "<?php\n\nclass A\n{\n    public function b()\n    {\n        return array_map(function ($v) { return $v * 2; }, $this->items);\n    }\n}\n", "<?php\n\nclass A\n{\n    public function b()\n    {\n        return array_map(function ($v) {\n        return $v * 2;\n        }, $this->items);\n    }\n}\n"},
		{"off/c5", "off", "<?php\n\n$f = function () { return function () { return 1; }; };\n", "<?php\n\n$f = function () {\nreturn function () {\nreturn 1;\n};\n};\n"},
		{"off/c6", "off", "<?php\n\n$f = function () {};\n$g = function () { /* c */ return 1; };\n$h = function () { // c\n    return 1; };\n", "<?php\n\n$f = function () {\n};\n$g = function () { /* c */ return 1;\n};\n$h = function () { // c\n    return 1;\n};\n"},
		{"off/c7", "off", "<?php\n\nrun(\n    1,\n    function () use ($a) { return $a; }\n);\n$m = fn () => 1;\n", "<?php\n\nrun(\n    1,\n    function () use ($a) {\n    return $a;\n    }\n);\n$m = fn () => 1;\n"},
		{"def/e1", "def", "<?php\n\n$f = array_map(function ($v) { return [\n    1,\n]; }, $l);\n$g = function () { return 1; };\n$h = function () { foo(function () { return 2; },\n    3); };\n", "<?php\n\n$f = array_map(function ($v) {\nreturn [\n    1,\n];\n}, $l);\n$g = function () { return 1; };\n$h = function () {\nfoo(function () { return 2; },\n    3);\n};\n"},
		{"def/e2", "def", "<?php\n\nrun(new class {\n    public function go()\n    {\n        return [\n            \"k\" => function () { return 1; },\n        ];\n    }\n});\n", "<?php\n\nrun(new class {\n    public function go()\n    {\n        return [\n            \"k\" => function () { return 1; },\n        ];\n    }\n});\n"},
		{"def/e3", "def", "<?php\n$f = function () { ?>x<?php };\n$g = function () {/* c */\n    return 1; };\n$h = function () { /** d */ return 1; };\n", "<?php\n$f = function () { ?>x<?php };\n$g = function () {/* c */\n    return 1;\n};\n$h = function () { /** d */ return 1; };\n"},
		{"off/e1", "off", "<?php\n\n$f = array_map(function ($v) { return [\n    1,\n]; }, $l);\n$g = function () { return 1; };\n$h = function () { foo(function () { return 2; },\n    3); };\n", "<?php\n\n$f = array_map(function ($v) {\nreturn [\n    1,\n];\n}, $l);\n$g = function () {\nreturn 1;\n};\n$h = function () {\nfoo(function () {\nreturn 2;\n},\n    3);\n};\n"},
		{"off/e2", "off", "<?php\n\nrun(new class {\n    public function go()\n    {\n        return [\n            \"k\" => function () { return 1; },\n        ];\n    }\n});\n", "<?php\n\nrun(new class {\n    public function go()\n    {\n        return [\n            \"k\" => function () {\n            return 1;\n            },\n        ];\n    }\n});\n"},
		{"off/e3", "off", "<?php\n$f = function () { ?>x<?php };\n$g = function () {/* c */\n    return 1; };\n$h = function () { /** d */ return 1; };\n", "<?php\n$f = function () { ?>x<?php };\n$g = function () {/* c */\n    return 1;\n};\n$h = function () { /** d */ return 1;\n};\n"},
		{"offnl/e1", "offnl", "<?php\n\n$f = array_map(function ($v) { return [\n    1,\n]; }, $l);\n$g = function () { return 1; };\n$h = function () { foo(function () { return 2; },\n    3); };\n", "<?php\n\n$f = array_map(function ($v)\n{\nreturn [\n    1,\n];\n}, $l);\n$g = function ()\n{\nreturn 1;\n};\n$h = function ()\n{\nfoo(function ()\n{\nreturn 2;\n},\n    3);\n};\n"},
		{"offnl/e2", "offnl", "<?php\n\nrun(new class {\n    public function go()\n    {\n        return [\n            \"k\" => function () { return 1; },\n        ];\n    }\n});\n", "<?php\n\nrun(new class {\n    public function go()\n    {\n        return [\n            \"k\" => function ()\n            {\n            return 1;\n            },\n        ];\n    }\n});\n"},
		{"offnl/e3", "offnl", "<?php\n$f = function () { ?>x<?php };\n$g = function () {/* c */\n    return 1; };\n$h = function () { /** d */ return 1; };\n", "<?php\n$f = function ()\n{ ?>x<?php };\n$g = function ()\n{/* c */\n    return 1;\n};\n$h = function ()\n{ /** d */ return 1;\n};\n"},
	}
	for _, tc := range cases {
		got, _ := apply(t, configs[tc.config], tc.in)
		if got != tc.want {
			t.Errorf("%s:\n got: %q\nwant: %q", tc.name, got, tc.want)
		}
	}
}
