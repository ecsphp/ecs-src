package rules

import "testing"

func TestVisibilityRequired(t *testing.T) {
	got, changed := apply(t, VisibilityRequired{}, "<?php class A {\n    function run() {}\n    const X = 1;\n    static $s;\n    var $old;\n    public $ok;\n}")
	want := "<?php class A {\n    public function run() {}\n    public const X = 1;\n    public static $s;\n    public $old;\n    public $ok;\n}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// idempotent
	if _, changed := apply(t, VisibilityRequired{}, want); changed {
		t.Fatal("already-explicit visibility should not change")
	}
}

func TestVisibilityRequiredTypedProperty(t *testing.T) {
	got, changed := apply(t, VisibilityRequired{}, "<?php class A {\n    int $count;\n    ?string $name;\n}")
	want := "<?php class A {\n    public int $count;\n    public ?string $name;\n}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestVisibilityRequiredSkipsTraitAndEnumCase(t *testing.T) {
	if _, changed := apply(t, VisibilityRequired{}, "<?php class A {\n    use SomeTrait;\n}"); changed {
		t.Fatal("trait use must not get visibility")
	}
	if _, changed := apply(t, VisibilityRequired{}, "<?php enum Suit {\n    case Hearts;\n    case Spades;\n}"); changed {
		t.Fatal("enum cases must not get visibility")
	}
}

func TestVisibilityRequiredIgnoresPromotedParams(t *testing.T) {
	// constructor property promotion lives inside (), not the class body
	src := "<?php class A {\n    public function __construct(private int $x) {}\n}"
	if _, changed := apply(t, VisibilityRequired{}, src); changed {
		t.Fatalf("promoted params must not be treated as members: %q", src)
	}
}

func TestSingleTraitInsertPerStatement(t *testing.T) {
	got, changed := apply(t, SingleTraitInsertPerStatement{}, "<?php class A {\n    use TraitB, TraitC;\n}")
	want := "<?php class A {\n    use TraitB;\n    use TraitC;\n}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// top-level imports are NOT this fixer's job
	if _, changed := apply(t, SingleTraitInsertPerStatement{}, "<?php\nuse A, B;\n"); changed {
		t.Fatal("top-level import must be left to single_import_per_statement")
	}
}
