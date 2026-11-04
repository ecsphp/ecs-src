package rules

import "testing"

// Cases ported from PHP-CS-Fixer's NoUselessPrintfFixerTest
// (tests/Fixer/FunctionNotation); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestNoUselessPrintfPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, NoUselessPrintf{}, []aliasUpstreamCase{
		{"single-quoted", nil, `<?php printf('bar');`, `<?php print 'bar';`},
		{"double-quoted", nil, `<?php printf("bar\n");`, `<?php print "bar\n";`},
		{"case insensitive", nil, `<?php PrInTf("bar");`, `<?php print "bar";`},
		{"trailing comma", nil, `<?php printf("bar",);`, `<?php print "bar";`},
		{"leading NS separator", nil, `<?php \printf("bar");`, `<?php print "bar";`},
		{"skip not global function", nil,
			"<?php\nnamespace Foo {\n    function printf($arg) {}\n}\n\nnamespace Bar {\n    use function Foo\\printf;\n    printf('bar');\n}",
			"<?php\nnamespace Foo {\n    function printf($arg) {}\n}\n\nnamespace Bar {\n    use function Foo\\printf;\n    printf('bar');\n}"},
		{"skip function cases", nil,
			"<?php\nprintf();\nprintf('%s%s', 'foo', 'bar');\n\\Foo\\printf('bar');\nprintf(...$bar);",
			"<?php\nprintf();\nprintf('%s%s', 'foo', 'bar');\n\\Foo\\printf('bar');\nprintf(...$bar);"},
		{"skip class cases", nil,
			"<?php\nclass Foo {\n    public function printf($arg) {}\n}\n\nclass Bar {\n    public static function printf($arg) {}\n}\n\n(new Foo())->printf('bar');\nBar::printf('bar');",
			"<?php\nclass Foo {\n    public function printf($arg) {}\n}\n\nclass Bar {\n    public static function printf($arg) {}\n}\n\n(new Foo())->printf('bar');\nBar::printf('bar');"},
		{"skip first class callable", nil, `<?php return printf(...);`, `<?php return printf(...);`},
	})
}
