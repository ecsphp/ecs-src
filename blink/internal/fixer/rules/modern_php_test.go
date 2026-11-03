package rules

import "testing"

func TestModernSyntaxIsLeftIntact(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
	}{
		{"pipe operator", BinaryOperatorSpaces{}, "<?php\n$out = $items |> array_values(...) |> count(...);\n"},
		{"pipe operator on a new line", BinaryOperatorSpaces{}, "<?php\n$out = $items\n    |> array_unique(...)\n    |> array_values(...);\n"},
		{"pipe operator, other operators configured", BinaryOperatorSpaces{}.WithConfig(map[string]any{"operators": map[string]any{"|": "no_space", ">": "no_space"}}).(fixerRule), "<?php\n$out = $items |> f(...);\n"},
	}
	for _, c := range cases {
		got, changed := apply(t, c.rule, c.src)
		if changed || got != c.src {
			t.Errorf("%s: changed=%v\n got: %q\nwant: %q", c.name, changed, got, c.src)
		}
	}
}
