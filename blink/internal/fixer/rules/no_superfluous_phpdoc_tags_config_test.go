package rules

import "testing"

func TestNoSuperfluousPhpdocTagsConfig(t *testing.T) {
	// default keeps "@param mixed" on an untyped parameter (allow_mixed behavior)
	mixed := "<?php\nclass A {\n    /**\n     * @param mixed $x\n     */\n    public function f($x) {}\n}\n"
	if _, changed := apply(t, NoSuperfluousPhpdocTags{}, mixed); changed {
		t.Fatal("default must keep @param mixed on an untyped parameter")
	}

	// allow_mixed=false makes it superfluous
	noMixed := NoSuperfluousPhpdocTags{}.WithConfig(map[string]any{"allow_mixed": false})
	got, changed := apply(t, noMixed.(fixerRule), mixed)
	if want := "<?php\nclass A {\n    /**\n     */\n    public function f($x) {}\n}\n"; !changed || got != want {
		t.Fatalf("allow_mixed=false: changed=%v got=%q want=%q", changed, got, want)
	}

	// remove_inheritdoc drops a standalone @inheritDoc line
	inherit := "<?php\nclass A {\n    /**\n     * @inheritDoc\n     */\n    public function f() {}\n}\n"
	if _, changed := apply(t, NoSuperfluousPhpdocTags{}, inherit); changed {
		t.Fatal("default must keep @inheritDoc")
	}
	rm := NoSuperfluousPhpdocTags{}.WithConfig(map[string]any{"remove_inheritdoc": true})
	got, changed = apply(t, rm.(fixerRule), inherit)
	if want := "<?php\nclass A {\n    /**\n     */\n    public function f() {}\n}\n"; !changed || got != want {
		t.Fatalf("remove_inheritdoc: changed=%v got=%q want=%q", changed, got, want)
	}

	// a single-line docblock is processed too: remove_inheritdoc drops the tag,
	// leaving "/** */" for NoEmptyPhpdoc to clear (php-cs-fixer behavior)
	single := "<?php\nclass A {\n    /** @inheritDoc */\n    public function f() {}\n}\n"
	got, changed = apply(t, rm.(fixerRule), single)
	if want := "<?php\nclass A {\n    /** */\n    public function f() {}\n}\n"; !changed || got != want {
		t.Fatalf("single-line remove_inheritdoc: changed=%v got=%q want=%q", changed, got, want)
	}
	// a single-line superfluous @param is removed with allow_mixed=false
	singleParam := "<?php\nclass A {\n    /** @param int $x */\n    public function f(int $x) {}\n}\n"
	got, changed = apply(t, noMixed.(fixerRule), singleParam)
	if want := "<?php\nclass A {\n    /** */\n    public function f(int $x) {}\n}\n"; !changed || got != want {
		t.Fatalf("single-line superfluous @param: changed=%v got=%q want=%q", changed, got, want)
	}
}

func TestNoSuperfluousPhpdocTagsInheritDocWithDescription(t *testing.T) {
	rm := NoSuperfluousPhpdocTags{}.WithConfig(map[string]any{"remove_inheritdoc": true}).(fixerRule)

	// an inline {@inheritDoc} followed by a real description is kept: it is not
	// bounded by a tag or the comment end, so PHP leaves it unchanged too
	src := "<?php\nclass A {\n    /**\n     * {@inheritDoc}\n     *\n     * Asset-specific override for legacy public asset URLs.\n     *\n     * Backward compatibility rules apply.\n     */\n    public function f() {}\n}\n"
	if got, changed := apply(t, rm, src); changed || got != src {
		t.Fatalf("{@inheritDoc} with description must be kept: changed=%v got=%q", changed, got)
	}
}
