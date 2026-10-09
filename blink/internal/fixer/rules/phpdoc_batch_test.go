package rules

import (
	"testing"

	"blink/internal/fixer"
)

// idempotent asserts that a second Fix pass over the fixer's own output is a no-op.
func idempotent(t *testing.T, r fixerRule, once string) {
	t.Helper()
	if _, changed := apply(t, r, once); changed {
		t.Fatalf("not idempotent, second pass changed: %q", once)
	}
}

func TestNoEmptyPhpdoc(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoEmptyPhpdoc{}, "<?php\n/**\n */\nfunction f() {}")
	if want := "<?php\nfunction f() {}"; !changed || got != want {
		t.Fatalf("multi: changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, NoEmptyPhpdoc{}, got)

	got, changed = apply(t, NoEmptyPhpdoc{}, "<?php\n/** */\n$x = 1;")
	if want := "<?php\n$x = 1;"; !changed || got != want {
		t.Fatalf("single: changed=%v got=%q want=%q", changed, got, want)
	}

	// a docblock with content is kept
	keep := "<?php\n/**\n * Summary.\n */\nfunction f() {}"
	if _, changed := apply(t, NoEmptyPhpdoc{}, keep); changed {
		t.Fatal("non-empty docblock must be kept")
	}
}

func TestPhpdocTypes(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * @param Array $a\n * @param NULL|STRING $b\n * @return VOID\n */\nfunction f($a, $b) {}"
	got, changed := apply(t, PhpdocTypes{}, src)
	want := "<?php\n/**\n * @param array $a\n * @param null|string $b\n * @return void\n */\nfunction f($a, $b) {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, PhpdocTypes{}, got)

	// array suffix and nullable are handled
	got, _ = apply(t, PhpdocTypes{}, "<?php\n/**\n * @param INT[]|Null $a\n * @var ?BOOL $b\n */\nfunction f($a) {}")
	if want := "<?php\n/**\n * @param int[]|null $a\n * @var ?bool $b\n */\nfunction f($a) {}"; got != want {
		t.Fatalf("suffix/nullable: got %q", got)
	}

	// $this keyword is normalized
	got, _ = apply(t, PhpdocTypes{}, "<?php\n/**\n * @return $This\n */\nfunction f() {}")
	if want := "<?php\n/**\n * @return $this\n */\nfunction f() {}"; got != want {
		t.Fatalf("$this: got %q", got)
	}

	// a class name that is not a keyword is untouched
	if _, changed := apply(t, PhpdocTypes{}, "<?php\n/**\n * @param MyClass $a\n */\nfunction f($a) {}"); changed {
		t.Fatal("non-keyword class name must not change")
	}
	// prose is not touched
	if _, changed := apply(t, PhpdocTypes{}, "<?php\n/**\n * Returns an Array of things.\n */\nfunction f() {}"); changed {
		t.Fatal("prose must not change")
	}
}

func TestPhpdocNoAliasTag(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * @type int $a\n * @link https://example.com docs\n */\nfunction f() {}"
	got, changed := apply(t, PhpdocNoAliasTag{}, src)
	want := "<?php\n/**\n * @var int $a\n * @see https://example.com docs\n */\nfunction f() {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, PhpdocNoAliasTag{}, got)

	// prose mentioning the words is not touched, and a longer tag is safe
	if _, changed := apply(t, PhpdocNoAliasTag{}, "<?php\n/**\n * See the @typedef and the link below.\n */\nfunction f() {}"); changed {
		t.Fatal("prose and unrelated tags must not change")
	}
}

func TestPhpdocNoPackage(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * @package App\n * @subpackage Sub\n * @var int\n */\nfunction f() {}"
	got, changed := apply(t, PhpdocNoPackage{}, src)
	want := "<?php\n/**\n * @var int\n */\nfunction f() {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, PhpdocNoPackage{}, got)

	if _, changed := apply(t, PhpdocNoPackage{}, "<?php\n/**\n * @var int\n */\nfunction f() {}"); changed {
		t.Fatal("no package tag must not change")
	}
}

func TestPhpdocNoAccess(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * @access private\n * @var int\n */\nfunction f() {}"
	got, changed := apply(t, PhpdocNoAccess{}, src)
	want := "<?php\n/**\n * @var int\n */\nfunction f() {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, PhpdocNoAccess{}, got)
}

func TestPhpdocSingleLineVarSpacing(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, PhpdocSingleLineVarSpacing{}, "<?php\n/**@var int$x*/\n$x = 1;")
	if want := "<?php\n/** @var int $x */\n$x = 1;"; !changed || got != want {
		t.Fatalf("glued: changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, PhpdocSingleLineVarSpacing{}, got)

	got, _ = apply(t, PhpdocSingleLineVarSpacing{}, "<?php\n/**  @var   MyClass   $a   */\n$a = 1;")
	if want := "<?php\n/** @var MyClass $a */\n$a = 1;"; got != want {
		t.Fatalf("runs: got %q", got)
	}

	// @param with a description, whitespace runs collapsed
	got, _ = apply(t, PhpdocSingleLineVarSpacing{}, "<?php\n/** @param string $name  the   name */\nfunction f($name) {}")
	if want := "<?php\n/** @param string $name the name */\nfunction f($name) {}"; got != want {
		t.Fatalf("param desc: got %q", got)
	}

	// already correct is a no-op
	if _, changed := apply(t, PhpdocSingleLineVarSpacing{}, "<?php\n/** @var int $x */\n$x = 1;"); changed {
		t.Fatal("normalized single-line must not change")
	}
	// multi-line docblock is left alone
	if _, changed := apply(t, PhpdocSingleLineVarSpacing{}, "<?php\n/**\n * @var int $x\n */\n$x = 1;"); changed {
		t.Fatal("multi-line docblock must not change")
	}
}

func TestPhpdocTrimConsecutiveBlankLineSeparation(t *testing.T) {
	t.Parallel()
	src := "<?php\n/**\n * Summary.\n *\n *\n *\n * @param int $a\n */\nfunction f($a) {}"
	got, changed := apply(t, PhpdocTrimConsecutiveBlankLineSeparation{}, src)
	want := "<?php\n/**\n * Summary.\n *\n * @param int $a\n */\nfunction f($a) {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	idempotent(t, PhpdocTrimConsecutiveBlankLineSeparation{}, got)

	// a single blank separator is kept
	if _, changed := apply(t, PhpdocTrimConsecutiveBlankLineSeparation{}, want); changed {
		t.Fatal("single blank line must be kept")
	}
}

func TestNoBlankLinesAfterPhpdoc(t *testing.T) {
	t.Parallel()
	got, changed := apply(t, NoBlankLinesAfterPhpdoc{}, "<?php\n/**\n * Bar.\n */\n\n\nclass Bar {}")
	if want := "<?php\n/**\n * Bar.\n */\nclass Bar {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, NoBlankLinesAfterPhpdoc{}, got)

	// indentation of the documented element is preserved
	got, _ = apply(t, NoBlankLinesAfterPhpdoc{}, "<?php\nclass A\n{\n    /**\n     * doc\n     */\n\n    public $x;\n}")
	if want := "<?php\nclass A\n{\n    /**\n     * doc\n     */\n    public $x;\n}"; got != want {
		t.Fatalf("indent: got %q", got)
	}

	// no blank line means no change
	if _, changed := apply(t, NoBlankLinesAfterPhpdoc{}, "<?php\n/**\n * doc\n */\nclass Bar {}"); changed {
		t.Fatal("docblock directly above code must not change")
	}

	// a floating @var docblock followed by a blank line then a comment is left
	// unchanged - PHP skips the fix when the next token is a comment
	src := "<?php\nif (file_exists($f)) {\n    include $f;\n    /** @var array<string, mixed> $parameters */\n\n    // override default with forced\n    $compiled = array_merge($compiled, $parameters);\n}"
	if got, changed := apply(t, NoBlankLinesAfterPhpdoc{}, src); changed || got != src {
		t.Fatalf("blank before comment must not change: changed=%v got=%q", changed, got)
	}
}

// TestPhpdocBatchSourceURLs verifies every SourceURL matches SourceURLFor(Name()).
func TestPhpdocBatchSourceURLs(t *testing.T) {
	t.Parallel()
	fixers := []fixer.Fixer{
		NoEmptyPhpdoc{},
		PhpdocTypes{},
		PhpdocNoAliasTag{},
		PhpdocNoPackage{},
		PhpdocNoAccess{},
		PhpdocSingleLineVarSpacing{},
		PhpdocTrimConsecutiveBlankLineSeparation{},
		NoBlankLinesAfterPhpdoc{},
	}
	for _, f := range fixers {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}
