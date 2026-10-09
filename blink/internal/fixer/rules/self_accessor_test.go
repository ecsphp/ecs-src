package rules

import "testing"

func TestSelfAccessorSkipsNestedClass(t *testing.T) {
	t.Parallel()
	// A reference to the outer class name from inside a nested anonymous class
	// must be left untouched, while a reference from the outer class's own
	// top-level method is rewritten to "self".
	src := "<?php class Outer {\n" +
		"    const NAME = 1;\n" +
		"    public function top(): string\n" +
		"    {\n" +
		"        return Outer::NAME;\n" +
		"    }\n" +
		"    public function make()\n" +
		"    {\n" +
		"        return new class {\n" +
		"            public function get()\n" +
		"            {\n" +
		"                return Outer::NAME;\n" +
		"            }\n" +
		"        };\n" +
		"    }\n" +
		"}"
	want := "<?php class Outer {\n" +
		"    const NAME = 1;\n" +
		"    public function top(): string\n" +
		"    {\n" +
		"        return self::NAME;\n" +
		"    }\n" +
		"    public function make()\n" +
		"    {\n" +
		"        return new class {\n" +
		"            public function get()\n" +
		"            {\n" +
		"                return Outer::NAME;\n" +
		"            }\n" +
		"        };\n" +
		"    }\n" +
		"}"
	if got, changed := apply(t, SelfAccessor{}, src); !changed || got != want {
		t.Fatalf("nested anonymous class: changed=%v\n got=%q\nwant=%q", changed, got, want)
	}

	// A class whose only self-name reference lives inside a nested anonymous
	// class must be left completely unchanged.
	only := "<?php class Solo {\n" +
		"    const NAME = 1;\n" +
		"    public function make()\n" +
		"    {\n" +
		"        return new class {\n" +
		"            public function get()\n" +
		"            {\n" +
		"                return Solo::NAME;\n" +
		"            }\n" +
		"        };\n" +
		"    }\n" +
		"}"
	if got, changed := apply(t, SelfAccessor{}, only); changed || got != only {
		t.Fatalf("reference only inside anonymous class must not change: changed=%v got=%q", changed, got)
	}

	// A closure parameter type-hint referencing the enclosing class is left
	// untouched: PHP's fixer skips lambdas entirely.
	closure := "<?php class Email {\n" +
		"    public static function load(): void\n" +
		"    {\n" +
		"        $x = function (Email $email): void {\n" +
		"            return;\n" +
		"        };\n" +
		"    }\n" +
		"}"
	if got, changed := apply(t, SelfAccessor{}, closure); changed || got != closure {
		t.Fatalf("closure param type must not change: changed=%v got=%q", changed, got)
	}

	// Same for arrow functions.
	arrow := "<?php class Email {\n" +
		"    public function m(): void\n" +
		"    {\n" +
		"        $f = fn (Email $e): int => 1;\n" +
		"    }\n" +
		"}"
	if got, changed := apply(t, SelfAccessor{}, arrow); changed || got != arrow {
		t.Fatalf("arrow fn param type must not change: changed=%v got=%q", changed, got)
	}

	// A genuine same-class type-hint in a real method signature is still rewritten.
	method := "<?php class Email {\n" +
		"    public function with(Email $other): void\n" +
		"    {\n" +
		"    }\n" +
		"}"
	wantMethod := "<?php class Email {\n" +
		"    public function with(self $other): void\n" +
		"    {\n" +
		"    }\n" +
		"}"
	if got, changed := apply(t, SelfAccessor{}, method); !changed || got != wantMethod {
		t.Fatalf("method param type should rewrite: changed=%v got=%q", changed, got)
	}
}
