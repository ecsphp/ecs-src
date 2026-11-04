package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestNoSuperfluousPhpdocTags(t *testing.T) {
	t.Parallel()
	f := NoSuperfluousPhpdocTags{}

	// @param equal to the native type, no description -> removed
	assertFix(t, f,
		"<?php\n/**\n * @param int $iterations\n */\nfunction f(int $iterations) {}",
		"<?php\n/**\n */\nfunction f(int $iterations) {}", true)

	// @param with a generic phpdoc type is more specific -> kept
	assertFix(t, f,
		"<?php\n/**\n * @param array<callable> $b\n */\nfunction f(array $b) {}",
		"<?php\n/**\n * @param array<callable> $b\n */\nfunction f(array $b) {}", false)

	// @param with a description -> kept
	assertFix(t, f,
		"<?php\n/**\n * @param int $n the count\n */\nfunction f(int $n) {}",
		"<?php\n/**\n * @param int $n the count\n */\nfunction f(int $n) {}", false)

	// @param mixed on an untyped param -> kept (ECS allow_mixed: mixed adds info)
	assertFix(t, f,
		"<?php\n/**\n * @param mixed $v\n */\nfunction f($v) {}",
		"<?php\n/**\n * @param mixed $v\n */\nfunction f($v) {}", false)

	// @param mixed matching a native mixed type -> removed (superfluous)
	assertFix(t, f,
		"<?php\n/**\n * @param mixed $v\n */\nfunction f(mixed $v) {}",
		"<?php\n/**\n */\nfunction f(mixed $v) {}", true)

	// untyped param with a real phpdoc type -> kept (phpdoc adds info)
	assertFix(t, f,
		"<?php\n/**\n * @param string $v\n */\nfunction f($v) {}",
		"<?php\n/**\n * @param string $v\n */\nfunction f($v) {}", false)

	// @return matching the native return type -> removed
	assertFix(t, f,
		"<?php\n/**\n * @return never\n */\nfunction f(): never {}",
		"<?php\n/**\n */\nfunction f(): never {}", true)

	// @return mixed without a native return type -> kept
	assertFix(t, f,
		"<?php\n/**\n * @return mixed\n */\nfunction f() {}",
		"<?php\n/**\n * @return mixed\n */\nfunction f() {}", false)

	// @return more specific than native -> kept
	assertFix(t, f,
		"<?php\n/**\n * @return array<int>|float\n */\nfunction f(): array|float {}",
		"<?php\n/**\n * @return array<int>|float\n */\nfunction f(): array|float {}", false)

	// FQ phpdoc type resolves to the imported short native type (same class) -> removed
	assertFix(t, f,
		"<?php\nnamespace App;\nuse A\\B\\Foo;\n/**\n * @param \\A\\B\\Foo $x\n */\nfunction f(Foo $x) {}",
		"<?php\nnamespace App;\nuse A\\B\\Foo;\n/**\n */\nfunction f(Foo $x) {}", true)

	// FQ phpdoc type is a DIFFERENT class than the imported native short name -> kept
	assertFix(t, f,
		"<?php\nnamespace App;\nuse Mautic\\LeadBundle\\Entity\\Tag;\n/**\n * @param \\MauticPlugin\\Foo\\Entity\\Tag $tag\n */\nfunction f(Tag $tag) {}",
		"<?php\nnamespace App;\nuse Mautic\\LeadBundle\\Entity\\Tag;\n/**\n * @param \\MauticPlugin\\Foo\\Entity\\Tag $tag\n */\nfunction f(Tag $tag) {}", false)

	// imported short name repeated in the docblock (same class) -> removed
	assertFix(t, f,
		"<?php\nnamespace App;\nuse A\\B\\Foo;\n/**\n * @param Foo $x\n */\nfunction f(Foo $x) {}",
		"<?php\nnamespace App;\nuse A\\B\\Foo;\n/**\n */\nfunction f(Foo $x) {}", true)

	// nullable equivalence: string|null == ?string -> removed
	assertFix(t, f,
		"<?php\n/**\n * @param string|null $x\n */\nfunction f(?string $x) {}",
		"<?php\n/**\n */\nfunction f(?string $x) {}", true)

	// @var equal to the typed property -> removed
	assertFix(t, f,
		"<?php\nclass C {\n/**\n * @var array\n */\nprotected array $items;\n}",
		"<?php\nclass C {\n/**\n */\nprotected array $items;\n}", true)

	// @var mixed on an untyped property -> kept
	assertFix(t, f,
		"<?php\nclass C {\n/**\n * @var mixed\n */\nprotected $value;\n}",
		"<?php\nclass C {\n/**\n * @var mixed\n */\nprotected $value;\n}", false)

	// param default value does not break signature parsing
	assertFix(t, f,
		"<?php\n/**\n * @param bool $dev\n */\nfunction f(bool $dev = false) {}",
		"<?php\n/**\n */\nfunction f(bool $dev = false) {}", true)

	// a docblock with a summary keeps the summary, drops only the superfluous tag
	assertFix(t, f,
		"<?php\n/**\n * Does a thing.\n *\n * @param int $n\n */\nfunction f(int $n) {}",
		"<?php\n/**\n * Does a thing.\n *\n */\nfunction f(int $n) {}", true)
}

// A docblock documenting a closure parameter, placed before an expression
// statement (a method call taking the closure), is not attached to a function or
// property declaration, so its @param must be kept even with allow_unused_params.
func TestNoSuperfluousPhpdocTagsKeepsDocBeforeExpression(t *testing.T) {
	t.Parallel()
	f := NoSuperfluousPhpdocTags{}.WithConfig(map[string]any{"allow_unused_params": false}).(fixerRule)

	src := "<?php\nclass A {\n    public function run(array $stmts): void\n    {\n" +
		"        /** @param Use_::TYPE_* $useType */\n" +
		"        $this->traverser->traverse($stmts, static function (int $useType) {});\n" +
		"    }\n}\n"
	assertFix(t, f, src, src, false)
}

func TestNoSuperfluousPhpdocTagsSourceURL(t *testing.T) {
	t.Parallel()
	f := NoSuperfluousPhpdocTags{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Fatalf("SourceURL %q, want %q", got, want)
	}
}

// A @param whose native type is fully covered is superfluous even when the
// parameter carries an attribute ("#[\SensitiveParameter] array $x").
func TestNoSuperfluousPhpdocTagsAttributedParam(t *testing.T) {
	t.Parallel()
	src := "<?php\n\nclass C\n{\n    /**\n     * @param array $data\n     * @return void\n     */\n    public function run(#[\\SensitiveParameter] array $data)\n    {\n    }\n}\n"
	want := "<?php\n\nclass C\n{\n    /**\n     * @return void\n     */\n    public function run(#[\\SensitiveParameter] array $data)\n    {\n    }\n}\n"
	if got, changed := apply(t, NoSuperfluousPhpdocTags{}, src); !changed || got != want {
		t.Fatalf("attributed-param @param must be removed; changed=%v got=%q", changed, got)
	}
}

// PHP-CS-Fixer's findDocumentedElement stops at a promoted constructor
// parameter's visibility (a custom token), so such a docblock is never checked;
// one before a "readonly"-led promotion or a plain parameter documents a property.
func TestNoSuperfluousPhpdocTagsPromotedProperty(t *testing.T) {
	t.Parallel()
	f := NoSuperfluousPhpdocTags{}.WithConfig(map[string]any{"allow_mixed": true}).(fixerRule)
	kept := "<?php\nclass A\n{\n    public function __construct(\n        /**\n         * @var string\n         */\n" +
		"        public string $a,\n        /**\n         * @var int\n         */\n        private readonly int $b,\n    ) {}\n}\n"
	assertFix(t, f, kept, kept, false)
	assertFix(t, f,
		"<?php\nclass A\n{\n    public function __construct(\n        /**\n         * @var string\n         */\n"+
			"        readonly string $a,\n    ) {}\n}\n",
		"<?php\nclass A\n{\n    public function __construct(\n        /**\n         */\n"+
			"        readonly string $a,\n    ) {}\n}\n", true)
}

// "var" is not in PHP-CS-Fixer's MODIFIER_KINDS, so a docblock before a "var"
// property documents nothing the fixer checks; a one-line docblock emptied by
// the fixer becomes "/**  */".
func TestNoSuperfluousPhpdocTagsVarAndEmptiedSingleLine(t *testing.T) {
	t.Parallel()
	f := NoSuperfluousPhpdocTags{}.WithConfig(map[string]any{"allow_mixed": true}).(fixerRule)
	kept := "<?php\nclass A\n{\n    /** @var string */\n    var string $a;\n}\n"
	assertFix(t, f, kept, kept, false)
	assertFix(t, f,
		"<?php\nclass A\n{\n    /** @var string */\n    public string $a;\n    /** @return int */\n    public function f(): int {}\n}\n",
		"<?php\nclass A\n{\n    /**  */\n    public string $a;\n    /**  */\n    public function f(): int {}\n}\n", true)
}

// PHP 8.4 set-visibility ("private(set)") is one of the fixer's modifiers, so
// the docblock before it documents a property, also on a promoted parameter
// that it leads (ConstructorPromotionTransformer leaves those alone).
func TestNoSuperfluousPhpdocTagsSetVisibility(t *testing.T) {
	t.Parallel()
	f := NoSuperfluousPhpdocTags{}.WithConfig(map[string]any{"allow_mixed": true}).(fixerRule)
	assertFix(t, f,
		"<?php\nfinal class A\n{\n    /** @var int */ public private(set) int $a;\n    /** @var int */ private(set) int $b;\n    /** @var int */ protected(set) readonly int $c;\n    /**\n     * @var string\n     */\n    public(set) string $s;\n\n    public function __construct(\n        /** @var int */ private(set) int $d,\n        /** @var int */ public private(set) int $e,\n    ) {\n    }\n}\n",
		"<?php\nfinal class A\n{\n    /**  */ public private(set) int $a;\n    /**  */ private(set) int $b;\n    /**  */ protected(set) readonly int $c;\n    /**\n     */\n    public(set) string $s;\n\n    public function __construct(\n        /**  */ private(set) int $d,\n        /** @var int */ public private(set) int $e,\n    ) {\n    }\n}\n", true)
}
