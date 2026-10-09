package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen3ClassSepMethods(t *testing.T) {
	src := "<?php class A {\n" +
		"    public function a() {}\n" +
		"    public function b() {}\n" +
		"}"
	want := "<?php class A {\n" +
		"    public function a() {}\n" +
		"\n" +
		"    public function b() {}\n" +
		"}"
	got, changed := apply(t, ClassAttributesSeparation{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// already-correct output is a no-op
	if out, changed := apply(t, ClassAttributesSeparation{}, want); changed || out != want {
		t.Fatalf("second run must be a no-op: changed=%v got=%q", changed, out)
	}
}

func TestGen3ClassSepKeywordTypedProperty(t *testing.T) {
	// a property typed with a keyword type ("array") is still a property and must
	// be separated from the previous member by a blank line
	src := "<?php class A {\n" +
		"    private int $a;\n" +
		"    private array $b = [];\n" +
		"}"
	want := "<?php class A {\n" +
		"    private int $a;\n" +
		"\n" +
		"    private array $b = [];\n" +
		"}"
	got, changed := apply(t, ClassAttributesSeparation{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestGen3ClassSepPropertiesAndConsts(t *testing.T) {
	src := "<?php class A {\n" +
		"    const X = 1;\n" +
		"    const Y = 2;\n" +
		"    private $a;\n" +
		"    private $b;\n" +
		"}"
	want := "<?php class A {\n" +
		"    const X = 1;\n" +
		"\n" +
		"    const Y = 2;\n" +
		"\n" +
		"    private $a;\n" +
		"\n" +
		"    private $b;\n" +
		"}"
	got, changed := apply(t, ClassAttributesSeparation{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestGen3ClassSepTraitImportsStayTight(t *testing.T) {
	src := "<?php class A {\n" +
		"    use TraitA;\n" +
		"    use TraitB;\n" +
		"}"
	if got, changed := apply(t, ClassAttributesSeparation{}, src); changed || got != src {
		t.Fatalf("consecutive trait imports must stay tight: changed=%v got=%q", changed, got)
	}
	// a blank line between two trait imports is collapsed
	blank := "<?php class A {\n" +
		"    use TraitA;\n" +
		"\n" +
		"    use TraitB;\n" +
		"}"
	got, changed := apply(t, ClassAttributesSeparation{}, blank)
	if !changed || got != src {
		t.Fatalf("blank between trait imports must collapse: changed=%v got=%q", changed, got)
	}
}

func TestGen3ClassSepEnumCasesStayTight(t *testing.T) {
	src := "<?php enum Suit {\n" +
		"    case Hearts;\n" +
		"    case Spades;\n" +
		"}"
	if got, changed := apply(t, ClassAttributesSeparation{}, src); changed || got != src {
		t.Fatalf("consecutive enum cases must stay tight: changed=%v got=%q", changed, got)
	}
	blank := "<?php enum Suit {\n" +
		"    case Hearts;\n" +
		"\n" +
		"    case Spades;\n" +
		"}"
	got, changed := apply(t, ClassAttributesSeparation{}, blank)
	if !changed || got != src {
		t.Fatalf("blank between enum cases must collapse: changed=%v got=%q", changed, got)
	}
}

func TestGen3ClassSepDocblockStaysAttached(t *testing.T) {
	src := "<?php class A {\n" +
		"    public function a() {}\n" +
		"    /** b */\n" +
		"    public function b() {}\n" +
		"}"
	want := "<?php class A {\n" +
		"    public function a() {}\n" +
		"\n" +
		"    /** b */\n" +
		"    public function b() {}\n" +
		"}"
	got, changed := apply(t, ClassAttributesSeparation{}, src)
	if !changed || got != want {
		t.Fatalf("blank line must go above the docblock: changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// idempotent
	if out, changed := apply(t, ClassAttributesSeparation{}, want); changed || out != want {
		t.Fatalf("docblock case must be idempotent: changed=%v got=%q", changed, out)
	}
}

func TestGen3ClassSepAttributeStaysAttached(t *testing.T) {
	src := "<?php class A {\n" +
		"    public function a() {}\n" +
		"    #[Test]\n" +
		"    public function b() {}\n" +
		"}"
	want := "<?php class A {\n" +
		"    public function a() {}\n" +
		"\n" +
		"    #[Test]\n" +
		"    public function b() {}\n" +
		"}"
	got, changed := apply(t, ClassAttributesSeparation{}, src)
	if !changed || got != want {
		t.Fatalf("blank line must go above the attribute: changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestGen3ClassSepLeadingComment(t *testing.T) {
	// a line comment leading the next member is its trivia: the blank line goes
	// above the comment, matching ECS
	src := "<?php class A {\n" +
		"    public function a() {}\n" +
		"    // note\n" +
		"    public function b() {}\n" +
		"}"
	want := "<?php class A {\n" +
		"    public function a() {}\n" +
		"\n" +
		"    // note\n" +
		"    public function b() {}\n" +
		"}"
	if got, changed := apply(t, ClassAttributesSeparation{}, src); !changed || got != want {
		t.Fatalf("blank line must go above a leading comment: changed=%v got=%q", changed, got)
	}
}

func TestGen3ClassSepNestedBodyUntouched(t *testing.T) {
	src := "<?php class A {\n" +
		"    public function a() {\n" +
		"        $x = 1;\n" +
		"\n" +
		"        $y = 2;\n" +
		"    }\n" +
		"    public function b() {}\n" +
		"}"
	want := "<?php class A {\n" +
		"    public function a() {\n" +
		"        $x = 1;\n" +
		"\n" +
		"        $y = 2;\n" +
		"    }\n" +
		"\n" +
		"    public function b() {}\n" +
		"}"
	got, changed := apply(t, ClassAttributesSeparation{}, src)
	if !changed || got != want {
		t.Fatalf("only the inter-member gap must change: changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
}

func TestGen3ClassSepGluedMembersLeftAlone(t *testing.T) {
	// members on one line are not reformatted (conservative)
	src := "<?php class A { public $a; public $b; }"
	if got, changed := apply(t, ClassAttributesSeparation{}, src); changed || got != src {
		t.Fatalf("single-line members must be left alone: changed=%v got=%q", changed, got)
	}
}

func TestGen3ClassSepSourceURL(t *testing.T) {
	f := ClassAttributesSeparation{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
	}
}
