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
	want := "<?php\n\nnamespace {\n    \n    function run(): void {\n    }\n}\n"
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
