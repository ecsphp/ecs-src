package rules

import (
	"testing"

	"blink/internal/fixer"
)

func TestGen5PhpdocMiscReturnSelfReference(t *testing.T) {
	t.Parallel()
	f := PhpdocReturnSelfReference{}

	// this -> $this
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return this\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return $this\n     */\n    public function x(){}\n}", true)

	// @self -> self
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return @self\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return self\n     */\n    public function x(){}\n}", true)

	// $static -> static
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return $static\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return static\n     */\n    public function x(){}\n}", true)

	// case-insensitive alias match
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return This\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return $this\n     */\n    public function x(){}\n}", true)

	// description after the type is preserved
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return this the current instance\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return $this the current instance\n     */\n    public function x(){}\n}", true)

	// union member is replaced, others kept
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return this|null\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return $this|null\n     */\n    public function x(){}\n}", true)

	// single-line docblock
	assertFix(t, f,
		"<?php\nclass A{\n    /** @return @static */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /** @return static */\n    public function x(){}\n}", true)

	// no-op: already-correct target types are left alone
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return $this\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return $this\n     */\n    public function x(){}\n}", false)
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return self\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return self\n     */\n    public function x(){}\n}", false)

	// no-op: unrelated return type
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return int\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return int\n     */\n    public function x(){}\n}", false)

	// no-op: alias not the return type (@param), and bare "self"/"static" are not keys
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @param this $x\n     */\n    public function x($x){}\n}",
		"<?php\nclass A{\n    /**\n     * @param this $x\n     */\n    public function x($x){}\n}", false)
	assertFix(t, f,
		"<?php\nclass A{\n    /**\n     * @return static\n     */\n    public function x(){}\n}",
		"<?php\nclass A{\n    /**\n     * @return static\n     */\n    public function x(){}\n}", false)
}

func TestGen5PhpdocMiscReturnSelfReferenceSourceURL(t *testing.T) {
	t.Parallel()
	var f fixer.Fixer = PhpdocReturnSelfReference{}
	if got, want := f.SourceURL(), fixer.SourceURLFor(f.Name()); got != want {
		t.Fatalf("SourceURL %q, want %q", got, want)
	}
}
