package rules

import "testing"

// Ported from PHP-CS-Fixer v3.95.27 tests/Fixer/*/NativeFunctionInvocationFixerTest.php (provideFix*Cases).

var nativeFunctionInvocationUpstreamCases = []phpCsFixerCase{
	{"provideFixCases#0", "<?php\n\n\\is_string($foo);\n", "<?php\n\n\\is_string($foo);\n", nil},
	{"provideFixCases#1", "<?php\n\n\\is_string($foo);\n", "<?php\n\nis_string($foo);\n", nil},
	{"provideFixCases#2", "<?php\n\nclass Foo\n{\n    public function bar($foo)\n    {\n        return \\is_string($foo);\n    }\n}\n", "<?php\n\nclass Foo\n{\n    public function bar($foo)\n    {\n        return \\is_string($foo);\n    }\n}\n", nil},
	{"provideFixCases#3", "<?php\n\njson_encode($foo);\n\\strlen($foo);\n", "<?php\n\njson_encode($foo);\nstrlen($foo);\n", nil},
	{"provideFixCases#4", "<?php\n\nclass Foo\n{\n    public function bar($foo)\n    {\n        return \\IS_STRING($foo);\n    }\n}\n", "<?php\n\nclass Foo\n{\n    public function bar($foo)\n    {\n        return IS_STRING($foo);\n    }\n}\n", nil},
	{"fix multiple calls in single code", "<?php\n\njson_encode($foo);\n\\strlen($foo);\n\\strlen($foo);\n", "<?php\n\njson_encode($foo);\nstrlen($foo);\nstrlen($foo);\n", nil},
	{"provideFixCases#6", "<?php $name = \\get_class($foo, );", "<?php $name = get_class($foo, );", nil},
	{"provideFixCases#7", "<?php\n\nis_string($foo);\n", "<?php\n\nis_string($foo);\n", map[string]any{"exclude": []any{"is_string"}}},
	{"provideFixCases#8", "<?php\n\nclass Foo\n{\n    public function bar($foo)\n    {\n        return is_string($foo);\n    }\n}\n", "<?php\n\nclass Foo\n{\n    public function bar($foo)\n    {\n        return is_string($foo);\n    }\n}\n", map[string]any{"exclude": []any{"is_string"}}},
	{"provideFixCases#9", "<?php echo count([1]);", "<?php echo count([1]);", map[string]any{"scope": "namespaced"}},
	{"provideFixCases#10", "<?php\nnamespace space1 { ?>\n<?php echo \\count([2]) ?>\n<?php }namespace {echo count([1]);}\n", "<?php\nnamespace space1 { ?>\n<?php echo count([2]) ?>\n<?php }namespace {echo count([1]);}\n", map[string]any{"scope": "namespaced"}},
	{"provideFixCases#11", "<?php\nnamespace Bar {\n    echo \\strLEN(\"in 1\");\n}\n\nnamespace {\n    echo strlen(\"out 1\");\n}\n\nnamespace {\n    echo strlen(\"out 2\");\n}\n\nnamespace Bar{\n    echo \\strlen(\"in 2\");\n}\n\nnamespace {\n    echo strlen(\"out 3\");\n}\n", "<?php\nnamespace Bar {\n    echo strLEN(\"in 1\");\n}\n\nnamespace {\n    echo strlen(\"out 1\");\n}\n\nnamespace {\n    echo strlen(\"out 2\");\n}\n\nnamespace Bar{\n    echo strlen(\"in 2\");\n}\n\nnamespace {\n    echo strlen(\"out 3\");\n}\n", map[string]any{"scope": "namespaced"}},
	{"provideFixCases#12", "<?php\nnamespace space11 ?>\n\n    <?php\necho \\strlen(__NAMESPACE__);\nnamespace space2;\necho \\strlen(__NAMESPACE__);\n", "<?php\nnamespace space11 ?>\n\n    <?php\necho strlen(__NAMESPACE__);\nnamespace space2;\necho strlen(__NAMESPACE__);\n", map[string]any{"scope": "namespaced"}},
	{"provideFixCases#13", "<?php namespace PhpCsFixer\\Tests\\Fixer\\Casing;\\count([1]);", "<?php namespace PhpCsFixer\\Tests\\Fixer\\Casing;count([1]);", map[string]any{"scope": "namespaced"}},
	{"provideFixCases#14", "<?php\nnamespace Space12;\n\necho \\count([1]);\n\nnamespace Space2;\n\necho \\count([1]);\n?>\n", "<?php\nnamespace Space12;\n\necho count([1]);\n\nnamespace Space2;\n\necho count([1]);\n?>\n", map[string]any{"scope": "namespaced"}},
	{"provideFixCases#15", "<?php namespace {echo strlen(\"out 2\");}", "<?php namespace {echo strlen(\"out 2\");}", map[string]any{"scope": "namespaced"}},
	{"provideFixCases#16", "<?php\nnamespace space13 {\n    echo \\strlen(\"in 1\");\n}\n\nnamespace space2 {\n    echo \\strlen(\"in 2\");\n}\n\nnamespace { // global\n    echo strlen(\"global 1\");\n}\n", "<?php\nnamespace space13 {\n    echo strlen(\"in 1\");\n}\n\nnamespace space2 {\n    echo strlen(\"in 2\");\n}\n\nnamespace { // global\n    echo strlen(\"global 1\");\n}\n", map[string]any{"scope": "namespaced"}},
	{"provideFixCases#17", "<?php\nnamespace space1 {\n    echo \\count([1]);\n}\nnamespace {\n    echo \\count([1]);\n}\n", "<?php\nnamespace space1 {\n    echo count([1]);\n}\nnamespace {\n    echo count([1]);\n}\n", map[string]any{"scope": "all"}},
	{"include set + 1, exclude 1", "<?php\n                    echo \\count([1]);\n                    \\some_other($a, 3);\n                    echo strlen($a);\n                    not_me();\n                ", "<?php\n                    echo count([1]);\n                    some_other($a, 3);\n                    echo strlen($a);\n                    not_me();\n                ", map[string]any{"include": []any{"@internal", "some_other"}, "exclude": []any{"strlen"}}},
	{"include @all", "<?php\n                    echo \\count([1]);\n                    \\some_other($a, 3);\n                    echo \\strlen($a);\n                    \\me_as_well();\n                ", "<?php\n                    echo count([1]);\n                    some_other($a, 3);\n                    echo strlen($a);\n                    me_as_well();\n                ", map[string]any{"include": []any{"@all"}}},
	{"include @compiler_optimized", "<?php\n                    // do not fix\n                    $a = strrev($a);\n                    $a .= str_repeat($a, 4);\n                    $b = already_prefixed_function();\n                    // fix\n                    $c = \\get_class($d);\n                    $e = \\intval($f);\n                ", "<?php\n                    // do not fix\n                    $a = strrev($a);\n                    $a .= str_repeat($a, 4);\n                    $b = \\already_prefixed_function();\n                    // fix\n                    $c = get_class($d);\n                    $e = intval($f);\n                ", map[string]any{"include": []any{"@compiler_optimized"}}},
	{"provideFixCases#21", "<?php class Foo {\n                        public function & strlen($name) {\n                        }\n                    }\n                ", "<?php class Foo {\n                        public function & strlen($name) {\n                        }\n                    }\n                ", nil},
	{"scope namespaced and strict enabled", "<?php\n                    $a = not_compiler_optimized_function();\n                    $b = intval($c);\n                ", "<?php\n                    $a = \\not_compiler_optimized_function();\n                    $b = \\intval($c);\n                ", map[string]any{"scope": "namespaced", "strict": true}},
	{"provideFixCases#23", "<?php\n                    use function foo\\json_decode;\n                    json_decode($base);\n                ", "<?php\n                    use function foo\\json_decode;\n                    json_decode($base);\n                ", map[string]any{"include": []any{"@all"}}},
	{"provideFixCases#24", "<?php return [\\set() => 42];", "<?php return [set() => 42];", map[string]any{"include": []any{"@all"}}},
	{"attribute and strict", "<?php\n                #[\\Attribute(\\Attribute::TARGET_CLASS)]\n                class Foo {}\n            ", "<?php\n                #[\\Attribute(\\Attribute::TARGET_CLASS)]\n                class Foo {}\n            ", map[string]any{"strict": true}},
	{"null safe operator", "<?php $x?->count();", "<?php $x?->count();", nil},
	{"multiple function-calls-like in attribute", "<?php\n                #[Foo(), Bar(), Baz()]\n                class Foo {}\n            ", "<?php\n                #[Foo(), Bar(), Baz()]\n                class Foo {}\n            ", map[string]any{"include": []any{"@all"}}},
	{"property hooks", "<?php\nclass Foo\n{\n    public string $bar = '' {\n        get => $this->bar;\n        set(string $bar) => \\strtolower($bar);\n    }\n    public string $baz = '' {\n        get => $this->baz;\n        set(string $baz) { $this->baz = \\strtoupper($baz); }\n    }\n}", "<?php\nclass Foo\n{\n    public string $bar = '' {\n        get => $this->bar;\n        set(string $bar) => strtolower($bar);\n    }\n    public string $baz = '' {\n        get => $this->baz;\n        set(string $baz) { $this->baz = strtoupper($baz); }\n    }\n}", map[string]any{"include": []any{"@all"}}},
}

func TestNativeFunctionInvocationPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPhpCsFixerCases(t, NativeFunctionInvocation{}, nativeFunctionInvocationUpstreamCases)
}

// Regression cases, checked against the PHP engine, with the options of a
// typical ECS config.
func TestNativeFunctionInvocationRegressions(t *testing.T) {
	t.Parallel()
	config := map[string]any{"scope": "namespaced"}
	runPhpCsFixerCases(t, NativeFunctionInvocation{}, []phpCsFixerCase{
		{"imported function still gets a leading slash", "<?php\n\nnamespace App;\n\nuse function count;\n\n$n = \\count($items) + \\strlen($name);\n", "<?php\n\nnamespace App;\n\nuse function count;\n\n$n = count($items) + strlen($name);\n", config},
		{"global namespace loses slashes with scope namespaced", "<?php\n\n$n = count($items) + array_map($fn, $items);\n", "<?php\n\n$n = \\count($items) + \\array_map($fn, $items);\n", config},
		{"function declared in the namespace shadows the native one", "<?php\n\nnamespace App;\n\nfunction strlen(string $s): int { return 0; }\n\n$n = strlen($name) + \\count($items);\n", "<?php\n\nnamespace App;\n\nfunction strlen(string $s): int { return 0; }\n\n$n = strlen($name) + count($items);\n", config},
		{"first-class callable, exit, die, methods and new are left alone", "<?php\n\nnamespace App;\n\n$f = strlen(...);\nexit(\\count($a));\ndie();\n$o->count();\nFoo::count();\nnew count();\n", "<?php\n\nnamespace App;\n\n$f = strlen(...);\nexit(count($a));\ndie();\n$o->count();\nFoo::count();\nnew count();\n", config},
		{"a keyword used as a named argument is not a namespace", "<?php\n\nnamespace App;\n\nfoo(namespace: 1);\n$n = \\count($a);\n", "<?php\n\nnamespace App;\n\nfoo(namespace: 1);\n$n = count($a);\n", config},
	})
}
