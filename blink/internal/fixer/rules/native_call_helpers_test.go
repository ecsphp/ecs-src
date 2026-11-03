package rules

import (
	"strings"
	"testing"

	"blink/internal/lexer"
	"blink/internal/tokens"
)

// nativeCallCase is one upstream fixer test case: want is the expected output
// and src the input, or "" when the input must be left as is.
type nativeCallCase struct{ name, want, src string }

// runNativeCallCases applies f to every case, as PHP-CS-Fixer's doTest does.
func runNativeCallCases(t *testing.T, f fixerRule, cases []nativeCallCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src, changed := c.src, true
			if src == "" {
				src, changed = c.want, false
			}
			assertFix(t, f, src, c.want, changed)
		})
	}
}

// globalCallNames lists, in order, the names aliasIsGlobalFunctionCall
// reports as global function calls in src.
func globalCallNames(src string) []string {
	s := tokens.New(lexer.Lex(src))
	var names []string
	for i := range s.Len() {
		if aliasIsGlobalFunctionCall(s, i) {
			names = append(names, s.At(i).Value)
		}
	}
	return names
}

// Cases from PHP-CS-Fixer's FunctionsAnalyzerTest::provideIsGlobalFunctionCallCases
// that the native-function fixers rely on, asserting the names found, plus
// blink regressions.
func TestNativeCallIsGlobalFunctionCall(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{"constant", "<?php CONSTANT;", nil},
		{"call", "<?php foo();", []string{"foo"}},
		{"call with argument", `<?php foo("bar");`, []string{"foo"}},
		{"fully qualified", `<?php \foo("bar");`, []string{"foo"}},
		{"namespaced", `<?php foo\bar("baz");`, nil},
		{"static", `<?php foo::bar("baz");`, nil},
		{"method", `<?php $foo->bar("baz");`, nil},
		{"new", `<?php new bar("baz");`, nil},
		{"declaration", "<?php function foo() {}", nil},
		{"function with ref. return", "<?php function & foo() {}", nil},
		{"namespace operator", `<?php namespace\foo("bar");`, nil},
		{"import in another namespace", "<?php\nnamespace A {\n    use function A;\n}\nnamespace B {\n    use function D;\n    A();\n}\n", []string{"A"}},
		{"declared in global code", "<?php\nfunction A(){}\nA();\n", []string{"A"}},
		{"declared in global code, other case", "<?php\nfunction A(){}\na();\n", []string{"a"}},
		{"declared in braced global namespace", "<?php\nnamespace {\n    function A(){}\n    A();\n}\n", []string{"A"}},
		{"declared in braced namespace", "<?php\nnamespace Z {\n    function A(){}\n    A();\n}\n", nil},
		{"declared in namespace", "<?php\nnamespace Z;\n\nfunction A(){}\nA();\n", nil},
		{"function signature ref. return, calls itself", "<?php\nfunction & A(){}\nA();\n", []string{"A"}},
		{"method of the same name", "<?php\nclass Foo\n{\n    public function A(){}\n}\nA();\n", []string{"A"}},
		{"declared in another namespace", "<?php\nnamespace A {\n    function A(){}\n}\nnamespace B {\n    A();\n}\n", []string{"A"}},
		{"imported from a namespace", "<?php\nuse function X\\a;\nA();\n", nil},
		{"class import", "<?php\nuse A;\nA();\n", []string{"A"}},
		{"const import", "<?php\nuse const A;\nA();\n", []string{"A"}},
		{"other function imported", "<?php\nuse function A;\nstr_repeat($a, $b);\n", []string{"str_repeat"}},
		{"closure after a declaration", "<?php\nnamespace {\n    function A(){}\n    A();\n    $b = function(){};\n}\n", []string{"A"}},
		{"repeated", "<?php implode($a);implode($a);implode($a);implode($a);implode($a);implode($a);", []string{"implode", "implode", "implode", "implode", "implode", "implode"}},
		{"methods of anonymous classes", "<?php\n$z = new class(\n    new class(){ private function A(){} }\n){\n    public function A() {}\n};\n\nA();\n", []string{"A"}},
		{"arrow function", "<?php $foo = fn() => false;", nil},
		{"inside a method", `<?php foo("bar"); class A { function Foo(){ foo(); } }`, []string{"foo", "foo"}},
		{"functions that can be confused with a property hook", "<?php\n$array = [\n    1 => 2,\n    set() => 3,\n    $foo . set() => 3,\n];\n$x = set(set(), set());\nset();\nif (true) { set(); }\nset();", []string{"set", "set", "set", "set", "set", "set", "set", "set"}},

		// blink regressions
		{"first-class callable", "<?php $f = strlen(...);", nil},
		{"global import", "<?php\nnamespace A;\nuse function \\dirname;\ndirname($x);\n", []string{"dirname"}},
		{"global import without leading slash", "<?php\nnamespace A;\nuse function dirname;\ndirname($x);\n", []string{"dirname"}},
		{"aliased import", "<?php\nnamespace A;\nuse function B\\other as dirname;\ndirname($x);\n", nil},
		{"group import is not considered", "<?php\nnamespace A;\nuse function B\\{dirname};\ndirname($x);\n", []string{"dirname"}},
		{"attribute name", "<?php\n#[Foo(1)]\nfunction a() {}\n", nil},
		{"return by reference closure", "<?php $f = function &() { return bar(); };", []string{"bar"}},
		{"reference assignment", "<?php $a = &foo();", []string{"foo"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := globalCallNames(c.src)
			if len(got) != len(c.want) {
				t.Fatalf("got %q, want %q", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("got %q, want %q", got, c.want)
				}
			}
		})
	}
}

func TestNativeCallArguments(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src  string
		want []string
	}{
		{"<?php f();", nil},
		{"<?php f( );", nil},
		{"<?php f($a);", []string{"$a"}},
		{"<?php f($a, [1, 2], g(3, 4));", []string{"$a", " [1, 2]", " g(3, 4)"}},
		{"<?php f($a, $b,);", []string{"$a", " $b"}},
		{"<?php f(function () { return [1, 2]; }, $c);", []string{"function () { return [1, 2]; }", " $c"}},
		{"<?php f(#[A(1, 2)] fn () => 1, 2);", []string{"#[A(1, 2)] fn () => 1", " 2"}},
	}
	for _, c := range cases {
		s := tokens.New(lexer.Lex(c.src))
		open := -1
		for i := range s.Len() {
			if isPunctVal(s, i, "(") {
				open = i
				break
			}
		}
		var got []string
		for _, a := range aliasArguments(s, open, s.MatchForward(open)) {
			var v strings.Builder
			for i := a[0]; i <= a[1]; i++ {
				v.WriteString(s.At(i).Value)
			}
			got = append(got, v.String())
		}
		if len(got) != len(c.want) {
			t.Fatalf("%s: got %q, want %q", c.src, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%s: got %q, want %q", c.src, got, c.want)
			}
		}
	}
}

func TestNativeCallConstantStringAt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src  string
		want bool
	}{
		{"<?php 'a';", true},
		{`<?php "a";`, true},
		{`<?php "a\$b";`, true},
		{`<?php "a $ b";`, true},
		{`<?php "a $b";`, false},
		{`<?php "a ${b}";`, false},
		{`<?php "a {$b}";`, false},
		{"<?php b'a';", true},
		{`<?php B"a";`, true},
		{"<?php <<<'X'\na\nX;\n", false},
		{"<?php `ls`;", false},
	}
	for _, c := range cases {
		s := tokens.New(lexer.Lex(c.src))
		first := nextMeaningfulIndex(s, 0)
		if _, got := constantStringAt(s, first); got != c.want {
			t.Fatalf("%s: got %v, want %v", c.src, got, c.want)
		}
	}
}
