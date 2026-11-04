package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestStatementIndentationAttributeList(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"class A {\n    #[\n        Foo,\n        Bar(max: 3)\n    ]\n    public string $a;\n}", "class A {\n    #[\n        Foo,\n        Bar(max: 3)\n    ]\n    public string $a;\n}"},
		{"class A {\n    #[\n        Foo,\n        Bar(max: 3),\n    ]\n    public string $a;\n}", "class A {\n    #[\n        Foo,\n        Bar(max: 3),\n    ]\n    public string $a;\n}"},
		{"class A {\n    #[\n    Foo,\n    Bar\n    ]\n    public string $a;\n}", "class A {\n    #[\n        Foo,\n        Bar\n    ]\n    public string $a;\n}"},
		{"class A {\n    #[\n            Foo,\n            Bar\n    ]\n    public function f() {}\n}", "class A {\n    #[\n        Foo,\n        Bar\n    ]\n    public function f() {}\n}"},
		{"class A {\n    public function __construct(\n        #[Baz]\n        #[\n            Foo,\n            Bar(mode: Bar::MODE),\n        ]\n        public string $a\n    ) {\n    }\n}", "class A {\n    public function __construct(\n        #[Baz]\n        #[\n            Foo,\n            Bar(mode: Bar::MODE),\n        ]\n        public string $a\n    ) {\n    }\n}"},
		{"class A {\n    #[\n        Foo(\n            x: 1,\n        ),\n        Bar\n    ]\n    public string $a;\n}", "class A {\n    #[\n        Foo(\n            x: 1,\n        ),\n        Bar\n    ]\n    public string $a;\n}"},
		{"#[\n    Foo,\n    Bar\n]\nfinal class A {\n}", "#[\n    Foo,\n    Bar\n]\nfinal class A {\n}"},
		{"#[\n    Foo,\n    Bar\n]\nfunction f() {\n}", "#[\n    Foo,\n    Bar\n]\nfunction f() {\n}"},
		{"class A {\n    #[Foo, Bar]\n    public string $a;\n}", "class A {\n    #[Foo, Bar]\n    public string $a;\n}"},
		{"class A {\n    #[Foo(\n        x: 1\n    )]\n    public string $a;\n}", "class A {\n    #[Foo(\n        x: 1\n    )]\n    public string $a;\n}"},
		{"class A {\n    #[\n        Foo\n]\n    public function f() {\n        return 1;\n    }\n}", "class A {\n    #[\n        Foo\n    ]\n    public function f() {\n        return 1;\n    }\n}"},
		{"function f(#[\n    Foo\n] $a) {\n}", "function f(#[\n    Foo\n] $a) {\n}"},
		{"class A {\n    public function __construct(\n        #[\n            Bar\n        ] private int $b\n    ) {\n    }\n}", "class A {\n    public function __construct(\n        #[\n            Bar\n        ] private int $b\n    ) {\n    }\n}"},
	}
	for _, c := range cases {
		got, changed := apply(t, StatementIndentation{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
