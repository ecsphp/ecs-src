package rules

import "testing"

func TestBinaryOperatorSpacesConfigCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default single", BinaryOperatorSpaces{}, "<?php $a=[1=>2];", "<?php $a = [1 => 2];"},
		{"no space", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "no_space"}).(fixerRule), "<?php $a = [1 => 2];", "<?php $a=[1=>2];"},
		{"operator override", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "single_space", "operators": map[string]any{"=>": "no_space"}}).(fixerRule), "<?php $a=[1 => 2];", "<?php $a = [1=>2];"},
		{"align unsupported is a no-op", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "align"}).(fixerRule), "<?php $a=1;", "<?php $a=1;"},
		{"nil default leaves rest alone", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": nil, "operators": map[string]any{"=": "single_space"}}).(fixerRule), "<?php $a=[1=>2];", "<?php $a = [1=>2];"},
		{"extra operator via map", BinaryOperatorSpaces{}.WithConfig(map[string]any{"operators": map[string]any{"+=": "single_space"}}).(fixerRule), "<?php $a+=1;", "<?php $a += 1;"},
		{"collapse aligned arrow inside attribute", BinaryOperatorSpaces{}, "<?php #[X([\n    'a'   => 1,\n    'bb'  => 2,\n])]\nclass C {}", "<?php #[X([\n    'a' => 1,\n    'bb' => 2,\n])]\nclass C {}"},
		{"arrow inside attribute string left alone", BinaryOperatorSpaces{}, "<?php #[X('a   => b')]\nclass C {}", "<?php #[X('a   => b')]\nclass C {}"},
		{"reference assignment is spaced around =", BinaryOperatorSpaces{}, "<?php $a   =&$b;", "<?php $a = &$b;"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
