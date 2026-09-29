package rules

import "testing"

func TestGen3TypesConfigCases(t *testing.T) {
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"nullable default", NullableTypeDeclarationForDefaultNullValue{}, "<?php function f(int $a = null, int|string $b = null) {}", "<?php function f(?int $a = null, int|string $b = null) {}"},
		{"nullable true adds union null", NullableTypeDeclarationForDefaultNullValue{}.WithConfig(map[string]any{"use_nullable_type_declaration": true}).(fixerRule), "<?php function f(int|string $b = null) {}", "<?php function f(int|string|null $b = null) {}"},
		{"nullable false removes ?", NullableTypeDeclarationForDefaultNullValue{}.WithConfig(map[string]any{"use_nullable_type_declaration": false}).(fixerRule), "<?php function f(?int $a = null) {}", "<?php function f(int $a = null) {}"},
		{"nullable false removes |null", NullableTypeDeclarationForDefaultNullValue{}.WithConfig(map[string]any{"use_nullable_type_declaration": false}).(fixerRule), "<?php function f(int|null $a = null) {}", "<?php function f(int $a = null) {}"},
		{"declaration spaces default", TypeDeclarationSpaces{}, "<?php function f(int$a) {} class A { public int$b; const int  X = 1; }", "<?php function f(int $a) {} class A { public int $b; const int  X = 1; }"},
		{"elements property only", TypeDeclarationSpaces{}.WithConfig(map[string]any{"elements": []any{"property"}}).(fixerRule), "<?php function f(int$a) {} class A { public int$b; }", "<?php function f(int$a) {} class A { public int $b; }"},
		{"elements function only", TypeDeclarationSpaces{}.WithConfig(map[string]any{"elements": []any{"function"}}).(fixerRule), "<?php function f(int$a) {} class A { public int$b; }", "<?php function f(int $a) {} class A { public int$b; }"},
		{"elements constant", TypeDeclarationSpaces{}.WithConfig(map[string]any{"elements": []any{"constant"}}).(fixerRule), "<?php class A { const int  X = 1; }", "<?php class A { const int X = 1; }"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
