package rules

import "testing"

// TestNoPhp4ConstructorEngine holds cases checked against the PHP engine's output.
func TestNoPhp4ConstructorEngine(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"<?php\nclass Foo extends Bar\n{\n    public function Foo($a)\n    {\n        $this->Bar($a);\n    }\n}\n", "<?php\nclass Foo extends Bar\n{\n    public function __construct($a)\n    {\n        parent::__construct($a);\n    }\n}\n"},
		{"<?php\nnamespace\\foo();\nclass Foo\n{\n    public function Foo() {}\n}\n", "<?php\nnamespace\\foo();\nclass Foo\n{\n    public function __construct() {}\n}\n"},
		{"<?php\nclass Foo\n{\n    /** Doc. */\n    public function __construct($a) {}\n\n    /** Old. */\n    public function Foo($a)\n    {\n        $this->__construct($a);\n    }\n}\n", "<?php\nclass Foo\n{\n    /** Doc. */\n    public function __construct($a) {}\n}\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, NoPhp4Constructor{}, c.input); got != c.want {
			t.Errorf("input:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.want)
		}
	}
}
