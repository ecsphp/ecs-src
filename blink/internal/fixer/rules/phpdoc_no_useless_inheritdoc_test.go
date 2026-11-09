package rules

import "testing"

func TestPhpdocNoUselessInheritdoc(t *testing.T) {
	t.Parallel()
	f := PhpdocNoUselessInheritdoc{}

	// a sole {@inheritDoc}: the whole block and its indentation are removed
	got, changed := apply(t, f, "<?php\nclass A {\n    /**\n     * {@inheritDoc}\n     */\n    public function a() {}\n}")
	if want := "<?php\nclass A {\n    public function a() {}\n}"; !changed || got != want {
		t.Fatalf("sole: changed=%v got=%q", changed, got)
	}

	// single-line form
	got, changed = apply(t, f, "<?php\nclass A {\n    /** {@inheritdoc} */\n    public function b() {}\n}")
	if want := "<?php\nclass A {\n    public function b() {}\n}"; !changed || got != want {
		t.Fatalf("single: changed=%v got=%q", changed, got)
	}

	// with another tag: only the inheritdoc line is dropped, the block stays
	got, changed = apply(t, f, "<?php\nclass C {\n    /**\n     * {@inheritDoc}\n     * @param int $x\n     */\n    public function c($x) {}\n}")
	if want := "<?php\nclass C {\n    /**\n     * @param int $x\n     */\n    public function c($x) {}\n}"; !changed || got != want {
		t.Fatalf("withtag: changed=%v got=%q", changed, got)
	}

	// the docblock right before a class with nothing to inherit from
	got, changed = apply(t, f, "<?php\n/** {@inheritdoc} */\nclass E {}")
	if want := "<?php\nclass E {}"; !changed || got != want {
		t.Fatalf("classdoc: changed=%v got=%q", changed, got)
	}

	// an interface with no parent is fixed like a class
	got, changed = apply(t, f, "<?php\ninterface I {\n    /** {@inheritdoc} */\n    public function i();\n}")
	if want := "<?php\ninterface I {\n    public function i();\n}"; !changed || got != want {
		t.Fatalf("interface: changed=%v got=%q", changed, got)
	}

	// a trait-using class: its members may inherit, its own docblock may not
	got, changed = apply(t, f, "<?php\n/** {@inheritdoc} */\nclass T {\n    use Foo;\n\n    /** {@inheritdoc} */\n    public function t() {}\n}")
	if want := "<?php\nclass T {\n    use Foo;\n\n    /** {@inheritdoc} */\n    public function t() {}\n}"; !changed || got != want {
		t.Fatalf("trait: changed=%v got=%q", changed, got)
	}

	// a nested anonymous class is judged on its own parents
	got, changed = apply(t, f, "<?php\nclass N extends P {\n    /** {@inheritdoc} */\n    public function n() {\n        return new class {\n            /** {@inheritdoc} */\n            public function m() {}\n        };\n    }\n}")
	if want := "<?php\nclass N extends P {\n    /** {@inheritdoc} */\n    public function n() {\n        return new class {\n            public function m() {}\n        };\n    }\n}"; !changed || got != want {
		t.Fatalf("nested: changed=%v got=%q", changed, got)
	}

	// untouched: a class that extends or implements, a docblock outside any
	// class, a trait, and Foo::class (not a class declaration)
	for name, src := range map[string]string{
		"extends":    "<?php\nclass A extends B {\n    /** {@inheritdoc} */\n    protected $signature = 'x';\n}",
		"implements": "<?php\nclass A implements I {\n    /**\n     * {@inheritdoc}\n     */\n    public function a() {}\n}",
		"function":   "<?php\n/**\n * {@inheritDoc}\n * @param int $x\n */\nfunction c($x) {}",
		"traitdecl":  "<?php\ntrait T {\n    /** {@inheritdoc} */\n    public function t() {}\n}",
		"classconst": "<?php\n$x = Foo::class;\n/** {@inheritdoc} */\nfunction f() {}",
	} {
		if got, changed := apply(t, f, src); changed {
			t.Fatalf("%s must not change, got %q", name, got)
		}
	}

	// no inheritdoc: untouched
	if _, changed := apply(t, f, "<?php\n/**\n * @param int $x\n */\nfunction d($x) {}"); changed {
		t.Fatal("no inheritdoc must not change")
	}
}
