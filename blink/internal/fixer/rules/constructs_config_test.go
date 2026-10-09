package rules

import "testing"

func TestConstructsConfigCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"followed default", SingleSpaceAroundConstruct{}, "<?php if($a){return  1;}", "<?php if ($a){return  1;}"},
		{"followed list", SingleSpaceAroundConstruct{}.WithConfig(map[string]any{"constructs_followed_by_a_single_space": []any{"return"}}).(fixerRule), "<?php if($a){return  1;}", "<?php if($a){return 1;}"},
		{"preceded", SingleSpaceAroundConstruct{}.WithConfig(map[string]any{"constructs_preceded_by_a_single_space": []any{"else"}}).(fixerRule), "<?php if ($a) {}else {}", "<?php if ($a) {} else {}"},
		{"preceded joins block comment before elseif", SingleSpaceAroundConstruct{}.WithConfig(map[string]any{"constructs_preceded_by_a_single_space": []any{"elseif"}}).(fixerRule), "<?php if ($a) {\n}\n/* note */\nelseif ($b) {\n}", "<?php if ($a) {\n}\n/* note */ elseif ($b) {\n}"},
		{"preceded keeps line comment before else", SingleSpaceAroundConstruct{}.WithConfig(map[string]any{"constructs_preceded_by_a_single_space": []any{"else"}}).(fixerRule), "<?php if ($a) {\n}\n// note\nelse {\n}", "<?php if ($a) {\n}\n// note\nelse {\n}"},
		{"yield from", SingleSpaceAroundConstruct{}.WithConfig(map[string]any{"constructs_contain_a_single_space": []any{"yield_from"}}).(fixerRule), "<?php function f() { yield   from $a; }", "<?php function f() { yield from $a; }"},
		{"parens default none", NoSpacesInsideParenthesis{}, "<?php foo( $a );", "<?php foo($a);"},
		{"parens single", NoSpacesInsideParenthesis{}.WithConfig(map[string]any{"space": "single"}).(fixerRule), "<?php foo($a, bar());", "<?php foo( $a, bar() );"},
		{"parens single skips cast and trims empty", NoSpacesInsideParenthesis{}.WithConfig(map[string]any{"space": "single"}).(fixerRule), "<?php if ($a) { $b = (int) $c; foo( ); }", "<?php if ( $a ) { $b = (int) $c; foo(); }"},
		{"unary default", UnaryOperatorSpaces{}, "<?php $i ++; $x = ! $b;", "<?php $i++; $x = ! $b;"},
		{"unary only_dec_inc", UnaryOperatorSpaces{}.WithConfig(map[string]any{"only_dec_inc": true}).(fixerRule), "<?php $i ++; $x = ! $b;", "<?php $i++; $x = ! $b;"},
		{"unary all", UnaryOperatorSpaces{}.WithConfig(map[string]any{"only_dec_inc": false}).(fixerRule), "<?php $x = ! $b - 1;", "<?php $x = !$b - 1;"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
