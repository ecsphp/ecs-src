package rules

import "testing"

func TestPhpdocTrim(t *testing.T) {
	src := "<?php\n/**\n *\n * Summary.\n *\n */\nfunction f() {}"
	got, changed := apply(t, PhpdocTrim{}, src)
	want := "<?php\n/**\n * Summary.\n */\nfunction f() {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// already-trimmed docblock is untouched
	if _, changed := apply(t, PhpdocTrim{}, want); changed {
		t.Fatal("trimmed docblock must not change")
	}
}

func TestPhpdocNoEmptyReturn(t *testing.T) {
	src := "<?php\n/**\n * Do it.\n *\n * @return void\n */\nfunction f() {}"
	got, changed := apply(t, PhpdocNoEmptyReturn{}, src)
	want := "<?php\n/**\n * Do it.\n *\n */\nfunction f() {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// a real return type is kept
	keep := "<?php\n/**\n * @return int\n */\nfunction f() {}"
	if _, changed := apply(t, PhpdocNoEmptyReturn{}, keep); changed {
		t.Fatal("@return int must be kept")
	}
}

func TestPhpdocScalar(t *testing.T) {
	src := "<?php\n/**\n * @param integer $a\n * @param boolean|null $b\n * @return double\n */\nfunction f($a, $b) {}"
	got, changed := apply(t, PhpdocScalar{}, src)
	want := "<?php\n/**\n * @param int $a\n * @param bool|null $b\n * @return float\n */\nfunction f($a, $b) {}"
	if !changed || got != want {
		t.Fatalf("changed=%v\n got: %q\nwant: %q", changed, got, want)
	}
	// a description mentioning "integer" is not a type and stays
	keep := "<?php\n/**\n * This returns an integer value.\n */\nfunction f() {}"
	if _, changed := apply(t, PhpdocScalar{}, keep); changed {
		t.Fatal("prose 'integer' must not change")
	}
}

func TestPhpdocScalarEdgeCases(t *testing.T) {
	// a class named like an alias (Laravel's Str) must not be touched
	if _, changed := apply(t, PhpdocScalar{}, "<?php\n/**\n * @param Str $a\n */\nfunction f($a) {}"); changed {
		t.Fatal("class name Str must not become string")
	}
	// array suffix is normalized
	got, _ := apply(t, PhpdocScalar{}, "<?php\n/**\n * @param integer[]|null $a\n */\nfunction f($a) {}")
	if want := "<?php\n/**\n * @param int[]|null $a\n */\nfunction f($a) {}"; got != want {
		t.Fatalf("array suffix: got %q", got)
	}
	// single-line docblock
	got, _ = apply(t, PhpdocScalar{}, "<?php\n/** @var integer $x */\n$x = 1;")
	if want := "<?php\n/** @var int $x */\n$x = 1;"; got != want {
		t.Fatalf("single-line: got %q", got)
	}
}

func TestPhpdocNoEmptyReturnKeepsUnion(t *testing.T) {
	// "null" as part of a union is a real type, not an empty return
	for _, src := range []string{
		"<?php\n/**\n * @return null|string\n */\nfunction f() {}",
		"<?php\n/**\n * @return void|int\n */\nfunction f() {}",
	} {
		if _, changed := apply(t, PhpdocNoEmptyReturn{}, src); changed {
			t.Fatalf("union return must be kept: %q", src)
		}
	}
}
