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
}
