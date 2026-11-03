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

// PHP-CS-Fixer only visits classes and interfaces: an enum or a trait naming
// itself is left alone.
func TestSelfAccessorSkipsEnumsAndTraits(t *testing.T) {
	t.Parallel()
	src := "<?php\nenum Suit: string\n{\n    case Hearts = 'h';\n" +
		"    public static function of(Suit $s): Suit { return match ($s) { Suit::Hearts => Suit::Hearts }; }\n}\n" +
		"trait Greets\n{\n    public function f(Greets $g): Greets { return new Greets(); }\n}\n"
	if got, changed := apply(t, SelfAccessor{}, src); changed || got != src {
		t.Fatalf("enum/trait must not change: changed=%v got=%q", changed, got)
	}
	iface := "<?php\ninterface Shape\n{\n    const ID = Shape::BASE;\n    public function with(Shape $s): Shape;\n}\n"
	wantIface := "<?php\ninterface Shape\n{\n    const ID = self::BASE;\n    public function with(self $s): self;\n}\n"
	if got, changed := apply(t, SelfAccessor{}, iface); !changed || got != wantIface {
		t.Fatalf("interface: changed=%v got=%q", changed, got)
	}
}

// The name matches case-insensitively, and a name qualified with the current
// namespace is shortened to "self"; other qualified names, property types and
// a type after "&" are kept, as in PHP-CS-Fixer.
func TestSelfAccessorQualifiedAndCase(t *testing.T) {
	t.Parallel()
	src := "<?php\nnamespace Acme\\Shop;\nclass Cart\n{\n    public Cart $next;\n" +
		"    public function f(?Cart $a, Cart|int $b, Other&Cart $c): ?Cart\n    {\n" +
		"        $x = \\Acme\\Shop\\Cart::A;\n        $y = \\Other\\Cart::A;\n        $z = Shop\\Cart::A;\n" +
		"        $w = cart::A;\n        return new \\acme\\shop\\Cart();\n    }\n}\n"
	want := "<?php\nnamespace Acme\\Shop;\nclass Cart\n{\n    public Cart $next;\n" +
		"    public function f(?self $a, self|int $b, Other&Cart $c): ?self\n    {\n" +
		"        $x = self::A;\n        $y = \\Other\\Cart::A;\n        $z = Shop\\Cart::A;\n" +
		"        $w = self::A;\n        return new self();\n    }\n}\n"
	if got, changed := apply(t, SelfAccessor{}, src); !changed || got != want {
		t.Fatalf("changed=%v\n got=%q\nwant=%q", changed, got, want)
	}
	global := "<?php\nclass Cart\n{\n    public function f() { return \\Cart::A; }\n}\n"
	wantGlobal := "<?php\nclass Cart\n{\n    public function f() { return self::A; }\n}\n"
	if got, changed := apply(t, SelfAccessor{}, global); !changed || got != wantGlobal {
		t.Fatalf("global namespace: changed=%v got=%q", changed, got)
	}
}
