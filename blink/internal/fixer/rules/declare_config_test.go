package rules

import "testing"

func TestDeclareEqualNormalizeConfigCases(t *testing.T) {
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default none", DeclareEqualNormalize{}, "<?php declare(strict_types = 1);", "<?php declare(strict_types=1);"},
		{"single", DeclareEqualNormalize{}.WithConfig(map[string]any{"space": "single"}).(fixerRule), "<?php declare(strict_types=1);", "<?php declare(strict_types = 1);"},
		{"single normalizes wide space", DeclareEqualNormalize{}.WithConfig(map[string]any{"space": "single"}).(fixerRule), "<?php declare(ticks  =  1);", "<?php declare(ticks = 1);"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
