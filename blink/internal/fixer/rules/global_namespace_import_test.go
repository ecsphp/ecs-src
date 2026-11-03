package rules

import "testing"

func TestGlobalNamespaceImportPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPhpCsFixerCases(t, GlobalNamespaceImport{}, globalNamespaceImportUpstreamCases)
}

// Regression cases, checked against the PHP engine, with the options of a
// typical ECS config.
func TestGlobalNamespaceImportRegressions(t *testing.T) {
	t.Parallel()
	config := map[string]any{"import_constants": true, "import_functions": true}
	runPhpCsFixerCases(t, GlobalNamespaceImport{}, []phpCsFixerCase{
		{"a keyword used as a named argument is not a second namespace", "<?php\n\nnamespace App;\n\nuse function count;\n\n$n = count($a);\nfoo(namespace: 1, class: 2);\n", "<?php\n\nnamespace App;\n\nuse function count;\n\n$n = \\count($a);\nfoo(namespace: 1, class: 2);\n", config},
		{"docblock types, kinds order and a comment after the last import", "<?php\n\nnamespace App;\n\nuse Foo;\nuse ArrayObject;\nuse Closure;\nuse DateTimeImmutable;\nuse Stringable;\nuse function strlen;\nuse const PHP_EOL;\n // keep\n\n/**\n * @param DateTimeImmutable|\\Foo\\Bar $d\n * @return array<int, Closure(int): Stringable>\n */\nfunction f($d) { return new ArrayObject(PHP_EOL, strlen($d)); }\n", "<?php\n\nnamespace App;\n\nuse Foo; // keep\n\n/**\n * @param \\DateTimeImmutable|\\Foo\\Bar $d\n * @return array<int, \\Closure(int): \\Stringable>\n */\nfunction f($d) { return new \\ArrayObject(\\PHP_EOL, \\strlen($d)); }\n", config},
		{"a short name taken by another import stays qualified", "<?php\n\nnamespace App;\n\nuse Other\\Exception;\nuse function sprintf;\n\nthrow new \\Exception(sprintf(\"x\"));\n", "<?php\n\nnamespace App;\n\nuse Other\\Exception;\n\nthrow new \\Exception(\\sprintf(\"x\"));\n", config},
		{"no doc comment, slash or import: not a candidate", "<?php namespace App; echo 1;\n", "<?php namespace App; echo 1;\n", config},
	})
}

// Docblock types follow Preg's /u patterns, where \w, \h and \d are Unicode;
// checked against the PHP engine.
func TestGlobalNamespaceImportUnicodeDocTypes(t *testing.T) {
	t.Parallel()
	config := map[string]any{"import_classes": true}
	runPhpCsFixerCases(t, GlobalNamespaceImport{}, []phpCsFixerCase{
		{"a non-ASCII class name", "<?php\nnamespace App;\nuse Ünï;\n\n/** @param Ünï $u */\nfunction f($u) {}\n", "<?php\nnamespace App;\n\n/** @param \\Ünï $u */\nfunction f($u) {}\n", config},
		{"non-ASCII names in unions, generics and shapes; a Unicode digit cannot start a name", "<?php\nnamespace App;\nuse Bé;\nuse Çà;\nuse Ünï;\nuse Ŝ;\n\n/**\n * @param Ünï|Çà<Ŝ> $u\n * @return \\٣x\n * @var array{ä: Bé}\n */\nfunction f($u) {}\n", "<?php\nnamespace App;\n\n/**\n * @param \\Ünï|\\Çà<\\Ŝ> $u\n * @return \\٣x\n * @var array{ä: \\Bé}\n */\nfunction f($u) {}\n", config},
		{"a Unicode horizontal space around the type", "<?php\nnamespace App;\nuse Nb;\nuse Wide;\n\n/** @param\u2003Wide $u */\nfunction f($u) {}\n/** @param Nb\u2003$u */\nfunction g($u) {}\n", "<?php\nnamespace App;\n\n/** @param\u2003\\Wide $u */\nfunction f($u) {}\n/** @param \\Nb\u2003$u */\nfunction g($u) {}\n", config},
		// PHP-CS-Fixer fails on this file ("Cannot set empty content"), leaving it unchanged
		{"a lone backslash in a string type is never imported", "<?php\nnamespace App;\n\n/** @param \"\\ \" $u */\nfunction f($u) {}\nnew \\Foo();\n", "<?php\nnamespace App;\n\n/** @param \"\\ \" $u */\nfunction f($u) {}\nnew \\Foo();\n", config},
	})
}

// PHP-CS-Fixer runs the constant, function and class phases one after the
// other, each on the tokens the previous one changed.
func TestGlobalNamespaceImportPhasesInSequence(t *testing.T) {
	t.Parallel()
	config := map[string]any{"import_constants": true, "import_functions": true, "import_classes": true}
	runPhpCsFixerCases(t, GlobalNamespaceImport{}, []phpCsFixerCase{
		{"an attribute taken for a function call is not also imported as a class", "<?php\nnamespace App\\Sub;\nuse Arg;\nuse Attr1;\nuse function Attr2;\n\n#[Attr1, Attr2(Arg::class)]\nclass K {}\n", "<?php\nnamespace App\\Sub;\n\n#[\\Attr1, \\Attr2(\\Arg::class)]\nclass K {}\n", config},
	})
}
