package rules

import "testing"

// Agreement cases: inputs where blink's ClassKeyword and PHP-CS-Fixer 3.95
// produce the same output. PHP-CS-Fixer only rewrites a string whose class is
// autoloadable; these use classes that exist in its own runtime, plus strings
// that are not class names, so both tools agree. See TestClassKeywordStaticPort
// for the one behavior that cannot match without a runtime.
func TestClassKeywordPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, ClassKeyword{}, []aliasUpstreamCase{
		{"single quoted fqcn", nil, "<?php\n$a = 'PhpCsFixer\\Tokenizer\\Tokens';\n", "<?php\n$a = \\PhpCsFixer\\Tokenizer\\Tokens::class;\n"},
		{"double quoted fqcn", nil, "<?php\n$b = \"\\\\PhpCsFixer\\\\Tokenizer\\\\Tokens\";\n", "<?php\n$b = \\PhpCsFixer\\Tokenizer\\Tokens::class;\n"},
		{"not a class name", nil, "<?php\n$d = 'hello world';\n", "<?php\n$d = 'hello world';\n"},
		{"empty string", nil, "<?php\n$f = '';\n", "<?php\n$f = '';\n"},
		{"numeric string", nil, "<?php\n$g = '123';\n", "<?php\n$g = '123';\n"},
		{"bare word is not namespaced", nil, "<?php\n$h = 'DateTime';\n", "<?php\n$h = 'DateTime';\n"},
	})
}

// TestClassKeywordStaticPort documents the one deviation from PHP-CS-Fixer: with
// no runtime to resolve classes, blink rewrites any valid namespaced name, so a
// name PHP-CS-Fixer would leave (its class does not exist) is still rewritten.
func TestClassKeywordStaticPort(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, ClassKeyword{}, "<?php\n$c = 'This\\Does\\Not\\Exist';\n")
	want := "<?php\n$c = \\This\\Does\\Not\\Exist::class;\n"
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}
