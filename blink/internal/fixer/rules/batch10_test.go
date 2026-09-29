package rules

import "testing"

func TestTrailingCommaInMultiline(t *testing.T) {
	got, changed := apply(t, TrailingCommaInMultiline{}, "<?php $a = [\n    1,\n    2\n];")
	if want := "<?php $a = [\n    1,\n    2,\n];"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// single-line array unchanged; comment stays the last thing before ]
	if _, changed := apply(t, TrailingCommaInMultiline{}, "<?php $a = [1, 2];"); changed {
		t.Fatal("single-line array must not gain a comma")
	}
	if _, changed := apply(t, TrailingCommaInMultiline{}, "<?php $a = [\n    1,\n    // note\n];"); changed {
		t.Fatal("comma must not go after a trailing comment")
	}
	// offset access is not an array literal
	if _, changed := apply(t, TrailingCommaInMultiline{}, "<?php $x = $a[\n    0\n];"); changed {
		t.Fatal("offset access must not gain a comma")
	}
}

func TestNoTrailingCommaInSingleline(t *testing.T) {
	got, changed := apply(t, NoTrailingCommaInSingleline{}, "<?php $a = [1, 2,]; foo(1,);")
	if want := "<?php $a = [1, 2]; foo(1);"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// multi-line trailing comma stays
	if _, changed := apply(t, NoTrailingCommaInSingleline{}, "<?php $a = [\n    1,\n];"); changed {
		t.Fatal("multi-line trailing comma must stay")
	}
}

func TestNoSpacesAroundOffset(t *testing.T) {
	got, changed := apply(t, NoSpacesAroundOffset{}, "<?php echo $a[ 0 ][ 'k' ];")
	if want := "<?php echo $a[0]['k'];"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// array literal is not an offset
	if _, changed := apply(t, NoSpacesAroundOffset{}, "<?php $a = [ 1, 2 ];"); changed {
		t.Fatal("array literal spaces are not offset spaces")
	}
}

func TestObjectOperatorWithoutWhitespace(t *testing.T) {
	got, changed := apply(t, ObjectOperatorWithoutWhitespace{}, "<?php $a -> b -> c();")
	if want := "<?php $a->b->c();"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNativeFunctionCasing(t *testing.T) {
	got, changed := apply(t, NativeFunctionCasing{}, "<?php echo STRLEN($x) . Count($y);")
	if want := "<?php echo strlen($x) . count($y);"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// a method named like a native function is not touched
	if _, changed := apply(t, NativeFunctionCasing{}, "<?php $o->Count();"); changed {
		t.Fatal("method call must not be lowercased")
	}
}

func TestIntegerLiteralCase(t *testing.T) {
	got, changed := apply(t, IntegerLiteralCase{}, "<?php $a = 0XFF; $b = 0B101;")
	if want := "<?php $a = 0xFF; $b = 0b101;"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, IntegerLiteralCase{}, "<?php $a = 255;"); changed {
		t.Fatal("decimal literal must not change")
	}
}

func TestNoEmptyComment(t *testing.T) {
	got, changed := apply(t, NoEmptyComment{}, "<?php //\n$a = 1; /*  */")
	if want := "<?php \n$a = 1; "; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// attribute must not be removed
	if _, changed := apply(t, NoEmptyComment{}, "<?php #[Attr]\nclass A {}"); changed {
		t.Fatal("attribute must not be removed as an empty comment")
	}
}

func TestSingleLineCommentSpacing(t *testing.T) {
	got, changed := apply(t, SingleLineCommentSpacing{}, "<?php //foo\n#bar")
	if want := "<?php // foo\n# bar"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	if _, changed := apply(t, SingleLineCommentSpacing{}, "<?php #[Attr]"); changed {
		t.Fatal("attribute must not gain a space")
	}
}

func TestNoUnusedImports(t *testing.T) {
	got, changed := apply(t, NoUnusedImports{}, "<?php\nuse App\\Used;\nuse App\\Unused;\n\n$x = new Used();\n")
	if want := "<?php\nuse App\\Used;\n\n$x = new Used();\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// an import referenced only in a doc comment is kept
	if _, changed := apply(t, NoUnusedImports{}, "<?php\nuse App\\Thing;\n\n/** @return Thing */\nfunction f() {}\n"); changed {
		t.Fatal("import used in a doc comment must be kept")
	}
}
