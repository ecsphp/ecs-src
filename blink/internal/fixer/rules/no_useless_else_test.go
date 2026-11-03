package rules

import "testing"

// TestNoUselessElseEngine holds cases checked against the PHP engine's output.
func TestNoUselessElseEngine(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"<?php\nif ($a) {\n    exit;\n} else {\n    foo();\n}\n", "<?php\nif ($a) {\n    exit;\n}  \n    foo();\n\n"},
		{"<?php\nif ($a) {\n    die(1);\n} else {\n    foo();\n}\n", "<?php\nif ($a) {\n    die(1);\n}  \n    foo();\n\n"},
		{"<?php\nif ($a) {\n    $b->exit();\n} else {\n    foo();\n}\n", "<?php\nif ($a) {\n    $b->exit();\n} else {\n    foo();\n}\n"},
		{"<?php\nif ($a) {\n    return;\n} else {\n    function foo() {}\n}\n", "<?php\nif ($a) {\n    return;\n} else {\n    function foo() {}\n}\n"},
		{"<?php\nif ($a) {\n    return;\n} else {\n    $f = function () {};\n}\n", "<?php\nif ($a) {\n    return;\n}  \n    $f = function () {};\n\n"},
		{"<?php\nif ($a) {\n    return;\n} else { /* nothing */ }\n", "<?php\nif ($a) {\n    return;\n}   /* nothing */ \n"},
		// named-argument labels spelling a terminator or "function" are T_STRING
		{"<?php\nif ($a) {\n    foo(return: 1);\n} else {\n    bar();\n}\n", "<?php\nif ($a) {\n    foo(return: 1);\n} else {\n    bar();\n}\n"},
		{"<?php\nif ($a) {\n    foo(exit: 1);\n} else {\n    bar();\n}\n", "<?php\nif ($a) {\n    foo(exit: 1);\n} else {\n    bar();\n}\n"},
		{"<?php\nif ($a) {\n    foo(die: 1);\n} else {\n    bar();\n}\n", "<?php\nif ($a) {\n    foo(die: 1);\n} else {\n    bar();\n}\n"},
		{"<?php\nwhile ($x) {\n    if ($a) {\n        foo(break: 1);\n    } else {\n        bar();\n    }\n}\n", "<?php\nwhile ($x) {\n    if ($a) {\n        foo(break: 1);\n    } else {\n        bar();\n    }\n}\n"},
		{"<?php\nwhile ($x) {\n    if ($a) {\n        foo(continue: 1);\n    } else {\n        bar();\n    }\n}\n", "<?php\nwhile ($x) {\n    if ($a) {\n        foo(continue: 1);\n    } else {\n        bar();\n    }\n}\n"},
		{"<?php\nif ($a) {\n    foo(1, throw: 1);\n} else {\n    bar();\n}\n", "<?php\nif ($a) {\n    foo(1, throw: 1);\n} else {\n    bar();\n}\n"},
		{"<?php\nif ($a) {\n    foo(goto: 1);\n} else {\n    bar();\n}\n", "<?php\nif ($a) {\n    foo(goto: 1);\n} else {\n    bar();\n}\n"},
		{"<?php\nif ($a) {\n    return;\n} else {\n    foo(function: 1);\n}\n", "<?php\nif ($a) {\n    return;\n}  \n    foo(function: 1);\n\n"},
		{"<?php\nif ($a) {\n    return;\n} else {\n    foo(1, class: 1);\n}\n", "<?php\nif ($a) {\n    return;\n}  \n    foo(1, class: 1);\n\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, NoUselessElse{}, c.input); got != c.want {
			t.Errorf("input:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.want)
		}
	}
}
