package rules

import "testing"

func TestFullyQualifiedStrictTypes(t *testing.T) {
	t.Parallel()
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

	// a URL after @see is not a type: case must not be rewritten to an import's
	if _, changed := apply(t, f, "<?php\nnamespace App;\n\nuse Vendor\\Package;\nuse WP_Query;\n\n/**\n * @see https://github.com/vendor/package/issues/1\n * @see https://developer.wordpress.org/reference/classes/wp_query/__construct/\n * @link https://developer.wordpress.org/reference/classes/wp_query/\n */\nfunction foo(WP_Query $query, Package $p): void\n{\n}\n"); changed {
		t.Fatal("URLs in @see/@link must not be rewritten")
	}
}

// Only what PHP-CS-Fixer's TypeExpression parses as a class name is shortened:
// array/object-shape keys, the parts of a dashed pseudo-type ("key-of",
// "positive-int"), the letters of a number constant and a tag spelled with
// another case are not types, even when an import has the same short name.
func TestFullyQualifiedStrictTypesDocNonTypes(t *testing.T) {
	t.Parallel()
	head := "<?php\nnamespace App;\nuse Vendor\\Limit;\nuse Vendor\\Key;\nuse Vendor\\Value;\nuse Vendor\\Positive;\nuse Vendor\\E3;\nuse Vendor\\Bar;\n/**\n"
	cases := []struct{ name, in, want string }{
		{"optional shape key", " * @param array{limit?: mixed, ...} $a\n", ""},
		{"optional shape key unspaced", " * @param array{limit?:int} $a\n", ""},
		{"object shape keys", " * @param object{limit:int,key?:int} $a\n", ""},
		{"key-of", " * @param key-of<\\Vendor\\Bar> $a\n", " * @param key-of<Bar> $a\n"},
		{"value-of", " * @param value-of<\\Vendor\\Value> $a\n", " * @param value-of<Value> $a\n"},
		{"dashed pseudo-type", " * @param positive-int|non-empty-array<int> $a\n", ""},
		{"number constant", " * @param 1e3|0x1E3|\\Vendor\\Bar $a\n", " * @param 1e3|0x1E3|Bar $a\n"},
		{"tag case", " * @Param \\Vendor\\Bar $a\n", ""},
		{"shape value still shortened", " * @param list{\\Vendor\\Bar} $a\n", " * @param list{Bar} $a\n"},
	}
	for _, c := range cases {
		in := head + c.in + " */\nfunction f($a) {}\n"
		want := in
		if c.want != "" {
			want = head + c.want + " */\nfunction f($a) {}\n"
		}
		if got, _ := apply(t, FullyQualifiedStrictTypes{}, in); got != want {
			t.Errorf("%s\n got: %q\nwant: %q", c.name, got, want)
		}
	}
}
