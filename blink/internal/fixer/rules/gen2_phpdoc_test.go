package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen2PhpdocIndent(t *testing.T) {
	f := PhpdocIndent{}

	// misaligned continuation "*" and closing "*/" inside a class (4-space indent)
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n* hi\n     */\n    public $x;\n}",
		"<?php\nclass A{\n    /**\n     * hi\n     */\n    public $x;\n}", true)

	// top-level docblock aligns to column 0
	assertFix(t, f,
		"<?php\n/**\n  * hi\n  */\nclass A{}",
		"<?php\n/**\n * hi\n */\nclass A{}", true)

	// already correct - no-op
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * hi\n     */\n    public $x;\n}",
		"<?php\nclass A{\n    /**\n     * hi\n     */\n    public $x;\n}", false)

	// blank continuation line keeps its "*" aligned, no content
	assertFix(t, f,
		"<?php\n/**\n *\n * hi\n */\nclass A{}",
		"<?php\n/**\n *\n * hi\n */\nclass A{}", false)

	// single-line docblock is untouched
	assertFix(t, f,
		"<?php\n/** @var int */\nclass A{}",
		"<?php\n/** @var int */\nclass A{}", false)
}

func TestGen2PhpdocOrderByValue(t *testing.T) {
	f := PhpdocOrderByValue{}

	// two @covers lines sorted by value
	assertFix(t, f,
		"<?php\n/**\n * @covers \\B\\Foo\n * @covers \\A\\Bar\n */\nclass T{}",
		"<?php\n/**\n * @covers \\A\\Bar\n * @covers \\B\\Foo\n */\nclass T{}", true)

	// already sorted - no-op
	assertFix(t, f,
		"<?php\n/**\n * @covers \\A\\Bar\n * @covers \\B\\Foo\n */\nclass T{}",
		"<?php\n/**\n * @covers \\A\\Bar\n * @covers \\B\\Foo\n */\nclass T{}", false)

	// three lines, case-insensitive ordering
	assertFix(t, f,
		"<?php\n/**\n * @covers \\c\n * @covers \\A\n * @covers \\b\n */\nclass T{}",
		"<?php\n/**\n * @covers \\A\n * @covers \\b\n * @covers \\c\n */\nclass T{}", true)

	// single @covers - no-op
	assertFix(t, f,
		"<?php\n/**\n * @covers \\A\\Bar\n */\nclass T{}",
		"<?php\n/**\n * @covers \\A\\Bar\n */\nclass T{}", false)

	// non-contiguous @covers (a summary line between) - left alone
	assertFix(t, f,
		"<?php\n/**\n * @covers \\B\n * some text\n * @covers \\A\n */\nclass T{}",
		"<?php\n/**\n * @covers \\B\n * some text\n * @covers \\A\n */\nclass T{}", false)

	// @coversNothing must not be treated as @covers
	assertFix(t, f,
		"<?php\n/**\n * @coversNothing\n * @covers \\A\n */\nclass T{}",
		"<?php\n/**\n * @coversNothing\n * @covers \\A\n */\nclass T{}", false)
}

func TestGen2PhpdocSourceURLs(t *testing.T) {
	for _, r := range []fixer.Fixer{PhpdocIndent{}, PhpdocOrderByValue{}} {
		if got, want := r.SourceURL(), fixer.SourceURLFor(r.Name()); got != want {
			t.Fatalf("%T SourceURL %q, want %q", r, got, want)
		}
	}
}
