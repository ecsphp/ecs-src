package rules

import (
	"fmt"
	"strings"
	"testing"
)

// Cases ported from PHP-CS-Fixer's tests/Fixer/FunctionNotation/VoidReturnFixerTest.php (v3.95.27).
func TestVoidReturnUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, VoidReturn{}, voidReturnUpstreamCases)
}

var voidReturnUpstreamCases = []upstreamCase{
	{"provideFixCases#0", nil, "<?php class Test { public function __construct() {} }", ""},
	{"provideFixCases#1", nil, "<?php class Test { public function __destruct() {} }", ""},
	{"provideFixCases#2", nil, "<?php class Test { public function __clone() {} }", ""},
	{"provideFixCases#3", nil, "<?php function foo($param) { return $param; }", ""},
	{"provideFixCases#4", nil, "<?php function foo($param) { return null; }", ""},
	{"provideFixCases#5", nil, "<?php function foo($param) { yield; }", ""},
	{"provideFixCases#6", nil, "<?php function foo($param) { yield $param; }", ""},
	{"provideFixCases#7", nil, "<?php function foo($param) { yield from test(); }", ""},
	{"provideFixCases#8", nil, "<?php function foo($param): Void {}", ""},
	{"provideFixCases#9", nil, "<?php interface Test { public function foo($param); }", ""},
	{"provideFixCases#10", nil, "<?php function foo($param) { return function($a) use ($param): string {}; }", ""},
	{"provideFixCases#11", nil, "<?php abstract class Test { abstract public function foo($param); }", ""},
	{"provideFixCases#12", nil, "<?php use Foo\\ { function Bar }; function test() { return Bar(); }", ""},
	{"provideFixCases#13", nil, "<?php\n                /**\n                 * @return array\n                 */\n                function foo($param) {}\n            ", ""},
	{"provideFixCases#14", nil, "<?php\n                interface Test {\n                    /**\n                     * @return array\n                     */\n                    public function foo($param);\n                }\n            ", ""},
	{"provideFixCases#15", nil, "<?php function foo($param): void { return; }", "<?php function foo($param) { return; }"},
	{"provideFixCases#16", nil, "<?php function foo($param): void {}", "<?php function foo($param) {}"},
	{"provideFixCases#17", nil, "<?php class Test { public function foo($param): void { return; } }", "<?php class Test { public function foo($param) { return; } }"},
	{"provideFixCases#18", nil, "<?php class Test { public function foo($param): void {} }", "<?php class Test { public function foo($param) {} }"},
	{"provideFixCases#19", nil, "<?php trait Test { public function foo($param): void { return; } }", "<?php trait Test { public function foo($param) { return; } }"},
	{"provideFixCases#20", nil, "<?php trait Test { public function foo($param): void {} }", "<?php trait Test { public function foo($param) {} }"},
	{"provideFixCases#21", nil, "<?php $arr = []; usort($arr, function ($a, $b): void {});", "<?php $arr = []; usort($arr, function ($a, $b) {});"},
	{"provideFixCases#22", nil, "<?php $arr = []; $param = 1; usort($arr, function ($a, $b) use ($param): void {});", "<?php $arr = []; $param = 1; usort($arr, function ($a, $b) use ($param) {});"},
	{"provideFixCases#23", nil, "<?php function foo($param) { return function($a) use ($param): void {}; }", "<?php function foo($param) { return function($a) use ($param) {}; }"},
	{"provideFixCases#24", nil, "<?php function foo($param): void { $arr = []; usort($arr, function ($a, $b) use ($param): void {}); }", "<?php function foo($param) { $arr = []; usort($arr, function ($a, $b) use ($param) {}); }"},
	{"provideFixCases#25", nil, "<?php function foo() { $arr = []; return usort($arr, new class { public function __invoke($a, $b): void {} }); }", "<?php function foo() { $arr = []; return usort($arr, new class { public function __invoke($a, $b) {} }); }"},
	{"provideFixCases#26", nil, "<?php function foo(): void { $arr = []; usort($arr, new class { public function __invoke($a, $b): void {} }); }", "<?php function foo() { $arr = []; usort($arr, new class { public function __invoke($a, $b) {} }); }"},
	{"provideFixCases#27", nil, "<?php\n                function foo(): void {\n                    $a = function (): void {};\n                }", "<?php\n                function foo() {\n                    $a = function () {};\n                }"},
	{"provideFixCases#28", nil, "<?php\n                function foo(): void {\n                    (function (): void {\n                        return;\n                    })();\n                }", "<?php\n                function foo() {\n                    (function () {\n                        return;\n                    })();\n                }"},
	{"provideFixCases#29", nil, "<?php\n                function foo(): void {\n                    (function () {\n                        return 1;\n                    })();\n                }", "<?php\n                function foo() {\n                    (function () {\n                        return 1;\n                    })();\n                }"},
	{"provideFixCases#30", nil, "<?php\n                function foo(): void {\n                    $b = new class {\n                        public function b1(): void {}\n                        public function b2() { return 2; }\n                    };\n                }", "<?php\n                function foo() {\n                    $b = new class {\n                        public function b1() {}\n                        public function b2() { return 2; }\n                    };\n                }"},
	{"provideFixCases#31", nil, "<?php\n                /**\n                 * @return void\n                 */\n                function foo($param): void {}", "<?php\n                /**\n                 * @return void\n                 */\n                function foo($param) {}"},
	{"provideFixCases#32", nil, "<?php\n                interface Test {\n                    /**\n                     * @return void\n                     */\n                    public function foo($param): void;\n                }", "<?php\n                interface Test {\n                    /**\n                     * @return void\n                     */\n                    public function foo($param);\n                }"},
	{"provideFixCases#33", nil, "<?php\n                abstract class Test {\n                    /**\n                     * @return void\n                     */\n                    abstract protected function foo($param): void;\n                }", "<?php\n                abstract class Test {\n                    /**\n                     * @return void\n                     */\n                    abstract protected function foo($param);\n                }"},
	{"provideFixCases#34", nil, "<?php fn($a) => null;", ""},
	{"provideFixCases#35", nil, "<?php fn($a) => 1;", ""},
	{"provideFixCases#36", nil, "<?php fn($a) => var_dump($a);", ""},
	{"provideFixCases#37", nil, "<?php\n\nclass Foo\n{\n    public function hello(): void\n    {\n        $hello = function($a): void { echo 'Hello '.$a; };\n        echo $hello('World');\n    }\n}", "<?php\n\nclass Foo\n{\n    public function hello()\n    {\n        $hello = function($a) { echo 'Hello '.$a; };\n        echo $hello('World');\n    }\n}"},
	{"provideFixCases#38", map[string]any{"fix_lambda": false}, "<?php\n\nclass Foo\n{\n    public function hello(): void\n    {\n        $hello = function($a) { echo 'Hello '.$a; };\n        echo $hello('World');\n    }\n}", "<?php\n\nclass Foo\n{\n    public function hello()\n    {\n        $hello = function($a) { echo 'Hello '.$a; };\n        echo $hello('World');\n    }\n}"},
	{"provideFix80Cases#0", nil, "<?php\n\n            class Foo\n            {\n                /**\n                 * @return int|false\n                 */\n                #[\\ReturnTypeWillChange]\n                public function test() {}\n            }\n            ", ""},
	{"provideFix80Cases#1", nil, "<?php\n\n            /**\n             * @return void\n             */\n            #[\\Deprecated]\n            function test(): void {};\n            ", "<?php\n\n            /**\n             * @return void\n             */\n            #[\\Deprecated]\n            function test() {};\n            "},
}

// PHP-CS-Fixer generates one case per magic method: ": void" is added except
// where it is a syntax error or the method returns a value.
func TestVoidReturnUpstreamMagicMethods(t *testing.T) {
	t.Parallel()
	excluded := map[string]bool{
		"__clone": true, "__construct": true, "__debugInfo": true, "__destruct": true, "__isset": true,
		"__serialize": true, "__set_state": true, "__sleep": true, "__toString": true,
	}
	definitions := []struct {
		name      string
		arguments int
		static    bool
	}{
		{"__construct", 0, false}, {"__destruct", 0, false}, {"__call", 2, false}, {"__callStatic", 2, true},
		{"__get", 1, false}, {"__set", 2, false}, {"__isset", 1, false}, {"__unset", 1, false},
		{"__sleep", 0, false}, {"__wakeup", 0, false}, {"__serialize", 0, false}, {"__unserialize", 1, false},
		{"__toString", 0, false}, {"__invoke", 0, false}, {"__set_state", 1, true}, {"__clone", 0, false},
		{"__debugInfo", 0, false},
	}
	var cases []upstreamCase
	for _, d := range definitions {
		static := ""
		if d.static {
			static = " static"
		}
		var args []string
		for n := range d.arguments {
			args = append(args, fmt.Sprintf("$x%d", n))
		}
		code := fmt.Sprintf("<?php class Test { public%s function %s(%s)%%s {} }", static, d.name, strings.Join(args, ","))
		input := fmt.Sprintf(code, "")
		expected := input
		if !excluded[d.name] {
			expected = fmt.Sprintf(code, ": void")
		}
		c := upstreamCase{"magic method " + d.name, nil, expected, ""}
		if expected != input {
			c.input = input
		}
		cases = append(cases, c)
	}
	runUpstreamCases(t, VoidReturn{}, cases)
}
