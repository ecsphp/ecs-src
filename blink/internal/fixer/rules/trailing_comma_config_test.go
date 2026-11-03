package rules

import "testing"

func TestTrailingCommaConfigCases(t *testing.T) {
	t.Parallel()
	multi := "<?php foo(\n    $a,\n    $b\n);"
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"multiline default arrays only", TrailingCommaInMultiline{}, "<?php $a = [\n    1\n]; foo(\n    $a\n);", "<?php $a = [\n    1,\n]; foo(\n    $a\n);"},
		{"multiline arguments", TrailingCommaInMultiline{}.WithConfig(map[string]any{"elements": []any{"arguments"}}).(fixerRule), multi, "<?php foo(\n    $a,\n    $b,\n);"},
		{"multiline arguments skip arrays", TrailingCommaInMultiline{}.WithConfig(map[string]any{"elements": []string{"arguments"}}).(fixerRule), "<?php $a = [\n    1\n];", "<?php $a = [\n    1\n];"},
		{"multiline parameters", TrailingCommaInMultiline{}.WithConfig(map[string]any{"elements": []any{"parameters"}}).(fixerRule), "<?php function f(\n    $a,\n    $b\n) {}", "<?php function f(\n    $a,\n    $b,\n) {}"},
		{"multiline long array", TrailingCommaInMultiline{}.WithConfig(map[string]any{"elements": []any{"arrays"}}).(fixerRule), "<?php $a = array(\n    1\n);", "<?php $a = array(\n    1,\n);"},
		{"multiline heredoc skipped", TrailingCommaInMultiline{}.WithConfig(map[string]any{"after_heredoc": false}).(fixerRule), "<?php $a = [\n    <<<EOT\nx\nEOT\n];", "<?php $a = [\n    <<<EOT\nx\nEOT\n];"},
		{"multiline heredoc fixed", TrailingCommaInMultiline{}.WithConfig(map[string]any{"after_heredoc": true}).(fixerRule), "<?php $a = [\n    <<<EOT\nx\nEOT\n];", "<?php $a = [\n    <<<EOT\nx\nEOT,\n];"},
		{"single default", NoTrailingCommaInSingleline{}, "<?php foo($a, ); $b = [1,];", "<?php foo($a ); $b = [1];"},
		{"single arguments only", NoTrailingCommaInSingleline{}.WithConfig(map[string]any{"elements": []any{"arguments"}}).(fixerRule), "<?php foo($a, ); $b = [1,];", "<?php foo($a); $b = [1,];"},
		{"single array only", NoTrailingCommaInSingleline{}.WithConfig(map[string]any{"elements": []any{"array"}}).(fixerRule), "<?php foo($a,); $b = [1,];", "<?php foo($a,); $b = [1];"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}

// TestNoTrailingCommaInSinglelineKeepsAttributeArguments checks that the
// argument list of an attribute keeps its trailing comma, as PHP-CS-Fixer's
// "arguments" element skips attributes, while a call nested inside the
// attribute's arguments is still fixed.
func TestNoTrailingCommaInSinglelineKeepsAttributeArguments(t *testing.T) {
	t.Parallel()
	f := NoTrailingCommaInSingleline{}.WithConfig(map[string]any{
		"elements": []any{"arguments", "array", "array_destructuring", "group_import"},
	}).(fixerRule)
	cases := []struct{ in, want string }{
		{"<?php\n#[Attr(a: 1,)]\nclass C {}", "<?php\n#[Attr(a: 1,)]\nclass C {}"},
		{"<?php\n#[Foo, Ns\\Attr(a: B::C,)]\nclass C {}", "<?php\n#[Foo, Ns\\Attr(a: B::C,)]\nclass C {}"},
		{"<?php\n#[\\Attr(1,)]\nfunction f() {}", "<?php\n#[\\Attr(1,)]\nfunction f() {}"},
		{"<?php\n#[Attr(new B(1,),)]\nclass C {}", "<?php\n#[Attr(new B(1),)]\nclass C {}"},
		{"<?php\n$x = attr(1,);", "<?php\n$x = attr(1);"},
	}
	for _, c := range cases {
		if got, _ := apply(t, f, c.in); got != c.want {
			t.Errorf("got %q want %q", got, c.want)
		}
	}
}
