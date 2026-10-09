package rules

import "testing"

func TestCastSpacesConfigCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default single", CastSpaces{}, "<?php $a = (int)$b;", "<?php $a = (int) $b;"},
		{"none", CastSpaces{}.WithConfig(map[string]any{"space": "none"}).(fixerRule), "<?php $a = ( int )  $b;", "<?php $a = (int)$b;"},
		{"single explicit", CastSpaces{}.WithConfig(map[string]any{"space": "single"}).(fixerRule), "<?php $a = (int)$b;", "<?php $a = (int) $b;"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
