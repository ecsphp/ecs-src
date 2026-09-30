package rules

import (
	"testing"

	"blink/internal/fixer"
)

// TestDoubleAsteriskInlineVar promotes a single-asterisk inline /* @var */ to a docblock.
func TestDoubleAsteriskInlineVar(t *testing.T) {
	got, changed := apply(t, DoubleAsteriskInlineVar{}, "<?php\nfunction f() {\n    /* @var int $x */\n    return $x;\n}\n")
	if want := "<?php\nfunction f() {\n    /** @var int $x */\n    return $x;\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, DoubleAsteriskInlineVar{}, got)

	if _, changed := apply(t, DoubleAsteriskInlineVar{}, "<?php\n/** @var int $x */\n$x = 1;\n"); changed {
		t.Fatal("already-double-asterisk must not change")
	}
}

// TestFixTagTypo rewrites a plural @returns tag to @return.
func TestFixTagTypo(t *testing.T) {
	got, changed := apply(t, FixTagTypo{}, "<?php\nclass A {\n    /**\n     * @returns int\n     */\n    function f() {}\n}\n")
	if want := "<?php\nclass A {\n    /**\n     * @return int\n     */\n    function f() {}\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, FixTagTypo{}, got)
}

// TestTypeToVarTag rewrites a @type property tag to @var.
func TestTypeToVarTag(t *testing.T) {
	got, changed := apply(t, TypeToVarTag{}, "<?php\nclass A {\n    /** @type int */\n    public $x;\n}\n")
	if want := "<?php\nclass A {\n    /** @var int */\n    public $x;\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, TypeToVarTag{}, got)
}

// TestTypeToVarTagSkipsDescription leaves "@type" inside a description untouched,
// only rewriting it at the start of a doc line (mirrors #81).
func TestTypeToVarTagSkipsDescription(t *testing.T) {
	src := "<?php\nclass A {\n    /**\n     * Matches only \"@type\" at the start of a line\n     */\n    public $x;\n}\n"
	if got, changed := apply(t, TypeToVarTag{}, src); changed || got != src {
		t.Fatalf("@type in a description must not change: %q", got)
	}
}

// TestMergeDocBlockStart folds an empty leading line into the /** opener.
func TestMergeDocBlockStart(t *testing.T) {
	got, changed := apply(t, MergeDocBlockStart{}, "<?php\nclass A {\n    /*\n     *\n     * @var int\n     */\n    private $x;\n}\n")
	if want := "<?php\nclass A {\n    /**\n     * @var int\n     */\n    private $x;\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, MergeDocBlockStart{}, got)
}

// TestAddMissingVarName appends the variable name to a bare @var docblock.
func TestAddMissingVarName(t *testing.T) {
	got, changed := apply(t, AddMissingVarName{}, "<?php\nfunction f() {\n    /** @var int */\n    $x = 1;\n    return $x;\n}\n")
	if want := "<?php\nfunction f() {\n    /** @var int $x */\n    $x = 1;\n    return $x;\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, AddMissingVarName{}, got)
}

// TestSingleLineInlineVarDocBlock collapses a multi-line inline @var to one line.
func TestSingleLineInlineVarDocBlock(t *testing.T) {
	got, changed := apply(t, SingleLineInlineVarDocBlock{}, "<?php\nfunction f() {\n    /**\n     * @var int\n     */\n    $x = 1;\n    return $x;\n}\n")
	if want := "<?php\nfunction f() {\n    /** @var int */\n    $x = 1;\n    return $x;\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, SingleLineInlineVarDocBlock{}, got)
}

// TestRemoveSuperfluousReturnName drops the variable name after a @return type.
func TestRemoveSuperfluousReturnName(t *testing.T) {
	got, changed := apply(t, RemoveSuperfluousReturnName{}, "<?php\nclass A {\n    /**\n     * @return int $result\n     */\n    function f() {}\n}\n")
	if want := "<?php\nclass A {\n    /**\n     * @return int\n     */\n    function f() {}\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, RemoveSuperfluousReturnName{}, got)
}

// TestRemoveSuperfluousVarName drops the variable name from a property @var.
func TestRemoveSuperfluousVarName(t *testing.T) {
	got, changed := apply(t, RemoveSuperfluousVarName{}, "<?php\nclass A {\n    /**\n     * @var int $count\n     */\n    private $count;\n}\n")
	if want := "<?php\nclass A {\n    /**\n     * @var int\n     */\n    private $count;\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, RemoveSuperfluousVarName{}, got)
}

// TestFixParamNameTypo corrects a @param name to match the real argument.
func TestFixParamNameTypo(t *testing.T) {
	got, changed := apply(t, FixParamNameTypo{}, "<?php\nclass A {\n    /**\n     * @param int $bar\n     */\n    function f($foo) {}\n}\n")
	if want := "<?php\nclass A {\n    /**\n     * @param int $foo\n     */\n    function f($foo) {}\n}\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	idempotent(t, FixParamNameTypo{}, got)

	// a @param inside a // line comment is not a real annotation and is left alone
	src := "<?php\nclass A {\n    // e.g. \"@param $a Can be used\"\n    function isKnownType($type) {}\n}\n"
	if got, changed := apply(t, FixParamNameTypo{}, src); changed || got != src {
		t.Fatalf("line-comment @param must be left alone: changed=%v got=%q", changed, got)
	}
}

// TestSymplifyDocBlockSourceURLs verifies each SourceURL matches SourceURLFor(Name()).
func TestSymplifyDocBlockSourceURLs(t *testing.T) {
	fixers := []fixer.Fixer{
		DoubleAsteriskInlineVar{}, FixTagTypo{}, TypeToVarTag{}, MergeDocBlockStart{},
		AddMissingVarName{}, SingleLineInlineVarDocBlock{}, RemoveSuperfluousReturnName{},
		RemoveSuperfluousVarName{}, RemoveParamDescriptionDuplicateName{}, FixParamNameTypo{},
	}
	for _, f := range fixers {
		if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}
