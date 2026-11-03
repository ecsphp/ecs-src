package rules

import "testing"

func TestEmptyLoopBodyConfig(t *testing.T) {
	t.Parallel()
	const in = "<?php\nfor ($i = 0; f($i); ++$i) {\n}\nwhile (g()) ;\ndo {\n    $a++;\n} while ($a < 3);\nforeach ($xs as $x) /* c */ ;\nwhile (h()) {}\n"
	cases := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{"default (empty config)", map[string]any{}, "<?php\nfor ($i = 0; f($i); ++$i) ;\n\nwhile (g()) ;\ndo {\n    $a++;\n} while ($a < 3);\nforeach ($xs as $x) /* c */ ;\nwhile (h()) ;\n"},
		{"semicolon", map[string]any{"style": "semicolon"}, "<?php\nfor ($i = 0; f($i); ++$i) ;\n\nwhile (g()) ;\ndo {\n    $a++;\n} while ($a < 3);\nforeach ($xs as $x) /* c */ ;\nwhile (h()) ;\n"},
		{"braces", map[string]any{"style": "braces"}, "<?php\nfor ($i = 0; f($i); ++$i) {\n}\nwhile (g()) {}\ndo {\n    $a++;\n} while ($a < 3);\nforeach ($xs as $x) /* c */ {}\nwhile (h()) {}\n"},
	}
	for _, c := range cases {
		f := EmptyLoopBody{}.WithConfig(c.config).(fixerRule)
		got, _ := apply(t, f, in)
		if got != c.want {
			t.Errorf("%s:\n got=%q\nwant=%q", c.name, got, c.want)
		}
		if again, changed := apply(t, f, got); changed || again != got {
			t.Errorf("%s: not idempotent: %q", c.name, again)
		}
	}
}
