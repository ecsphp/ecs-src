package rules

import "testing"

// TestNoSuperfluousElseifExit holds cases checked against the PHP engine's output.
func TestNoSuperfluousElseifExit(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"<?php\nif ($a) {\n    exit;\n} elseif ($b) {\n    foo();\n}\n", "<?php\nif ($a) {\n    exit;\n}\nif ($b) {\n    foo();\n}\n"},
		{"<?php\nif ($a) {\n    die;\n} else if ($b) {\n    foo();\n}\n", "<?php\nif ($a) {\n    die;\n}\nif ($b) {\n    foo();\n}\n"},
		// a named-argument label spelling a terminator is a T_STRING
		{"<?php\nif ($a) {\n    foo(exit: 1);\n} elseif ($b) {\n    bar();\n}\n", "<?php\nif ($a) {\n    foo(exit: 1);\n} elseif ($b) {\n    bar();\n}\n"},
		{"<?php\nif ($a) {\n    foo(return: 1);\n} elseif ($b) {\n    bar();\n}\n", "<?php\nif ($a) {\n    foo(return: 1);\n} elseif ($b) {\n    bar();\n}\n"},
		{"<?php\nif ($a) {\n    foo(die: 1);\n} elseif ($b) {\n    bar();\n}\n", "<?php\nif ($a) {\n    foo(die: 1);\n} elseif ($b) {\n    bar();\n}\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, NoSuperfluousElseif{}, c.input); got != c.want {
			t.Errorf("input:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.want)
		}
	}
}
