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
