package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen4PhpdocLineSpan(t *testing.T) {
	f := PhpdocLineSpan{}

	// single-line property docblock is expanded to multi-line (4-space indent)
	assertFix(t, f,
		"<?php\nclass A{\n    /** @var int */\n    public $x;\n}",
		"<?php\nclass A{\n    /**\n     * @var int\n     */\n    public $x;\n}", true)

	// private/protected/var modifiers all trigger the expansion
	assertFix(t, f,
		"<?php\nclass A{\n    /** @var int */\n    private $x;\n}",
		"<?php\nclass A{\n    /**\n     * @var int\n     */\n    private $x;\n}", true)
	assertFix(t, f,
		"<?php\nclass A{\n    /** @var int */\n    var $x;\n}",
		"<?php\nclass A{\n    /**\n     * @var int\n     */\n    var $x;\n}", true)

	// docblock before a method with visibility is expanded
	assertFix(t, f,
		"<?php\nclass A{\n    /** does things */\n    public function run() {}\n}",
		"<?php\nclass A{\n    /**\n     * does things\n     */\n    public function run() {}\n}", true)

	// top-level docblock (no indent) expands to column 0
	assertFix(t, f,
		"<?php\n/** @var int */\npublic $x;",
		"<?php\n/**\n * @var int\n */\npublic $x;", true)

	// already multi-line - no-op
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @var int\n     */\n    public $x;\n}",
		"<?php\nclass A{\n    /**\n     * @var int\n     */\n    public $x;\n}", false)

	// docblock that does not document a class member is left alone
	assertFix(t, f,
		"<?php\n/** @var int */\n$x = 1;",
		"<?php\n/** @var int */\n$x = 1;", false)

	// free function (bare, no visibility) is intentionally not touched
	assertFix(t, f,
		"<?php\n/** does things */\nfunction f() {}",
		"<?php\n/** does things */\nfunction f() {}", false)

	// bare const (ambiguous top-level vs class) is left alone
	assertFix(t, f,
		"<?php\n/** @var int */\nconst X = 1;",
		"<?php\n/** @var int */\nconst X = 1;", false)
}

func TestGen4PhpdocLineSpanLocalStatic(t *testing.T) {
	// mautic parity: configured multi (const/method/property) must leave a
	// /** @var ... */ on a local `static $x` inside a method body UNCHANGED,
	// because it is not a class member (PHP getClassyElements skips locals).
	f := PhpdocLineSpan{}.WithConfig(map[string]any{
		"const": "multi", "method": "multi", "property": "multi",
	})

	src := "<?php\nclass A{\n    private function run(): void\n    {\n        /** @var array<string, string|null> $cache */\n        static $cache = [];\n    }\n}"
	assertFix(t, f, src, src, false)

	// a real class property IS still expanded under the same config
	assertFix(t, f,
		"<?php\nclass A{\n    /** @var int */\n    public $x;\n}",
		"<?php\nclass A{\n    /**\n     * @var int\n     */\n    public $x;\n}", true)
}

func TestGen4PhpdocLineSpanSourceURL(t *testing.T) {
	var f fixer.Fixer = PhpdocLineSpan{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Fatalf("SourceURL %q, want %q", got, want)
	}
}
