package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen4ASingleLineCommentStyle(t *testing.T) {
	t.Parallel()
	f := SingleLineCommentStyle{}

	// hash comment -> // comment
	assertFix(t, f, "<?php\n# foo\n", "<?php\n// foo\n", true)
	idempotent(t, f, "<?php\n// foo\n")

	// hash without a space keeps the text as-is, only the marker changes
	assertFix(t, f, "<?php\n#foo\n", "<?php\n//foo\n", true)

	// a bare "#" becomes "//"
	assertFix(t, f, "<?php\n#\n", "<?php\n//\n", true)

	// single-line block comment at end of line -> // comment
	assertFix(t, f, "<?php\n/* foo */\n$a = 1;", "<?php\n// foo\n$a = 1;", true)
	idempotent(t, f, "<?php\n// foo\n$a = 1;")

	// surrounding asterisks and whitespace are trimmed, inner stars kept
	assertFix(t, f, "<?php\n/*  a * b  */\n", "<?php\n// a * b\n", true)

	// empty block comment collapses to //
	assertFix(t, f, "<?php\n/**/\n", "<?php\n//\n", true)

	// already-// comment is a no-op
	if _, changed := apply(t, f, "<?php\n// foo\n"); changed {
		t.Fatal("already // comment must not change")
	}

	// DANGEROUS: a #[...] attribute is not a hash comment
	if got, changed := apply(t, f, "<?php\n#[Route('/x')]\nfunction f() {}"); changed {
		t.Fatalf("attribute must not be touched: got=%q", got)
	}

	// DANGEROUS: a multi-line /* */ block is left alone
	if got, changed := apply(t, f, "<?php\n/*\n * foo\n */\n$a = 1;"); changed {
		t.Fatalf("multi-line block comment must not change: got=%q", got)
	}

	// a single-line doc comment (/** */) is not a plain comment, left alone
	if got, changed := apply(t, f, "<?php\n/** foo */\n$a = 1;"); changed {
		t.Fatalf("doc comment must not change: got=%q", got)
	}

	// a block comment with code following on the same line must not be converted,
	// otherwise // would swallow that code
	if got, changed := apply(t, f, "<?php /* foo */ $a = 1;"); changed {
		t.Fatalf("inline block comment before code must not change: got=%q", got)
	}

	// a block comment containing a close tag is left alone
	if got, changed := apply(t, f, "<?php\n/* a ?> b */\n"); changed {
		t.Fatalf("block comment with ?> must not change: got=%q", got)
	}
}

func TestGen4ANoNullPropertyInitialization(t *testing.T) {
	t.Parallel()
	f := NoNullPropertyInitialization{}

	// untyped property loses its null default
	assertFix(t, f,
		"<?php\nclass Foo\n{\n    public $bar = null;\n}\n",
		"<?php\nclass Foo\n{\n    public $bar;\n}\n", true)
	idempotent(t, f, "<?php\nclass Foo\n{\n    public $bar;\n}\n")

	// multiple properties in one statement are all cleaned
	assertFix(t, f,
		"<?php\nclass Foo\n{\n    public $a = null, $b = null;\n}\n",
		"<?php\nclass Foo\n{\n    public $a, $b;\n}\n", true)

	// static untyped property is handled
	assertFix(t, f,
		"<?php\nclass Foo\n{\n    public static $foo = null;\n}\n",
		"<?php\nclass Foo\n{\n    public static $foo;\n}\n", true)

	// var-declared property is handled
	assertFix(t, f,
		"<?php\nclass Foo\n{\n    var $x = null;\n}\n",
		"<?php\nclass Foo\n{\n    var $x;\n}\n", true)

	// traits hold properties too
	assertFix(t, f,
		"<?php\ntrait T\n{\n    private $y = null;\n}\n",
		"<?php\ntrait T\n{\n    private $y;\n}\n", true)

	// an attribute above the property is untouched, the default still removed
	assertFix(t, f,
		"<?php\nclass Foo\n{\n    #[ORM\\Column]\n    public $z = null;\n}\n",
		"<?php\nclass Foo\n{\n    #[ORM\\Column]\n    public $z;\n}\n", true)

	// DANGEROUS: a typed property keeps its null default (typing differs)
	if got, changed := apply(t, f, "<?php\nclass Foo\n{\n    public ?string $baz = null;\n}\n"); changed {
		t.Fatalf("typed property must keep null default: got=%q", got)
	}

	// DANGEROUS: a class constant is not a property
	if got, changed := apply(t, f, "<?php\nclass Foo\n{\n    const X = null;\n}\n"); changed {
		t.Fatalf("const must not change: got=%q", got)
	}
	if got, changed := apply(t, f, "<?php\nclass Foo\n{\n    public const X = null;\n}\n"); changed {
		t.Fatalf("public const must not change: got=%q", got)
	}

	// DANGEROUS: a parameter default is not a property
	if got, changed := apply(t, f, "<?php\nclass Foo\n{\n    public function f($x = null) {}\n}\n"); changed {
		t.Fatalf("parameter default must not change: got=%q", got)
	}

	// a non-null default is left alone
	if got, changed := apply(t, f, "<?php\nclass Foo\n{\n    public $x = 1;\n}\n"); changed {
		t.Fatalf("non-null default must not change: got=%q", got)
	}

	// a property with no default is a no-op
	if got, changed := apply(t, f, "<?php\nclass Foo\n{\n    public $x;\n}\n"); changed {
		t.Fatalf("property without default must not change: got=%q", got)
	}
}

// TestGen4ASourceURL verifies each SourceURL matches SourceURLFor(Name()).
func TestGen4ASourceURL(t *testing.T) {
	t.Parallel()
	for _, f := range []fixer.Fixer{SingleLineCommentStyle{}, NoNullPropertyInitialization{}} {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}
