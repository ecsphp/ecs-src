package rules

import "testing"

func TestNoUnusedImportsRemovesUnused(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoUnusedImports{}, "<?php\nnamespace App;\nuse Vendor\\Foo;\nuse Vendor\\Bar;\n\nnew Foo();\n")
	if want := "<?php\nnamespace App;\nuse Vendor\\Foo;\n\nnew Foo();\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

// A use that is used inside its own brace-delimited namespace block must be kept,
// even when a different block defines a class of the same short name.
func TestNoUnusedImportsBraceNamespaceUsed(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nnamespace Foo\\Bar {\n    use Psr\\Log\\LoggerInterface;\n\n    class Baz\n    {\n        public function __construct(LoggerInterface $logger) {}\n    }\n}\n\nnamespace {\n    use Foo\\Bar\\Baz;\n\n    function run(): void {\n        Baz::go();\n    }\n}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); changed || got != src {
		t.Fatalf("brace-namespace import must be kept; changed=%v got=%q", changed, got)
	}
}

// An unused import inside a brace-delimited global namespace is still removed.
func TestNoUnusedImportsBraceNamespaceUnused(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nnamespace {\n    use Foo\\Bar\\Baz;\n\n    function run(): void {\n    }\n}\n"
	want := "<?php\n\nnamespace {\n\n    function run(): void {\n    }\n}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); !changed || got != want {
		t.Fatalf("unused brace-namespace import must be removed; changed=%v got=%q want=%q", changed, got, want)
	}
}

// An import used only as an attribute name ("#[AsCommand(...)]") is a class usage,
// even though a "(" follows the name, so it must be kept.
func TestNoUnusedImportsAttributeUsage(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nnamespace App;\n\nuse Symfony\\Component\\Console\\Attribute\\AsCommand;\n\n#[AsCommand(name: 'app:run')]\nclass RunCommand\n{\n}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); changed || got != src {
		t.Fatalf("attribute-used import must be kept; changed=%v got=%q", changed, got)
	}
}

// PHP-CS-Fixer drops an import of the current namespace with a case-sensitive
// prefix match: an import whose path differs from the namespace only by case
// is a different name to it, so it stays when it is used.
func TestNoUnusedImportsSameNamespaceIsCaseSensitive(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nnamespace acme\\Shop;\n\nuse Acme\\Shop\\CartInterface;\n\nfinal class Cart implements CartInterface\n{\n}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); changed || got != src {
		t.Fatalf("import differing from the namespace by case must be kept; changed=%v got=%q", changed, got)
	}
	same := "<?php\n\nnamespace Acme\\Shop;\n\nuse Acme\\Shop\\CartInterface;\n\nfinal class Cart implements CartInterface\n{\n}\n"
	want := "<?php\n\nnamespace Acme\\Shop;\n\n\nfinal class Cart implements CartInterface\n{\n}\n"
	if got, changed := apply(t, NoUnusedImports{}, same); !changed || got != want {
		t.Fatalf("import of the current namespace must be removed; changed=%v got=%q want=%q", changed, got, want)
	}
}

// A comment between "use" and the imported name is not part of the name: the
// import of the current namespace is still recognised as redundant. As in
// PHP-CS-Fixer, removing an import keeps the comments inside it, and the line
// break before the import moves after them.
func TestNoUnusedImportsCommentInsideImport(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nnamespace Acme\\Shop;\n\nuse /** @noinspection PhpUnusedAliasInspection */ Acme\\Shop\\CartInterface;\nuse /* kept */ Vendor\\Lib\\Money;\n\nfinal class Cart implements CartInterface\n{\n    public Money $total;\n}\n"
	want := "<?php\n\nnamespace Acme\\Shop;/** @noinspection PhpUnusedAliasInspection */\n\nuse /* kept */ Vendor\\Lib\\Money;\n\nfinal class Cart implements CartInterface\n{\n    public Money $total;\n}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	src = "<?php\nnamespace App;\nuse Vendor\\Foo; // trailing\nuse Vendor\\Bar;\n  use /* x */ Vendor\\Baz /* y */;\nuse Vendor\\Qux;\n\nnew Qux();\n"
	want = "<?php\nnamespace App;\n// trailing/* x *//* y */\nuse Vendor\\Qux;\n\nnew Qux();\n"
	if got, changed := apply(t, NoUnusedImports{}, src); !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

// PHP's T_OPEN_TAG includes the whitespace character after "<?php", so removing
// an import on the open tag's line keeps "<?php " apart from the next token
// (outputs from PHP-CS-Fixer).
func TestNoUnusedImportsOnOpenTagLine(t *testing.T) {
	t.Parallel()
	for src, want := range map[string]string{
		"<?php use Foo\\A; new B;":     "<?php new B;",
		"<?php  use Foo\\A;  new B;\n": "<?php new B;\n",
		"<html><?php use Foo\\A; ?>":   "<html><?php ?>",
		"<?php use Foo\\A;\nnew B;\n":  "<?php new B;\n",
		"<?php\nuse Foo\\A;\nnew B;\n": "<?php\nnew B;\n",
	} {
		if got, changed := apply(t, NoUnusedImports{}, src); !changed || got != want {
			t.Errorf("%q: changed=%v got=%q want=%q", src, changed, got, want)
		}
	}
}

// A comment between "use" and "function"/"const" does not turn the import
// into a class import (PHP-CS-Fixer reads the next meaningful token).
func TestNoUnusedImportsCommentBeforeImportKind(t *testing.T) {
	t.Parallel()
	for _, src := range []string{
		"<?php\nuse /* c */ function Foo\\bar;\nbar();\n",
		"<?php\nuse /** x */ const Foo\\BAR;\necho BAR;\n",
	} {
		if got, changed := apply(t, NoUnusedImports{}, src); changed || got != src {
			t.Errorf("used import must be kept: changed=%v got=%q", changed, got)
		}
	}
}

// PHP-CS-Fixer removes an unused import of the current namespace twice (once
// as unused, once as redundant), and the second pass eats one more line break.
func TestNoUnusedImportsUnusedSameNamespaceRemovedTwice(t *testing.T) {
	t.Parallel()
	src := "<?php\nnamespace Acme;\n\nuse Acme\\Foo;\n\nclass X {}\n"
	want := "<?php\nnamespace Acme;\n\nclass X {}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	src = "<?php\nnamespace Acme;\n\nuse Acme\\Foo;\n\nclass X extends Foo {}\n"
	want = "<?php\nnamespace Acme;\n\n\nclass X extends Foo {}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

// isImportUsed counts any matching name inside an attribute, whatever the
// import kind, but never a named-argument name or a goto label, and reads the
// neighbours of a name past comments.
func TestNoUnusedImportsUsageLikePhpCsFixer(t *testing.T) {
	t.Parallel()
	kept := "<?php\nnamespace App;\n\nuse Vendor\\Option;\n\n#[Route(path: '/', methods: Option())]\nfunction run(): void {}\n"
	if got, changed := apply(t, NoUnusedImports{}, kept); changed || got != kept {
		t.Fatalf("name inside an attribute must count; changed=%v got=%q", changed, got)
	}
	kept = "<?php\nnamespace App;\n\nuse Vendor\\Thing;\n\n$x = new /* the */ Thing();\n"
	if got, changed := apply(t, NoUnusedImports{}, kept); changed || got != kept {
		t.Fatalf("comment between new and the name must be skipped; changed=%v got=%q", changed, got)
	}
	src := "<?php\nnamespace App;\n\nuse Vendor\\Label;\nuse Vendor\\Limit;\n\nfetch(limit: 3);\nlabel:\necho 1;\n"
	want := "<?php\nnamespace App;\n\n\nfetch(limit: 3);\nlabel:\necho 1;\n"
	if got, changed := apply(t, NoUnusedImports{}, src); !changed || got != want {
		t.Fatalf("named argument and goto label are not usages; changed=%v got=%q want=%q", changed, got, want)
	}
}

// The expected outputs below were produced by PHP-CS-Fixer's NoUnusedImportsFixer
// (through ECS) on the same inputs.
func TestNoUnusedImportsMatchesPHPCSFixer(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, src, want string }{
		// multi-use: only the unused name is removed
		{"multi1", "<?php\nnamespace App;\nuse Foo\\Used, Bar\\Unused;\nnew Used();\n", "<?php\nnamespace App;\nuse Foo\\Used;\nnew Used();\n"},
		{"multi2", "<?php\nnamespace App;\nuse Foo\\Unused, Bar\\Used;\nnew Used();\n", "<?php\nnamespace App;\nuse Bar\\Used;\nnew Used();\n"},
		// multi-use with every name unused: the statement goes
		{"multi3", "<?php\nnamespace App;\nuse Foo\\A, Bar\\B;\nnew C();\n", "<?php\nnamespace App;\nnew C();\n"},
		// multi-line multi-use; PHP-CS-Fixer drops the ";" here too
		{"multi4", "<?php\nnamespace App;\nuse Foo\\A,\n    Bar\\B,\n    Baz\\C;\nnew B();\n", "<?php\nnamespace App;\nuse \n    Bar\\B\nnew B();\n"},
		{"multi5", "<?php\nnamespace App;\nuse Foo\\A as X, Bar\\B as Y;\nnew Y();\n", "<?php\nnamespace App;\nuse Bar\\B as Y;\nnew Y();\n"},
		// multi-use with an import of the current namespace
		{"multi6", "<?php\nnamespace App;\nuse App\\A, Bar\\B;\nnew A(); new B();\n", "<?php\nnamespace App;\nuse Bar\\B;\nnew A(); new B();\n"},
		{"multi7", "<?php\nnamespace App;\nuse function Foo\\a, Foo\\b;\nb();\n", "<?php\nnamespace App;\nuse function Foo\\b;\nb();\n"},
		// group use: the unused name is pruned
		{"group1", "<?php\nnamespace App;\nuse Foo\\{A, B};\nnew A();\n", "<?php\nnamespace App;\nuse Foo\\{A};\nnew A();\n"},
		{"group2", "<?php\nnamespace App;\nuse Foo\\{A, B};\nnew B();\n", "<?php\nnamespace App;\nuse Foo\\{B};\nnew B();\n"},
		// group use with every name unused
		{"group3", "<?php\nnamespace App;\nuse Foo\\{A, B};\nnew C();\n", "<?php\nnamespace App;\nnew C();\n"},
		{"group4", "<?php\nnamespace App;\nuse Foo\\{\n    A,\n    B,\n    C,\n};\nnew B();\n", "<?php\nnamespace App;\nuse Foo\\{\n    B,\n};\nnew B();\n"},
		{"group5", "<?php\nnamespace App;\nuse Foo\\{\n    A,\n    B\n};\nnew A();\n", "<?php\nnamespace App;\nuse Foo\\{\n    A\n};\nnew A();\n"},
		// a group use is not a usage of another import
		{"group6", "<?php\nnamespace App;\nuse function A\\{foo};\nuse B\\Foo;\nfoo();\n", "<?php\nnamespace App;\nuse function A\\{foo};\nfoo();\n"},
		{"group7", "<?php\nnamespace App;\nuse Foo\\{A, B as C, D\\E};\nnew E(); new C();\n", "<?php\nnamespace App;\nuse Foo\\{B as C, D\\E};\nnew E(); new C();\n"},
		{"group9", "<?php\nnamespace App;\nuse Foo\\{A, B};\nuse Bar\\{C};\nnew A();\n", "<?php\nnamespace App;\nuse Foo\\{A};\nnew A();\n"},
		{"group10", "<?php\nnamespace Foo;\nuse Foo\\{A, B};\nnew A(); new B();\n", "<?php\nnamespace Foo;\nnew A(); new B();\n"},
		{"trailing", "<?php\nnamespace App;\nuse Foo\\{A, B,};\nnew A();\n", "<?php\nnamespace App;\nuse Foo\\{A,};\nnew A();\n"},
		// "?>" ends a use statement and stays
		{"close1", "<?php\nuse Foo\\Bar ?>\nhello <?php echo 1;\n", "<?php\n?>\nhello <?php echo 1;\n"},
		{"close2", "<?php\nuse Foo\\Bar ?>\nhello <?php new Bar;\n", "<?php\nuse Foo\\Bar ?>\nhello <?php new Bar;\n"},
		{"close3", "<?php use Foo\\A, Foo\\B ?>\n<?php new B;\n", "<?php use Foo\\B ?>\n<?php new B;\n"},
		// a bare name is a constant: a class import is unused
		{"const1", "<?php\nnamespace App;\nuse Foo\\Bar;\necho Bar;\n", "<?php\nnamespace App;\necho Bar;\n"},
		{"const2", "<?php\nnamespace App;\nuse Foo\\Bar;\n$a ? 1 : Bar;\n", "<?php\nnamespace App;\n$a ? 1 : Bar;\n"},
		// "new BAR" is a class: a const import is unused
		{"const3", "<?php\nnamespace App;\nuse const Foo\\BAR;\nnew BAR;\n", "<?php\nnamespace App;\nnew BAR;\n"},
		{"const4", "<?php\nnamespace App;\nuse const Foo\\BAR;\n$x instanceof BAR;\n", "<?php\nnamespace App;\n$x instanceof BAR;\n"},
		{"const5", "<?php\nnamespace App;\nuse const Foo\\BAR;\necho BAR | 1;\n", "<?php\nnamespace App;\nuse const Foo\\BAR;\necho BAR | 1;\n"},
		// types: union, nullable return
		{"const6", "<?php\nnamespace App;\nuse Foo\\Bar;\nfunction f(Bar|null $x): ?Bar { return null; }\n", "<?php\nnamespace App;\nuse Foo\\Bar;\nfunction f(Bar|null $x): ?Bar { return null; }\n"},
		{"const7", "<?php\nnamespace App;\nuse Foo\\Bar;\nuse Foo\\Baz;\nfunction f(): Bar|Baz {}\n", "<?php\nnamespace App;\nuse Foo\\Bar;\nuse Foo\\Baz;\nfunction f(): Bar|Baz {}\n"},
		// non-capturing catch
		{"const8", "<?php\nnamespace App;\nuse Foo\\Bar;\ntry {} catch (Bar) {}\n", "<?php\nnamespace App;\nuse Foo\\Bar;\ntry {} catch (Bar) {}\n"},
		// by-reference parameter type
		{"const9", "<?php\nnamespace App;\nuse Foo\\Bar;\nfunction f(Bar &$x) {}\n", "<?php\nnamespace App;\nuse Foo\\Bar;\nfunction f(Bar &$x) {}\n"},
		// bitwise and with a variable
		{"const10", "<?php\nnamespace App;\nuse const Foo\\BAR;\n$y = BAR & $x;\n", "<?php\nnamespace App;\nuse const Foo\\BAR;\n$y = BAR & $x;\n"},
		{"const11", "<?php\nnamespace App;\nuse Foo\\Bar;\nclass X extends Y implements Z, Bar {}\n", "<?php\nnamespace App;\nuse Foo\\Bar;\nclass X extends Y implements Z, Bar {}\n"},
		// switch case
		{"const12", "<?php\nnamespace App;\nuse const Foo\\BAR;\nswitch ($a) { case BAR: break; }\n", "<?php\nnamespace App;\nuse const Foo\\BAR;\nswitch ($a) { case BAR: break; }\n"},
		// enum case
		{"const13", "<?php\nnamespace App;\nuse Foo\\Bar;\nenum E { case Bar; }\n", "<?php\nnamespace App;\nuse Foo\\Bar;\nenum E { case Bar; }\n"},
		// DNF type
		{"const14", "<?php\nnamespace App;\nuse Foo\\Bar;\nuse Foo\\Baz;\nfunction f((Bar&Baz)|null $x) {}\n", "<?php\nnamespace App;\nuse Foo\\Bar;\nuse Foo\\Baz;\nfunction f((Bar&Baz)|null $x) {}\n"},
		// constant in a {$...} interpolation
		{"const15", "<?php\nnamespace App;\nuse const Foo\\BAR;\necho \"{$a[BAR]}\";\n", "<?php\nnamespace App;\nuse const Foo\\BAR;\necho \"{$a[BAR]}\";\n"},
		{"const16", "<?php\nnamespace App;\nuse Foo\\Bar;\nforeach (Bar as $x) {}\n", "<?php\nnamespace App;\nforeach (Bar as $x) {}\n"},
		// promoted nullable property
		{"const17", "<?php\nnamespace App;\nuse Foo\\Bar;\nclass C { public function __construct(private ?Bar $b) {} }\n", "<?php\nnamespace App;\nuse Foo\\Bar;\nclass C { public function __construct(private ?Bar $b) {} }\n"},
		{"const18", "<?php\nnamespace App;\nuse const Foo\\BAR;\n$a = [BAR => 1, 2 => BAR];\n", "<?php\nnamespace App;\nuse const Foo\\BAR;\n$a = [BAR => 1, 2 => BAR];\n"},
		// variadic parameter type
		{"const19", "<?php\nnamespace App;\nuse Foo\\Bar;\n$f = fn(Bar ...$b) => 1;\n", "<?php\nnamespace App;\nuse Foo\\Bar;\n$f = fn(Bar ...$b) => 1;\n"},
		// class constant declaration
		{"const20", "<?php\nnamespace App;\nuse Foo\\Bar;\nclass C { const Bar = 1; }\n", "<?php\nnamespace App;\nclass C { const Bar = 1; }\n"},
		// comment in the namespace declaration is part of its name
		{"ns1", "<?php\nnamespace /* x */ App;\nuse App\\A;\nnew A();\n", "<?php\nnamespace /* x */ App;\nuse App\\A;\nnew A();\n"},
		// two namespaces
		{"ns2", "<?php\nnamespace App;\nuse Foo\\A;\nnamespace Other;\nnew A();\n", "<?php\nnamespace App;\nnamespace Other;\nnew A();\n"},
		// inline HTML before the open tag
		{"ns3", "<html>\n<?php\nuse Foo\\A;\nnew B();\n", "<html>\n<?php\nnew B();\n"},
		{"ns4", "<?php\nnamespace App {\n    use Foo\\A, Foo\\B;\n    new A();\n}\n", "<?php\nnamespace App {\n    use Foo\\A;\n    new A();\n}\n"},
		{"crlf", "<?php\r\nnamespace App;\r\n\r\nuse Foo\\A;\r\nuse Foo\\B;\r\n\r\nnew B();\r\n", "<?php\r\nnamespace App;\r\n\r\nuse Foo\\B;\r\n\r\nnew B();\r\n"},
		{"crlfmulti", "<?php\r\nnamespace App;\r\nuse Foo\\A,\r\n    Foo\\B;\r\nnew B();\r\n", "<?php\r\nnamespace App;\r\nuse \r\n    Foo\\B;\r\nnew B();\r\n"},
		// comments inside a multi-use are kept
		{"linecmt", "<?php\nnamespace App;\nuse Foo\\A, // first\n    Foo\\B; // second\nnew B();\n", "<?php\nnamespace App;\nuse // first\n    Foo\\B; // second\nnew B();\n"},
		{"attr", "<?php\nnamespace App;\nuse const Foo\\BAR;\n#[Attr(BAR)]\nclass C {}\n", "<?php\nnamespace App;\nuse const Foo\\BAR;\n#[Attr(BAR)]\nclass C {}\n"},
		{"goto", "<?php\nnamespace App;\nuse Foo\\Bar;\nBar:\ngoto Bar;\n", "<?php\nnamespace App;\nuse Foo\\Bar;\nBar:\ngoto Bar;\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, changed := apply(t, NoUnusedImports{}, tc.src)
			if got != tc.want || changed != (tc.src != tc.want) {
				t.Fatalf("changed=%v\ngot  %q\nwant %q", changed, got, tc.want)
			}
		})
	}
}
