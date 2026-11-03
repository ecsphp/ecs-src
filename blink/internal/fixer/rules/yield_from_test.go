package rules

import "testing"

// "yield from" is one keyword token, as T_YIELD_FROM.
func TestYieldFromToken(t *testing.T) {
	t.Parallel()
	yieldFrom := SingleSpaceAroundConstruct{}.WithConfig(map[string]any{
		"constructs_contain_a_single_space":     []any{"yield_from"},
		"constructs_followed_by_a_single_space": []any{"yield_from"},
	}).(fixerRule)
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"single space inside and after", yieldFrom, "<?php\nfunction g() {\n    yield\n        from x();\n    YIELD  FROM  $a;\n    yield /* c */ from g();\n}\n", "<?php\nfunction g() {\n    yield from x();\n    YIELD FROM $a;\n    yield /* c */ from g();\n}\n"},
		{"lowercased as one keyword", LowercaseKeywords{}, "<?php\nfunction g() {\n    YIELD  FROM g();\n}\n", "<?php\nfunction g() {\n    yield  from g();\n}\n"},
		{"yielded array is not an offset", NoSpacesAroundOffset{}, "<?php\nfunction g() {\n    yield from [ 1, 2 ];\n}\n", "<?php\nfunction g() {\n    yield from [ 1, 2 ];\n}\n"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s:\n got: %q\nwant: %q", c.name, got, c.want)
		}
	}
}
