package rules

import "testing"

func TestClassDefinitionNormalizesSpaces(t *testing.T) {
	got, changed := apply(t, ClassDefinition{}, "<?php class  A  extends  B  implements  C {}")
	want := "<?php class A extends B implements C {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestClassDefinitionLeavesCorrectHeader(t *testing.T) {
	src := "<?php class A extends B implements C {}"
	got, changed := apply(t, ClassDefinition{}, src)
	if changed || got != src {
		t.Fatalf("correct header must be untouched: changed=%v got=%q", changed, got)
	}
}

func TestClassDefinitionKeepsNewlines(t *testing.T) {
	// a multiline implements list keeps its layout
	src := "<?php class A implements\n    B,\n    C\n{}"
	got, changed := apply(t, ClassDefinition{}, src)
	if changed || got != src {
		t.Fatalf("multiline header must be untouched: changed=%v got=%q", changed, got)
	}
}

func TestClassDefinitionIgnoresClassConstant(t *testing.T) {
	if _, changed := apply(t, ClassDefinition{}, "<?php $x = Foo::class;"); changed {
		t.Fatal("::class must not be treated as a class header")
	}
}

func TestOrderedClassElementsSimple(t *testing.T) {
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

func TestOrderedClassElementsAlreadyOrdered(t *testing.T) {
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

func TestOrderedClassElementsSkipsCommentsAndAttributes(t *testing.T) {
	// a doc comment between members is trivia we do not move; skip the class
	docSrc := "<?php class A {\n" +
		"    public function run() {}\n" +
		"    /** the answer */\n" +
		"    const X = 1;\n" +
		"}"
	if got, changed := apply(t, OrderedClassElements{}, docSrc); changed || got != docSrc {
		t.Fatalf("class with a doc comment must be untouched: changed=%v got=%q", changed, got)
	}
	// #[Attr] lexes as a line comment here, so it too forces a skip
	attrSrc := "<?php class A {\n" +
		"    public function run() {}\n" +
		"    #[Column]\n" +
		"    public $prop;\n" +
		"}"
	if got, changed := apply(t, OrderedClassElements{}, attrSrc); changed || got != attrSrc {
		t.Fatalf("class with an attribute must be untouched: changed=%v got=%q", changed, got)
	}
}

func TestOrderedClassElementsSkipsEnumCases(t *testing.T) {
	src := "<?php enum Suit {\n" +
		"    public function label() {}\n" +
		"    case Hearts;\n" +
		"}"
	if got, changed := apply(t, OrderedClassElements{}, src); changed || got != src {
		t.Fatalf("enum with cases must be untouched: changed=%v got=%q", changed, got)
	}
}
