package rules

import "testing"

func TestLinebreakAfterOpeningTag(t *testing.T) {
	got, changed := apply(t, LinebreakAfterOpeningTag{}, "<?php $a = 1;\necho $a;\n")
	if want := "<?php\n$a = 1;\necho $a;\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// already on its own line: no-op
	if _, changed := apply(t, LinebreakAfterOpeningTag{}, "<?php\n$a = 1;\n"); changed {
		t.Fatal("code already on a new line must not change")
	}
	// bare "<?php ?>" (closing tag) is left alone
	if _, changed := apply(t, LinebreakAfterOpeningTag{}, "<?php ?>"); changed {
		t.Fatal("lone open/close tag must not change")
	}
}

func TestNullableTypeDeclaration(t *testing.T) {
	got, changed := apply(t, NullableTypeDeclaration{}, "<?php function a(int|null $x): string|null {}")
	if want := "<?php function a(?int $x): ?string {}"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// null first
	got, changed = apply(t, NullableTypeDeclaration{}, "<?php function b(null|int $y) {}")
	if want := "<?php function b(?int $y) {}"; !changed || got != want {
		t.Fatalf("null-first: changed=%v got=%q want=%q", changed, got, want)
	}
	// three-member union is left alone (not a single nullable)
	if _, changed := apply(t, NullableTypeDeclaration{}, "<?php function c(int|string|null $z) {}"); changed {
		t.Fatal("three-member union must not collapse to ?")
	}
	// already question-mark form: no-op
	if _, changed := apply(t, NullableTypeDeclaration{}, "<?php function d(?int $x) {}"); changed {
		t.Fatal("already ?int must not change")
	}
}

func TestOrderedTypes(t *testing.T) {
	got, changed := apply(t, OrderedTypes{}, "<?php function a(): B|A {}")
	if want := "<?php function a(): A|B {}"; !changed || got != want {
		t.Fatalf("return: changed=%v got=%q want=%q", changed, got, want)
	}
	// null first, case-insensitive alpha
	got, changed = apply(t, OrderedTypes{}, "<?php class C { public string|int|null $q; }")
	if want := "<?php class C { public null|int|string $q; }"; !changed || got != want {
		t.Fatalf("null-first: changed=%v got=%q want=%q", changed, got, want)
	}
	// intersection sorted too
	got, changed = apply(t, OrderedTypes{}, "<?php class C { private Foo&Bar $p; }")
	if want := "<?php class C { private Bar&Foo $p; }"; !changed || got != want {
		t.Fatalf("intersection: changed=%v got=%q want=%q", changed, got, want)
	}
	// already ordered: no-op
	if _, changed := apply(t, OrderedTypes{}, "<?php function a(): A|B {}"); changed {
		t.Fatal("already ordered must not change")
	}
}

func TestMethodChainingIndentation(t *testing.T) {
	got, changed := apply(t, MethodChainingIndentation{}, "<?php\n$x = $obj->foo()\n->bar()\n        ->baz();\n")
	if want := "<?php\n$x = $obj->foo()\n    ->bar()\n    ->baz();\n"; !changed || got != want {
		t.Fatalf("changed=%v got=%q want=%q", changed, got, want)
	}
	// already aligned: no-op
	if _, changed := apply(t, MethodChainingIndentation{}, "<?php\n$x = $obj->foo()\n    ->bar();\n"); changed {
		t.Fatal("already aligned chain must not change")
	}
}
