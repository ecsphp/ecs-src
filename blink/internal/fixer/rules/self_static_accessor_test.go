package rules

import "testing"

func TestSelfStaticAccessor(t *testing.T) {
	// final class: static accessor and "new static" become self
	src := "<?php final class A {\n    public function m(): static {\n        $x = static::create();\n        $y = new static();\n        return static::$prop;\n    }\n}"
	want := "<?php final class A {\n    public function m(): static {\n        $x = self::create();\n        $y = new self();\n        return self::$prop;\n    }\n}"
	if got, changed := apply(t, SelfStaticAccessor{}, src); !changed || got != want {
		t.Fatalf("final class: changed=%v\n got=%q\nwant=%q", changed, got, want)
	}

	// non-final class: left untouched
	nf := "<?php class B {\n    public function m() {\n        return static::create();\n    }\n}"
	if _, changed := apply(t, SelfStaticAccessor{}, nf); changed {
		t.Fatal("non-final class must not change static::")
	}

	// enum is implicitly final
	en := "<?php enum E {\n    public function m() {\n        return static::cases();\n    }\n}"
	enWant := "<?php enum E {\n    public function m() {\n        return self::cases();\n    }\n}"
	if got, changed := apply(t, SelfStaticAccessor{}, en); !changed || got != enWant {
		t.Fatalf("enum: changed=%v got=%q", changed, got)
	}

	// return type "static" in a final class is not an accessor
	rt := "<?php final class C {\n    public function m(): static {\n        return $this;\n    }\n}"
	if _, changed := apply(t, SelfStaticAccessor{}, rt); changed {
		t.Fatal("return type static must not change")
	}

	// "static function" (closure modifier) is not an accessor
	cl := "<?php final class D {\n    public function m() {\n        return static function () {};\n    }\n}"
	if _, changed := apply(t, SelfStaticAccessor{}, cl); changed {
		t.Fatal("static closure must not change")
	}

	// already self: no-op
	ok := "<?php final class E2 {\n    public function m() {\n        return self::create();\n    }\n}"
	if _, changed := apply(t, SelfStaticAccessor{}, ok); changed {
		t.Fatal("already self must not change")
	}
}
