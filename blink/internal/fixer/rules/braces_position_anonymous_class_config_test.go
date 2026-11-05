package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestBracesPositionAnonymousClassConfigs(t *testing.T) {
	t.Parallel()
	fixers := []fixerRule{
		BracesPosition{},
		BracesPosition{}.WithConfig(map[string]any{"allow_single_line_empty_anonymous_classes": true}).(fixerRule),
		BracesPosition{}.WithConfig(map[string]any{"allow_single_line_empty_anonymous_classes": false}).(fixerRule),
		BracesPosition{}.WithConfig(map[string]any{"allow_single_line_empty_anonymous_classes": false, "anonymous_classes_opening_brace": "next_line_unless_newline_at_signature_end"}).(fixerRule),
	}
	cases := []struct {
		src  string
		want [4]string
	}{
		{"<?php\n$a = new class {};\n", [4]string{"<?php\n$a = new class {};\n", "<?php\n$a = new class {};\n", "<?php\n$a = new class {\n};\n", "<?php\n$a = new class\n{\n};\n"}},
		{"<?php\n$a = new class{};\n", [4]string{"<?php\n$a = new class {};\n", "<?php\n$a = new class {};\n", "<?php\n$a = new class {\n};\n", "<?php\n$a = new class\n{\n};\n"}},
		{"<?php\n$a = new class { };\n", [4]string{"<?php\n$a = new class { };\n", "<?php\n$a = new class { };\n", "<?php\n$a = new class {\n};\n", "<?php\n$a = new class\n{\n};\n"}},
		{"<?php\n$a = new class {\n};\n", [4]string{"<?php\n$a = new class {\n};\n", "<?php\n$a = new class {\n};\n", "<?php\n$a = new class {\n};\n", "<?php\n$a = new class\n{\n};\n"}},
		{"<?php\n$a = new class { /* c */ };\n", [4]string{"<?php\n$a = new class { /* c */ };\n", "<?php\n$a = new class { /* c */ };\n", "<?php\n$a = new class { /* c */\n};\n", "<?php\n$a = new class\n{ /* c */\n};\n"}},
		{"<?php\n$a = new class { public $x; };\n", [4]string{"<?php\n$a = new class {\npublic $x;\n};\n", "<?php\n$a = new class {\npublic $x;\n};\n", "<?php\n$a = new class {\npublic $x;\n};\n", "<?php\n$a = new class\n{\npublic $x;\n};\n"}},
		{"<?php\n$a = new class(1, 2) extends Foo implements Bar {};\n", [4]string{"<?php\n$a = new class(1, 2) extends Foo implements Bar {};\n", "<?php\n$a = new class(1, 2) extends Foo implements Bar {};\n", "<?php\n$a = new class(1, 2) extends Foo implements Bar {\n};\n", "<?php\n$a = new class(1, 2) extends Foo implements Bar\n{\n};\n"}},
		{"<?php\nfunction f() {\n    return new class {};\n}\n", [4]string{"<?php\nfunction f()\n{\n    return new class {};\n}\n", "<?php\nfunction f()\n{\n    return new class {};\n}\n", "<?php\nfunction f()\n{\n    return new class {\n    };\n}\n", "<?php\nfunction f()\n{\n    return new class\n    {\n    };\n}\n"}},
		{"<?php\n$a = new class {\n    public function f()\n    {\n        return new class { };\n    }\n};\n", [4]string{"<?php\n$a = new class {\n    public function f()\n    {\n        return new class { };\n    }\n};\n", "<?php\n$a = new class {\n    public function f()\n    {\n        return new class { };\n    }\n};\n", "<?php\n$a = new class {\n    public function f()\n    {\n        return new class {\n        };\n    }\n};\n", "<?php\n$a = new class\n{\n    public function f()\n    {\n        return new class\n        {\n        };\n    }\n};\n"}},
		{"<?php\n$f = function () { return new class { public $a; }; };\n$g = function () { return new class {\n}; };\n", [4]string{"<?php\n$f = function () { return new class {\npublic $a;\n}; };\n$g = function () {\nreturn new class {\n};\n};\n", "<?php\n$f = function () { return new class {\npublic $a;\n}; };\n$g = function () {\nreturn new class {\n};\n};\n", "<?php\n$f = function () { return new class { public $a; }; };\n$g = function () {\nreturn new class {\n};\n};\n", "<?php\n$f = function () { return new class\n{ public $a; }; };\n$g = function () {\nreturn new class\n{\n};\n};\n"}},
		{"<?php\n$f = function () {\n    return new class {};\n};\n", [4]string{"<?php\n$f = function () {\n    return new class {};\n};\n", "<?php\n$f = function () {\n    return new class {};\n};\n", "<?php\n$f = function () {\n    return new class {\n    };\n};\n", "<?php\n$f = function () {\n    return new class\n    {\n    };\n};\n"}},
		{"<?php\n$f = function () { return new class {}; };\n$g = new class {};\n", [4]string{"<?php\n$f = function () { return new class {}; };\n$g = new class {};\n", "<?php\n$f = function () { return new class {}; };\n$g = new class {};\n", "<?php\n$f = function () { return new class {}; };\n$g = new class {\n};\n", "<?php\n$f = function () { return new class\n{}; };\n$g = new class\n{\n};\n"}},
		{"<?php\n$a = new readonly class {};\n$b = new #[Attr] class {};\n", [4]string{"<?php\n$a = new readonly class {};\n$b = new #[Attr] class {};\n", "<?php\n$a = new readonly class {};\n$b = new #[Attr] class {};\n", "<?php\n$a = new readonly class {\n};\n$b = new #[Attr] class {\n};\n", "<?php\n$a = new readonly class\n{\n};\n$b = new #[Attr] class\n{\n};\n"}},
		{"<?php\nfoo(new class {}, new class {});\n", [4]string{"<?php\nfoo(new class {}, new class {});\n", "<?php\nfoo(new class {}, new class {});\n", "<?php\nfoo(new class {\n}, new class {\n});\n", "<?php\nfoo(new class\n{\n}, new class\n{\n});\n"}},
	}
	for _, c := range cases {
		for k, f := range fixers {
			got, changed := apply(t, f, c.src)
			if got != c.want[k] || changed != (c.want[k] != c.src) {
				t.Errorf("config %d src=%q\nchanged=%v got=%q\nwant=%q", k, c.src, changed, got, c.want[k])
			}
		}
	}
}
