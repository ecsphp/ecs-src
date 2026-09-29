package rules

import "testing"

func TestPhpdocNoUselessInheritdoc(t *testing.T) {
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
	got, changed = apply(t, f, "<?php\n/**\n * {@inheritDoc}\n * @param int $x\n */\nfunction c($x) {}")
	if want := "<?php\n/**\n * @param int $x\n */\nfunction c($x) {}"; !changed || got != want {
		t.Fatalf("withtag: changed=%v got=%q", changed, got)
	}

	// no inheritdoc: untouched
	if _, changed := apply(t, f, "<?php\n/**\n * @param int $x\n */\nfunction d($x) {}"); changed {
		t.Fatal("no inheritdoc must not change")
	}
}
