package rules

import "testing"

func TestTypesSpacesConfigCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default none", TypesSpaces{}, "<?php function f(int | string $a) {}", "<?php function f(int|string $a) {}"},
		{"single", TypesSpaces{}.WithConfig(map[string]any{"space": "single"}).(fixerRule), "<?php function f(int|string $a) {}", "<?php function f(int | string $a) {}"},
		{"catch single", TypesSpaces{}.WithConfig(map[string]any{"space": "none", "space_multiple_catch": "single"}).(fixerRule), "<?php try {} catch (A|B $e) {}", "<?php try {} catch (A | B $e) {}"},
		{"catch follows space", TypesSpaces{}.WithConfig(map[string]any{"space": "none"}).(fixerRule), "<?php try {} catch (A | B $e) {}", "<?php try {} catch (A|B $e) {}"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
