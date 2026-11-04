package rules

import "testing"

// Cases ported from PHP-CS-Fixer's MbStrFunctionsFixerTest (base, version-
// independent mappings only).
func TestMbStrFunctionsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, MbStrFunctions{}, []aliasUpstreamCase{
		{"string literal", nil, `<?php $x = "strlen";`, `<?php $x = "strlen";`},
		{"static call", nil, `<?php $x = Foo::strlen("bar");`, `<?php $x = Foo::strlen("bar");`},
		{"new", nil, `<?php $x = new strlen("bar");`, `<?php $x = new strlen("bar");`},
		{"new namespaced", nil, `<?php $x = new \strlen("bar");`, `<?php $x = new \strlen("bar");`},
		{"new sub-namespace", nil, `<?php $x = new Foo\strlen("bar");`, `<?php $x = new Foo\strlen("bar");`},
		{"namespaced name", nil, `<?php $x = Foo\strlen("bar");`, `<?php $x = Foo\strlen("bar");`},
		{"static reference", nil, `<?php $x = strlen::call("bar");`, `<?php $x = strlen::call("bar");`},
		{"method call", nil, `<?php $x = $foo->strlen("bar");`, `<?php $x = $foo->strlen("bar");`},
		{"zero args", nil, `<?php $x = strlen();`, `<?php $x = strlen();`},
		{"too many args", nil, `<?php $x = strlen($a, $b);`, `<?php $x = strlen($a, $b);`},
		{"strlen", nil, `<?php $x = strlen("bar");`, `<?php $x = mb_strlen("bar");`},
		{"leading backslash", nil, `<?php $x = \strlen("bar");`, `<?php $x = \mb_strlen("bar");`},
		{"nested", nil, `<?php $x = strtolower(strstr("bar", "a"));`, `<?php $x = mb_strtolower(mb_strstr("bar", "a"));`},
		{"nested with spaces", nil, `<?php $x = strtolower( \strstr ("bar", "a"));`, `<?php $x = mb_strtolower( \mb_strstr ("bar", "a"));`},
		{"substr three args", nil, `<?php $x = substr("bar", 2, 1);`, `<?php $x = mb_substr("bar", 2, 1);`},
		{"interface declarations", nil,
			"<?php\ninterface Test\n{\n    public function &strlen($a);\n    public function strtolower($a);\n}",
			"<?php\ninterface Test\n{\n    public function &strlen($a);\n    public function strtolower($a);\n}"},
		{"str_split", nil, `<?php $a = str_split($a);`, `<?php $a = mb_str_split($a);`},
		{"use function imports", nil,
			"<?php\nnamespace Foo;\nuse function Bar\\strlen;\nuse function strtolower;\nuse function strtoupper;\nuse function \\str_split;\nreturn strlen($x) > 10 ? strtolower($x) : strtoupper($x);",
			"<?php\nnamespace Foo;\nuse function Bar\\strlen;\nuse function mb_strtolower;\nuse function mb_strtoupper;\nuse function \\mb_str_split;\nreturn strlen($x) > 10 ? mb_strtolower($x) : mb_strtoupper($x);"},
	})
}
