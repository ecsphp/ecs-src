package rules

import "testing"

func TestArrayNotationConfigCases(t *testing.T) {
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"array default short", ArraySyntax{}, "<?php $a = array(1, 2);", "<?php $a = [1, 2];"},
		{"array long", ArraySyntax{}.WithConfig(map[string]any{"syntax": "long"}).(fixerRule), "<?php $a = [1, [2]]; $b = $a[0]; [$x, $y] = $a;", "<?php $a = array(1, array(2)); $b = $a[0]; [$x, $y] = $a;"},
		{"list default short", ListSyntax{}, "<?php list($a, $b) = $x;", "<?php [$a, $b] = $x;"},
		{"list long", ListSyntax{}.WithConfig(map[string]any{"syntax": "long"}).(fixerRule), "<?php [$a, [$b]] = $x; $c = [1]; foreach ($y as [$d]) {}", "<?php list($a, list($b)) = $x; $c = [1]; foreach ($y as list($d)) {}"},
		{"before comma default", NoWhitespaceBeforeCommaInArray{}, "<?php $a = [1 , 2];", "<?php $a = [1, 2];"},
		{"before comma heredoc kept", NoWhitespaceBeforeCommaInArray{}.WithConfig(map[string]any{"after_heredoc": false}).(fixerRule), "<?php $a = [<<<EOT\nx\nEOT , 2];", "<?php $a = [<<<EOT\nx\nEOT , 2];"},
		{"before comma heredoc fixed", NoWhitespaceBeforeCommaInArray{}.WithConfig(map[string]any{"after_heredoc": true}).(fixerRule), "<?php $a = [<<<EOT\nx\nEOT , 2];", "<?php $a = [<<<EOT\nx\nEOT, 2];"},
		{"after comma default", WhitespaceAfterCommaInArray{}, "<?php $a = [1,  2,3];", "<?php $a = [1,  2, 3];"},
		{"after comma single space", WhitespaceAfterCommaInArray{}.WithConfig(map[string]any{"ensure_single_space": true}).(fixerRule), "<?php $a = [1,  2,3];", "<?php $a = [1, 2, 3];"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
