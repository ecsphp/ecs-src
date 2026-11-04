package rules

import "testing"

// TestDeclareStrictTypesEngine holds cases checked against the PHP engine's output.
func TestDeclareStrictTypesEngine(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"<?php\n\n// c\nfoo();\n", "<?php declare(strict_types=1);\n\n// c\nfoo();\n"},
		{"#!/usr/bin/env php\n<?php\nfoo();\n", "#!/usr/bin/env php\n<?php declare(strict_types=1);\nfoo();\n"},
		{"<?php\nfoo();\n?>\n", "<?php declare(strict_types=1);\nfoo();\n?>\n"},
		{"<?php\nfoo();\n?>\n<b>x</b>\n", "<?php\nfoo();\n?>\n<b>x</b>\n"},
		{"<?php\tfoo();\n", "<?php declare(strict_types=1);\nfoo();\n"},
		{"<?php\n    foo();\n", "<?php declare(strict_types=1);\nfoo();\n"},
		{"<?php\r\nfoo();\r\n", "<?php declare(strict_types=1);\nfoo();\r\n"},
		{"<?php declare(ticks=1);\nfoo();\n", "<?php declare(strict_types=1);\ndeclare(ticks=1);\nfoo();\n"},
		{"<?php\ndeclare(STRICT_TYPES = 0x0);\n", "<?php\ndeclare(strict_types = 1);\n"},
		{"<?php", "<?php declare(strict_types=1);"},
	}
	for _, c := range cases {
		if got, _ := apply(t, DeclareStrictTypes{}, c.input); got != c.want {
			t.Errorf("input:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.want)
		}
	}
}
