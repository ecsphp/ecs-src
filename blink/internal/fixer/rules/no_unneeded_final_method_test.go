package rules

import "testing"

// TestNoUnneededFinalMethodEngine holds cases checked against the PHP engine's output.
func TestNoUnneededFinalMethodEngine(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"<?php\nenum E: string\n{\n    case A = 'a';\n\n    final public function f() {}\n}\n", "<?php\nenum E: string\n{\n    case A = 'a';\n\n    public function f() {}\n}\n"},
		{"<?php\nfoo(class: 1);\ntrait T\n{\n    final private function f() {}\n}\n", "<?php\nfoo(class: 1);\ntrait T\n{\n    final private function f() {}\n}\n"},
		{"<?php\nclass A\n{\n    final private function __construct() {}\n    final  private static function f() {}\n    final /* c */ private function g() {}\n}\n", "<?php\nclass A\n{\n    final private function __construct() {}\n    private static function f() {}\n    /* c */ private function g() {}\n}\n"},
		{"<?php\n$a = new class {\n    final private function f() {}\n    final public function g() {}\n};\n", "<?php\n$a = new class {\n    private function f() {}\n    final public function g() {}\n};\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, NoUnneededFinalMethod{}, c.input); got != c.want {
			t.Errorf("input:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.want)
		}
	}
}
