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
		{"compound concat assignment spaced by default", BinaryOperatorSpaces{}, "<?php $a .='c';", "<?php $a .= 'c';"},
		{"compound add assignment spaced by default", BinaryOperatorSpaces{}, "<?php $a +=1;", "<?php $a += 1;"},
		{"compound assignment honors no_space default", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "no_space"}).(fixerRule), "<?php $a .= 'c';", "<?php $a.='c';"},
		{"collapse aligned arrow inside attribute", BinaryOperatorSpaces{}, "<?php #[X([\n    'a'   => 1,\n    'bb'  => 2,\n])]\nclass C {}", "<?php #[X([\n    'a' => 1,\n    'bb' => 2,\n])]\nclass C {}"},
		{"arrow inside attribute string left alone", BinaryOperatorSpaces{}, "<?php #[X('a   => b')]\nclass C {}", "<?php #[X('a   => b')]\nclass C {}"},
		{"reference assignment is spaced around =", BinaryOperatorSpaces{}, "<?php $a   =&$b;", "<?php $a = &$b;"},
		{"compound assign follows default without override", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "single_space", "operators": map[string]any{"=>": "single_space", "=": "single_space"}}).(fixerRule), "<?php $url .='c'; $a .=$b;", "<?php $url .= 'c'; $a .= $b;"},
		{"compound assign explicit no_space override", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "single_space", "operators": map[string]any{".=": "no_space"}}).(fixerRule), "<?php $url .= 'c';", "<?php $url.='c';"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
