package rules

import "testing"

// TestMultilineWhitespaceBeforeSemicolonsStrategies covers both strategies; the
// expected outputs come from PHP-CS-Fixer, including its loop bound: the
// statements its insertions push past the original token count are left
// alone (the last chain below, the last statement of the second case).
func TestMultilineWhitespaceBeforeSemicolonsStrategies(t *testing.T) {
	t.Parallel()
	chained := MultilineWhitespaceBeforeSemicolons{}.WithConfig(map[string]any{"strategy": "new_line_for_chained_calls"}).(fixerRule)
	cases := []struct {
		name     string
		rule     fixerRule
		in, want string
	}{
		{"new_line_for_chained_calls", chained, "<?php\n$this->a()\n    ->b()\n    ->c();\n$x = $y\n    ->foo(1, 2)\n    ->bar()   ;\n$z = Foo::make()\n    ->go(); // trailing\nfunction f() {\n    return $q\n        ->r()\n        ;\n}\n$k = foo()\n    ;\n$m = $n->o()->p();\nif ($a) {\n    $b->c(function () {\n        return 1;\n    })\n        ->d();\n}\n", "<?php\n$this->a()\n    ->b()\n    ->c()\n;\n$x = $y\n    ->foo(1, 2)\n    ->bar()\n;\n$z = Foo::make()\n    ->go() // trailing\n;\nfunction f() {\n    return $q\n        ->r()\n    ;\n}\n$k = foo();\n$m = $n->o()->p();\nif ($a) {\n    $b->c(function () {\n        return 1;\n    })\n        ->d();\n}\n"},
		{"new_line_for_chained_calls nested", chained, "<?php\nif ($a) {\n    $b->c(function () {\n        return 1;\n    })\n        ->d();\n}\n", "<?php\nif ($a) {\n    $b->c(function () {\n        return 1;\n    })\n        ->d()\n    ;\n}\n"},
		{"no_multi_line with comments", MultilineWhitespaceBeforeSemicolons{}, "<?php\n$a = foo() // c\n;\n$b = bar() // d\n;\n$c = 1\n;\n", "<?php\n$a = foo(); // c\n$b = bar(); // d\n$c = 1\n;\n"},
	}
	for _, c := range cases {
		if got, _ := apply(t, c.rule, c.in); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
