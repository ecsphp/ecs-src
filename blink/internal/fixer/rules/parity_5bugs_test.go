package rules

import "testing"

func TestArrowFnSpacing(t *testing.T) {
	got, changed := apply(t, FunctionDeclaration{}, "<?php $f = fn(int $x) => $x;")
	if want := "<?php $f = fn (int $x) => $x;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, FunctionDeclaration{}, "<?php $f = fn (int $x) => $x;"); changed {
		t.Fatal("already-spaced fn must not change")
	}
}

func TestPhpdocTypesGenericsAndConst(t *testing.T) {
	// a pseudo-type keyword inside generics is lowercased
	got, changed := apply(t, PhpdocTypes{}, "<?php\n/**\n * @implements Foo<Scalar>\n */\nclass A {}")
	if want := "<?php\n/**\n * @implements Foo<scalar>\n */\nclass A {}"; !changed || got != want {
		t.Fatalf("generics: changed=%v got=%q", changed, got)
	}
	// a class-constant reference (Ref::STATIC) is left untouched
	if _, changed := apply(t, PhpdocTypes{}, "<?php\n/**\n * @return Ref::STATIC|Ref::SELF\n */\nfunction f() {}"); changed {
		t.Fatal("class-constant references must not be lowercased")
	}
}

func TestNoBlankLinesAfterPhpdocKeepsFileDocblock(t *testing.T) {
	// a file-level docblock before "declare" keeps its blank line
	src := "<?php\n\n/**\n * file header\n */\n\ndeclare(strict_types=1);\n"
	if got, changed := apply(t, NoBlankLinesAfterPhpdoc{}, src); changed || got != src {
		t.Fatalf("file docblock blank must be kept: changed=%v got=%q", changed, got)
	}
	// a docblock attached to a class still has its trailing blank removed
	got, changed := apply(t, NoBlankLinesAfterPhpdoc{}, "<?php\n/**\n * x\n */\n\nclass A {}")
	if want := "<?php\n/**\n * x\n */\nclass A {}"; !changed || got != want {
		t.Fatalf("attached docblock: changed=%v got=%q", changed, got)
	}
}

func TestPhpdocIndentAlignsToDocumentedElement(t *testing.T) {
	// docblock mis-indented deeper than its method is realigned to the method
	src := "<?php\nclass A\n{\n        /**\n     * @return int\n     */\n    public function f(): int\n    {\n        return 1;\n    }\n}"
	want := "<?php\nclass A\n{\n    /**\n     * @return int\n     */\n    public function f(): int\n    {\n        return 1;\n    }\n}"
	got, changed := apply(t, PhpdocIndent{}, src)
	if !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}
