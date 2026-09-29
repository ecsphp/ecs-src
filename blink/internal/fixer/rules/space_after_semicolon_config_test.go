package rules

import "testing"

func TestSpaceAfterSemicolonConfigCases(t *testing.T) {
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default keeps empty for", SpaceAfterSemicolon{}, "<?php for ($i = 0; ; ++$i) {}", "<?php for ($i = 0; ; ++$i) {}"},
		{"remove in empty for", SpaceAfterSemicolon{}.WithConfig(map[string]any{"remove_in_empty_for_expressions": true}).(fixerRule), "<?php for ($i = 0; ; ++$i) {}", "<?php for ($i = 0;; ++$i) {}"},
		{"remove keeps normal statements", SpaceAfterSemicolon{}.WithConfig(map[string]any{"remove_in_empty_for_expressions": true}).(fixerRule), "<?php $a=1;$b=2;", "<?php $a=1; $b=2;"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
