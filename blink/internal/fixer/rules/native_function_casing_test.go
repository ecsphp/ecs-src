package rules

import "testing"

// NativeFunctionCasing lowercases what PHP-CS-Fixer's
// FunctionsAnalyzer::isGlobalFunctionCall reports as a global call.
func TestNativeFunctionCasingGlobalCalls(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		{"fully qualified call", "<?php\nnamespace App;\n$a = \\STRLEN('a');\n", "<?php\nnamespace App;\n$a = \\strlen('a');\n"},
		{"fully qualified name in an attribute", "<?php\n#[Foo, \\Count(3)]\n#[Count(3), Bar(\\COUNT($x))]\nclass A {}\n", "<?php\n#[Foo, \\count(3)]\n#[Count(3), Bar(\\count($x))]\nclass A {}\n"},
		{"namespaced and relative names", "<?php\n$b = Foo\\STRLEN('a');\n$c = namespace\\STRLEN('a');\n", "<?php\n$b = Foo\\STRLEN('a');\n$c = namespace\\STRLEN('a');\n"},
		{"a method returning by reference", "<?php\nclass A {\n    public function &COUNT() {}\n}\n", "<?php\nclass A {\n    public function &COUNT() {}\n}\n"},
		{"fully qualified first-class callable", "<?php\n$f = \\STRLEN(...);\n", "<?php\n$f = \\STRLEN(...);\n"},
	}
	for _, c := range cases {
		got, _ := apply(t, NativeFunctionCasing{}, c.src)
		if got != c.want {
			t.Errorf("%s:\n got: %q\nwant: %q", c.name, got, c.want)
		}
	}
}
