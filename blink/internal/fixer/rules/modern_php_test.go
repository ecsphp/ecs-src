package rules

import "testing"

func TestModernSyntaxIsLeftIntact(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
	}{
		{"pipe operator", BinaryOperatorSpaces{}, "<?php\n$out = $items |> array_values(...) |> count(...);\n"},
		{"pipe operator on a new line", BinaryOperatorSpaces{}, "<?php\n$out = $items\n    |> array_unique(...)\n    |> array_values(...);\n"},
		{"asymmetric visibility", SingleSpaceAroundConstruct{}, "<?php\nclass Box {\n    public private(set) bool $open = false;\n    public function __construct(public protected(set) int $size, PUBLIC(SET) $id) {}\n}\n"},
		{"enum case named like a keyword", ControlStructureBraces{}, "<?php\nenum Branch: string {\n    case If = 'i';\n    case Else = 'e';\n    case While;\n}\n"},
		{"named argument named like a keyword", NoAlternativeSyntax{}, "<?php\nif ($a) {\n    pick(\n        if: true,\n        else: false,\n    );\n}\n"},
		{"attribute named like a function", NativeFunctionCasing{}, "<?php\n#[Required, Count(min: 2), Checked]\nfunction f() {}\n"},
		{"attribute on a closure", NativeFunctionCasing{}, "<?php\n$f = #[Count] fn() => 1;\n"},
		{"first-class callable", NativeFunctionCasing{}, "<?php\n$f = STRLEN(...);\n"},
		{"pipe operator, other operators configured", BinaryOperatorSpaces{}.WithConfig(map[string]any{"operators": map[string]any{"|": "no_space", ">": "no_space"}}).(fixerRule), "<?php\n$out = $items |> f(...);\n"},
	}
	for _, c := range cases {
		got, changed := apply(t, c.rule, c.src)
		if changed || got != c.src {
			t.Errorf("%s: changed=%v\n got: %q\nwant: %q", c.name, changed, got, c.src)
		}
	}
}

func TestModernSyntaxNeighboursAreFixed(t *testing.T) {
	t.Parallel()
	setOnly := SingleSpaceAroundConstruct{}.WithConfig(map[string]any{"constructs_followed_by_a_single_space": []any{"private_set"}}).(fixerRule)
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"asymmetric visibility followed by two spaces", SingleSpaceAroundConstruct{}, "<?php\nclass Box {\n    public private(set)  bool $open;\n}\n", "<?php\nclass Box {\n    public private(set) bool $open;\n}\n"},
		{"control structure after an enum", ControlStructureBraces{}, "<?php\nenum E { case Else; }\nif ($a) b();\n", "<?php\nenum E { case Else; }\nif ($a) { b(); }\n"},
		{"native call inside an attribute argument", NativeFunctionCasing{}, "<?php\n#[Limit(max: COUNT($a))]\nfunction f() {}\n", "<?php\n#[Limit(max: count($a))]\nfunction f() {}\n"},
		{"native call after an attribute", NativeFunctionCasing{}, "<?php\n#[Checked]\nfunction f() { return Count($a); }\n", "<?php\n#[Checked]\nfunction f() { return count($a); }\n"},
		{"asymmetric visibility configured by name", setOnly, "<?php\nclass Box {\n    public  private(set)  bool $open;\n}\n", "<?php\nclass Box {\n    public  private(set) bool $open;\n}\n"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s:\n got: %q\nwant: %q", c.name, got, c.want)
		}
	}
}
