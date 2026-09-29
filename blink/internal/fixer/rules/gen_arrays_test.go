package rules

import (
	"testing"

	fixerpkg "blink/internal/fixer"
)

func TestGenArraysNoWhitespaceInEmptyArray(t *testing.T) {
	got, changed := apply(t, NoWhitespaceInEmptyArray{}, "<?php $a = [ ];")
	if want := "<?php $a = [];"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// newline inside the empty array is removed too
	got, changed = apply(t, NoWhitespaceInEmptyArray{}, "<?php $a = [\n];")
	if want := "<?php $a = [];"; !changed || got != want {
		t.Fatalf("newline: changed=%v got=%q want=%q", changed, got, want)
	}
	// already tight: no-op
	if _, changed := apply(t, NoWhitespaceInEmptyArray{}, "<?php $a = [];"); changed {
		t.Fatal("empty array without whitespace must not change")
	}
	// non-empty array is left alone
	if _, changed := apply(t, NoWhitespaceInEmptyArray{}, "<?php $a = [ 1 ];"); changed {
		t.Fatal("non-empty array must not change")
	}
	// long array syntax is out of scope
	if _, changed := apply(t, NoWhitespaceInEmptyArray{}, "<?php $a = array( );"); changed {
		t.Fatal("long array syntax must not change")
	}
}

func TestGenArraysNormalizeIndexBrace(t *testing.T) {
	got, changed := apply(t, NormalizeIndexBrace{}, "<?php echo $sample{$index};")
	if want := "<?php echo $sample[$index];"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// numeric literal index
	got, changed = apply(t, NormalizeIndexBrace{}, "<?php $a{0};")
	if want := "<?php $a[0];"; !changed || got != want {
		t.Fatalf("numeric: changed=%v got=%q want=%q", changed, got, want)
	}
	// chained access, mixed with square brackets
	got, changed = apply(t, NormalizeIndexBrace{}, "<?php $a[0]{1};")
	if want := "<?php $a[0][1];"; !changed || got != want {
		t.Fatalf("chained: changed=%v got=%q want=%q", changed, got, want)
	}
	// already converted: no-op
	if _, changed := apply(t, NormalizeIndexBrace{}, "<?php $a[0];"); changed {
		t.Fatal("square index must not change")
	}
	// real block braces must never be touched
	if _, changed := apply(t, NormalizeIndexBrace{}, "<?php if ($a) { echo 1; }"); changed {
		t.Fatal("control block must not change")
	}
	if _, changed := apply(t, NormalizeIndexBrace{}, "<?php class Foo { public $x; }"); changed {
		t.Fatal("class block must not change")
	}
	// variable-variable "${x}" must not be converted
	if _, changed := apply(t, NormalizeIndexBrace{}, "<?php ${x};"); changed {
		t.Fatal("variable-variable braces must not change")
	}
}

func TestGenArraysNoMultilineWhitespaceAroundDoubleArrow(t *testing.T) {
	got, changed := apply(t, NoMultilineWhitespaceAroundDoubleArrow{}, "<?php $a = [\n    'x' =>\n    1,\n];")
	if want := "<?php $a = [\n    'x' => 1,\n];"; !changed || got != want {
		t.Fatalf("after: changed=%v got=%q want=%q", changed, got, want)
	}
	// newline before the arrow collapses to a single space
	got, changed = apply(t, NoMultilineWhitespaceAroundDoubleArrow{}, "<?php $a = [\n    'x'\n    => 1,\n];")
	if want := "<?php $a = [\n    'x' => 1,\n];"; !changed || got != want {
		t.Fatalf("before: changed=%v got=%q want=%q", changed, got, want)
	}
	// single-line spacing is left alone (that is another fixer's job)
	if _, changed := apply(t, NoMultilineWhitespaceAroundDoubleArrow{}, "<?php $a = ['x'  =>  1];"); changed {
		t.Fatal("single-line spacing must not change")
	}
	// already normalized: no-op
	if _, changed := apply(t, NoMultilineWhitespaceAroundDoubleArrow{}, "<?php $a = ['x' => 1];"); changed {
		t.Fatal("normalized arrow must not change")
	}
	// a comment right after the arrow is respected
	if _, changed := apply(t, NoMultilineWhitespaceAroundDoubleArrow{}, "<?php $a = ['x' =>\n    // c\n    1];"); changed {
		t.Fatal("comment after arrow must be preserved")
	}
}

// TestGenArraysSourceURLs verifies every SourceURL matches SourceURLFor(Name()).
func TestGenArraysSourceURLs(t *testing.T) {
	fixers := []fixerpkg.Fixer{
		NoWhitespaceInEmptyArray{},
		NormalizeIndexBrace{},
		NoMultilineWhitespaceAroundDoubleArrow{},
	}
	for _, f := range fixers {
		if got, want := f.SourceURL(), fixerpkg.SourceURLFor(f.Name()); got != want {
			t.Errorf("%s: SourceURL %q != SourceURLFor %q", f.Name(), got, want)
		}
	}
}

// TestGenArraysIdempotent guarantees a second pass is a no-op.
func TestGenArraysIdempotent(t *testing.T) {
	fixers := []fixerRule{
		NoWhitespaceInEmptyArray{},
		NormalizeIndexBrace{},
		NoMultilineWhitespaceAroundDoubleArrow{},
	}
	corpus := []string{
		"<?php $a = [ ];\n",
		"<?php echo $sample{$index};\n",
		"<?php $a[0]{1};\n",
		"<?php if ($a) { echo 1; }\n",
		"<?php $a = [\n    'x' =>\n    1,\n    'y'\n    => 2,\n];\n",
	}
	for _, f := range fixers {
		for _, src := range corpus {
			once, _ := apply(t, f, src)
			twice, changed := apply(t, f, once)
			if changed || once != twice {
				t.Errorf("%T not idempotent\n src:  %q\n once: %q\n twice:%q", f, src, once, twice)
			}
		}
	}
}
