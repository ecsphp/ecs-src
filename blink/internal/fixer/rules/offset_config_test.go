package rules

import "testing"

func TestNoSpacesAroundOffsetConfig(t *testing.T) {
	src := "<?php $b [ 'a' ] [ 'b' ];"
	cases := []struct {
		name string
		rule fixerRule
		want string
	}{
		{"default inside", NoSpacesAroundOffset{}, "<?php $b ['a'] ['b'];"},
		{"inside", NoSpacesAroundOffset{}.WithConfig(map[string]any{"positions": []any{"inside"}}).(fixerRule), "<?php $b ['a'] ['b'];"},
		{"outside", NoSpacesAroundOffset{}.WithConfig(map[string]any{"positions": []any{"outside"}}).(fixerRule), "<?php $b[ 'a' ][ 'b' ];"},
		{"both", NoSpacesAroundOffset{}.WithConfig(map[string]any{"positions": []any{"inside", "outside"}}).(fixerRule), "<?php $b['a']['b'];"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
