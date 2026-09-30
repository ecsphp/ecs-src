package rules

import "testing"

func TestNoUnusedImportsRemovesUnused(t *testing.T) {
	got, changed := apply(t, NoUnusedImports{}, "<?php\nnamespace App;\nuse Vendor\\Foo;\nuse Vendor\\Bar;\n\nnew Foo();\n")
	if want := "<?php\nnamespace App;\nuse Vendor\\Foo;\n\nnew Foo();\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

// A use that is used inside its own brace-delimited namespace block must be kept,
// even when a different block defines a class of the same short name.
func TestNoUnusedImportsBraceNamespaceUsed(t *testing.T) {
	src := "<?php\n\nnamespace Foo\\Bar {\n    use Psr\\Log\\LoggerInterface;\n\n    class Baz\n    {\n        public function __construct(LoggerInterface $logger) {}\n    }\n}\n\nnamespace {\n    use Foo\\Bar\\Baz;\n\n    function run(): void {\n        Baz::go();\n    }\n}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); changed || got != src {
		t.Fatalf("brace-namespace import must be kept; changed=%v got=%q", changed, got)
	}
}

// An unused import inside a brace-delimited global namespace is still removed.
func TestNoUnusedImportsBraceNamespaceUnused(t *testing.T) {
	src := "<?php\n\nnamespace {\n    use Foo\\Bar\\Baz;\n\n    function run(): void {\n    }\n}\n"
	want := "<?php\n\nnamespace {\n    \n    function run(): void {\n    }\n}\n"
	if got, changed := apply(t, NoUnusedImports{}, src); !changed || got != want {
		t.Fatalf("unused brace-namespace import must be removed; changed=%v got=%q want=%q", changed, got, want)
	}
}
