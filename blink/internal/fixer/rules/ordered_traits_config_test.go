package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestOrderedTraitsConfigs(t *testing.T) {
	t.Parallel()
	fixers := []fixerRule{
		OrderedTraits{},
		OrderedTraits{}.WithConfig(map[string]any{"case_sensitive": false}).(fixerRule),
		OrderedTraits{}.WithConfig(map[string]any{"case_sensitive": true}).(fixerRule),
	}
	cases := []struct {
		src  string
		want [3]string
	}{
		{"<?php\nclass C {\n    use b;\n    use A;\n    use C;\n    use _d;\n    use a;\n}\n", [3]string{"<?php\nclass C {\n    use _d;\n    use A;\n    use a;\n    use b;\n    use C;\n}\n", "<?php\nclass C {\n    use _d;\n    use A;\n    use a;\n    use b;\n    use C;\n}\n", "<?php\nclass C {\n    use A;\n    use C;\n    use _d;\n    use a;\n    use b;\n}\n"}},
		{"<?php\nclass C {\n    use Z, b, A, _c;\n}\n", [3]string{"<?php\nclass C {\n    use _c, A, b, Z;\n}\n", "<?php\nclass C {\n    use _c, A, b, Z;\n}\n", "<?php\nclass C {\n    use A, Z, _c, b;\n}\n"}},
		{"<?php\nclass C {\n    use A, Z;\n    use AB;\n}\n", [3]string{"<?php\nclass C {\n    use AB;\n    use A, Z;\n}\n", "<?php\nclass C {\n    use AB;\n    use A, Z;\n}\n", "<?php\nclass C {\n    use AB;\n    use A, Z;\n}\n"}},
		{"<?php\nclass C {\n    use \\Foo\\Bar;\n    use Foo_Baz;\n    use \\Abc;\n    use foo\\a;\n}\n", [3]string{"<?php\nclass C {\n    use \\Abc;\n    use foo\\a;\n    use \\Foo\\Bar;\n    use Foo_Baz;\n}\n", "<?php\nclass C {\n    use \\Abc;\n    use foo\\a;\n    use \\Foo\\Bar;\n    use Foo_Baz;\n}\n", "<?php\nclass C {\n    use \\Abc;\n    use \\Foo\\Bar;\n    use Foo_Baz;\n    use foo\\a;\n}\n"}},
		{"<?php\nclass C {\n    use Ébc;\n    use abc;\n    use Zed;\n    use éa;\n}\n", [3]string{"<?php\nclass C {\n    use abc;\n    use Zed;\n    use Ébc;\n    use éa;\n}\n", "<?php\nclass C {\n    use abc;\n    use Zed;\n    use Ébc;\n    use éa;\n}\n", "<?php\nclass C {\n    use Zed;\n    use abc;\n    use Ébc;\n    use éa;\n}\n"}},
		{"<?php\ntrait T {\n    use B { foo as bar; }\n    use a;\n    // comment\n    use C;\n}\n", [3]string{"<?php\ntrait T {\n    use a;\n    use B { foo as bar; }\n    // comment\n    use C;\n}\n", "<?php\ntrait T {\n    use a;\n    use B { foo as bar; }\n    // comment\n    use C;\n}\n", "<?php\ntrait T {\n    use B { foo as bar; }\n    // comment\n    use C;\n    use a;\n}\n"}},
		{"<?php\nclass C {\n    use Beta;\n    use alpha;\n\n    public $x;\n\n    use Delta;\n    use gamma;\n}\n", [3]string{"<?php\nclass C {\n    use alpha;\n    use Beta;\n\n    public $x;\n\n    use Delta;\n    use gamma;\n}\n", "<?php\nclass C {\n    use alpha;\n    use Beta;\n\n    public $x;\n\n    use Delta;\n    use gamma;\n}\n", "<?php\nclass C {\n    use Beta;\n    use alpha;\n\n    public $x;\n\n    use Delta;\n    use gamma;\n}\n"}},
		{"<?php\nclass C {\n    /** @use B<int> */\n    use B;\n    /** @use A<int> */\n    use A; // a\n}\n", [3]string{"<?php\nclass C {\n    /** @use A<int> */\n    use A;\n    /** @use B<int> */\n    use B; // a\n}\n", "<?php\nclass C {\n    /** @use A<int> */\n    use A;\n    /** @use B<int> */\n    use B; // a\n}\n", "<?php\nclass C {\n    /** @use A<int> */\n    use A;\n    /** @use B<int> */\n    use B; // a\n}\n"}},
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
