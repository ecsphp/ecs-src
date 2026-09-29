package rules

import "testing"

// SELF/PARENT/STATIC as constant names (member access or declaration) must not
// be lowercased by lowercase_static_reference.
func TestLowercaseStaticReferenceKeepsConstants(t *testing.T) {
	for _, src := range []string{
		"<?php return new Name(ObjectReference::SELF);",
		"<?php class A {\n    public const string PARENT = 'parent';\n}",
		"<?php if ($x === ObjectReference::STATIC) {}",
	} {
		if got, changed := apply(t, LowercaseStaticReference{}, src); changed || got != src {
			t.Fatalf("constant must be kept: in=%q got=%q", src, got)
		}
	}
	// the real keyword still lowercases
	got, changed := apply(t, LowercaseStaticReference{}, "<?php SELF::make();")
	if want := "<?php self::make();"; !changed || got != want {
		t.Fatalf("keyword self: changed=%v got=%q", changed, got)
	}
}

// a class constant named like a magic method must not be recased.
func TestMagicMethodCasingKeepsConstant(t *testing.T) {
	if _, changed := apply(t, MagicMethodCasing{}, "<?php class A {\n    public const string __SET = '__set';\n}"); changed {
		t.Fatal("constant __SET must not become __set")
	}
	// an actual method declaration is recased
	got, changed := apply(t, MagicMethodCasing{}, "<?php class A {\n    public function __GET($n) {}\n}")
	if want := "<?php class A {\n    public function __get($n) {}\n}"; !changed || got != want {
		t.Fatalf("method: changed=%v got=%q", changed, got)
	}
}
