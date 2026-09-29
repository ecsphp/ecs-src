package rules

import "testing"

func TestPhpdocAlign(t *testing.T) {
	f := PhpdocAlign{}

	// left align: param uses spacing 2, return uses spacing 1
	got, changed := apply(t, f, "<?php\n\nclass T\n{\n    /**\n     * @param mixed $value\n     * @return   array\n     */\n    public function a($value) {}\n}\n")
	want := "<?php\n\nclass T\n{\n    /**\n     * @param  mixed  $value\n     * @return array\n     */\n    public function a($value) {}\n}\n"
	if !changed || got != want {
		t.Fatalf("param/return: changed=%v got=%q", changed, got)
	}

	// method: static keyword kept, hint/signature/description collapsed to spacing 1
	got, changed = apply(t, f, "<?php\n\nclass T\n{\n    /**\n     * @method   static \\Foo   bar(int $x)   Do the thing\n     * @method $this baz()\n     */\n    public function a() {}\n}\n")
	want = "<?php\n\nclass T\n{\n    /**\n     * @method static \\Foo bar(int $x) Do the thing\n     * @method $this baz()\n     */\n    public function a() {}\n}\n"
	if !changed || got != want {
		t.Fatalf("method: changed=%v got=%q", changed, got)
	}

	// an unbalanced multiline type is not a tag match, so the block is untouched
	src := "<?php\n\nclass T\n{\n    /**\n     * @param  mixed  $value\n     * @return ($value is array\n     *     ? true\n     *     : ($value is \\Traversable ? true : false)\n     * )\n     */\n    public function a($value) {}\n}\n"
	if _, changed := apply(t, f, src); changed {
		t.Fatal("unbalanced multiline @return type must be left untouched")
	}

	// a union member `$this` and a callable's inner params stay in the hint
	got, changed = apply(t, f, "<?php\n\nclass T\n{\n    /**\n     * @property-read HigherOrderBuilderProxy|$this $orWhere\n     * @param callable(mixed $a, mixed $b): mixed[] $merge Combine\n     */\n    public function a() {}\n}\n")
	want = "<?php\n\nclass T\n{\n    /**\n     * @property-read HigherOrderBuilderProxy|$this $orWhere\n     * @param  callable(mixed $a, mixed $b): mixed[]  $merge  Combine\n     */\n    public function a() {}\n}\n"
	if !changed || got != want {
		t.Fatalf("union/callable hint: changed=%v got=%q", changed, got)
	}
}
