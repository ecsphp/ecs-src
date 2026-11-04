package rules

import "testing"

func TestClassDefinitionNormalizesSpaces(t *testing.T) {
	t.Parallel()
	// the opening brace of a named class goes on the next line (PHP-CS-Fixer)
	got, changed := apply(t, ClassDefinition{}, "<?php class  A  extends  B  implements  C {}")
	want := "<?php class A extends B implements C\n{}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestClassDefinitionLeavesCorrectHeader(t *testing.T) {
	t.Parallel()
	src := "<?php class A extends B implements C\n{}"
	got, changed := apply(t, ClassDefinition{}, src)
	if changed || got != src {
		t.Fatalf("correct header must be untouched: changed=%v got=%q", changed, got)
	}
}

func TestClassDefinitionKeepsNewlines(t *testing.T) {
	t.Parallel()
	// a multiline implements list keeps its layout
	src := "<?php class A implements\n    B,\n    C\n{}"
	got, changed := apply(t, ClassDefinition{}, src)
	if changed || got != src {
		t.Fatalf("multiline header must be untouched: changed=%v got=%q", changed, got)
	}
}

func TestClassDefinitionIgnoresClassConstant(t *testing.T) {
	t.Parallel()
	if _, changed := apply(t, ClassDefinition{}, "<?php $x = Foo::class;"); changed {
		t.Fatal("::class must not be treated as a class header")
	}
}

func TestOrderedClassElementsSimple(t *testing.T) {
	t.Parallel()
	src := "<?php class A {\n" +
		"    public function run() {}\n" +
		"    public $prop;\n" +
		"    const X = 1;\n" +
		"    use SomeTrait;\n" +
		"}"
	want := "<?php class A {\n" +
		"    use SomeTrait;\n" +
		"    const X = 1;\n" +
		"    public $prop;\n" +
		"    public function run() {}\n" +
		"}"
	got, changed := apply(t, OrderedClassElements{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// idempotent: running on the sorted output changes nothing
	if out, changed := apply(t, OrderedClassElements{}, want); changed || out != want {
		t.Fatalf("second run must be a no-op: changed=%v got=%q", changed, out)
	}
}

func TestOrderedClassElementsOrderUseTraitOnly(t *testing.T) {
	t.Parallel()
	// order ["use_trait"] groups trait uses first and leaves every other member
	// in its source order (const stays after the property here)
	src := "<?php class A {\n" +
		"    public $prop;\n" +
		"    const X = 1;\n" +
		"    use SomeTrait;\n" +
		"}"
	want := "<?php class A {\n" +
		"    use SomeTrait;\n" +
		"    public $prop;\n" +
		"    const X = 1;\n" +
		"}"
	f := OrderedClassElements{}.WithConfig(map[string]any{"order": []any{"use_trait"}}).(fixerRule)
	got, changed := apply(t, f, src)
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// a class with the property before the const and no trait is left untouched
	noTrait := "<?php class B {\n    public $prop;\n    const X = 1;\n}"
	if out, changed := apply(t, f, noTrait); changed || out != noTrait {
		t.Fatalf("no-trait class must be a no-op: changed=%v got=%q", changed, out)
	}
}

func TestOrderedClassElementsAlreadyOrdered(t *testing.T) {
	t.Parallel()
	src := "<?php class A {\n" +
		"    const X = 1;\n" +
		"    public $prop;\n" +
		"    public function run() {}\n" +
		"}"
	got, changed := apply(t, OrderedClassElements{}, src)
	if changed || got != src {
		t.Fatalf("ordered class must be byte-identical: changed=%v got=%q", changed, got)
	}
}

// The comments, docblocks and attributes before an element move with it, and
// enum cases sort before methods (PHP-CS-Fixer).
func TestOrderedClassElementsMovesLeadingTrivia(t *testing.T) {
	t.Parallel()
	src := "<?php class A {\n    public function run() {}\n    /** the answer */\n    const X = 1;\n}\nclass B {\n    public function run() {}\n    #[Column]\n    public $prop;\n}\nenum Suit {\n    public function label() {}\n    case Hearts;\n}\n"
	want := "<?php class A {\n    /** the answer */\n    const X = 1;\n    public function run() {}\n}\nclass B {\n    #[Column]\n    public $prop;\n    public function run() {}\n}\nenum Suit {\n    case Hearts;\n    public function label() {}\n}\n"
	if got, _ := apply(t, OrderedClassElements{}, src); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
