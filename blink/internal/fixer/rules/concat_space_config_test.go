package rules

import "testing"

func TestConcatSpaceConfigCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default one", ConcatSpace{}, "<?php $a = 'a'.'b';", "<?php $a = 'a' . 'b';"},
		{"one explicit", ConcatSpace{}.WithConfig(map[string]any{"spacing": "one"}).(fixerRule), "<?php $a = 'a'.'b';", "<?php $a = 'a' . 'b';"},
		{"none", ConcatSpace{}.WithConfig(map[string]any{"spacing": "none"}).(fixerRule), "<?php $a = 'a' . 'b';", "<?php $a = 'a'.'b';"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
