package rules

import "testing"

func TestPhpdocToComment(t *testing.T) {
	f := PhpdocToComment{}

	// a docblock that documents nothing structural becomes a plain comment
	got, changed := apply(t, f, "<?php\n$first = true;\n\n/** This should be a comment */\nforeach ($c as $k => $v) {\n}\n")
	want := "<?php\n$first = true;\n\n/* This should be a comment */\nforeach ($c as $k => $v) {\n}\n"
	if !changed || got != want {
		t.Fatalf("convert: changed=%v got=%q", changed, got)
	}

	// a docblock naming the loop variable is kept
	if _, changed := apply(t, f, "<?php\n$first = true;\n\n/** @var Foo $v */\nforeach ($c as $k => $v) {\n}\n"); changed {
		t.Fatal("docblock naming the loop var must stay a docblock")
	}

	// a docblock before a class is kept
	if _, changed := apply(t, f, "<?php\n$x = 1;\n\n/** Class doc */\nclass A {}\n"); changed {
		t.Fatal("docblock before a class must stay a docblock")
	}

	// a file header docblock is kept
	if _, changed := apply(t, f, "<?php\n/** file header */\n\n$x = 1;\n"); changed {
		t.Fatal("header docblock must stay a docblock")
	}

	// a docblock naming an assigned variable is kept (documents that variable)
	if _, changed := apply(t, f, "<?php\n$x = 1;\n/** @var int */\n$z = 5;\n"); changed {
		t.Fatal("docblock before a variable assignment must stay a docblock")
	}

	// a docblock before an unrelated statement becomes a comment
	got, changed = apply(t, f, "<?php\n$x = 1;\n/** note */\necho 5;\n")
	want = "<?php\n$x = 1;\n/* note */\necho 5;\n"
	if !changed || got != want {
		t.Fatalf("echo: changed=%v got=%q", changed, got)
	}
}
