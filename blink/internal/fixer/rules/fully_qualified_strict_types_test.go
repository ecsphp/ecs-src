package rules

import "testing"

func TestFullyQualifiedStrictTypes(t *testing.T) {
	f := FullyQualifiedStrictTypes{}

	// code positions: extends/new/instanceof/::/catch, aliased + unimported
	got, changed := apply(t, f, "<?php\nnamespace App;\nuse Foo\\Bar;\nuse Foo\\Baz as Q;\nclass X extends \\Foo\\Bar implements \\A\\B\n{\n    public function m()\n    {\n        new \\Foo\\Baz();\n        $x instanceof \\Foo\\Bar;\n        echo \\Foo\\Bar::class;\n    }\n}\n")
	want := "<?php\nnamespace App;\nuse Foo\\Bar;\nuse Foo\\Baz as Q;\nclass X extends Bar implements \\A\\B\n{\n    public function m()\n    {\n        new Q();\n        $x instanceof Bar;\n        echo Bar::class;\n    }\n}\n"
	if !changed || got != want {
		t.Fatalf("code: changed=%v got=%q", changed, got)
	}

	// parameter + return type hints
	got, changed = apply(t, f, "<?php\nnamespace App;\nuse Foo\\Bar;\nfunction f(\\Foo\\Bar $x): \\Foo\\Bar {}\n")
	if want := "<?php\nnamespace App;\nuse Foo\\Bar;\nfunction f(Bar $x): Bar {}\n"; !changed || got != want {
		t.Fatalf("hints: changed=%v got=%q", changed, got)
	}

	// docblock @param/@return/@var types
	got, changed = apply(t, f, "<?php\nnamespace App;\nuse Foo\\Bar;\n/**\n * @param \\Foo\\Bar $x\n * @return \\Foo\\Bar\n */\nfunction f($x) {}\n")
	if want := "<?php\nnamespace App;\nuse Foo\\Bar;\n/**\n * @param Bar $x\n * @return Bar\n */\nfunction f($x) {}\n"; !changed || got != want {
		t.Fatalf("docblock: changed=%v got=%q", changed, got)
	}

	// docblock must not touch variables or array-shape keys
	if _, changed := apply(t, f, "<?php\nnamespace App;\nuse Foo\\Bar as Middleware;\n/**\n * @return ($middleware is null ? int : array{file: string})\n */\nfunction f() {}\n"); changed {
		t.Fatal("variables and keys in docblocks must not be shortened")
	}

	// unimported name left fully qualified
	if _, changed := apply(t, f, "<?php\nnamespace App;\nuse Foo\\Bar;\nnew \\Other\\Thing();\n"); changed {
		t.Fatal("unimported name must stay fully qualified")
	}
}
