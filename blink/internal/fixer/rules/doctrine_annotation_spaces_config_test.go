package rules

import "testing"

// Expected outputs are PHP-CS-Fixer's DoctrineAnnotationSpacesFixer results on
// the same input.
func TestDoctrineAnnotationSpacesConfig(t *testing.T) {
	t.Parallel()
	const in = "<?php\n/** @Spaced ( \"a\" ) */ class C1 {}\n/** @Args(a = 1, b=\"x\") */ class C2 {}\n/** @Foo(a=\"x\" , b=1) */ class C3 {}\n/** @Route(\"/p\",name=\"r\",  x=1) */ class C4 {}\n/** @Nested(x = {\"p\" = 1, \"q\"=2}) */ class C5 {}\n/** @Str(v = \"has , comma = and ( paren\") */ class C6 {}\n/** @ORM\\Column(type=\"string\", nullable=true) */ class C7 {}\n/** @Empty() */ class C8 {}\n/**\n * @param string $x\n * @see method( a )\n */\nclass C9 {}\n/**\n * @Multi(\n *     a = 1,\n *     b = {\"k\" : 2, \"l\":3, \"m\" = 4}\n * )\n */\nclass C10 {}\n/** @A(a,b,c) */ class C11 {}\n/** @lower(a , b) */ class C12 {}\n/** @A(x={\"k\":1}) @B( 2 ) */ class C13 {}\n"
	cases := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{"default (empty config)", map[string]any{}, "<?php\n/** @Spaced(\"a\") */ class C1 {}\n/** @Args(a=1, b=\"x\") */ class C2 {}\n/** @Foo(a=\"x\", b=1) */ class C3 {}\n/** @Route(\"/p\", name=\"r\",  x=1) */ class C4 {}\n/** @Nested(x={\"p\" = 1, \"q\" = 2}) */ class C5 {}\n/** @Str(v=\"has , comma = and ( paren\") */ class C6 {}\n/** @ORM\\Column(type=\"string\", nullable=true) */ class C7 {}\n/** @Empty() */ class C8 {}\n/**\n * @param string $x\n * @see method( a )\n */\nclass C9 {}\n/**\n * @Multi(\n *     a=1,\n *     b={\"k\" : 2, \"l\" : 3, \"m\" = 4}\n * )\n */\nclass C10 {}\n/** @A(a, b, c) */ class C11 {}\n/** @lower(a, b) */ class C12 {}\n/** @A(x={\"k\" : 1}) @B(2) */ class C13 {}\n"},
		{"before_array_assignments_colon false", map[string]any{"before_array_assignments_colon": false}, "<?php\n/** @Spaced(\"a\") */ class C1 {}\n/** @Args(a=1, b=\"x\") */ class C2 {}\n/** @Foo(a=\"x\", b=1) */ class C3 {}\n/** @Route(\"/p\", name=\"r\",  x=1) */ class C4 {}\n/** @Nested(x={\"p\" = 1, \"q\" = 2}) */ class C5 {}\n/** @Str(v=\"has , comma = and ( paren\") */ class C6 {}\n/** @ORM\\Column(type=\"string\", nullable=true) */ class C7 {}\n/** @Empty() */ class C8 {}\n/**\n * @param string $x\n * @see method( a )\n */\nclass C9 {}\n/**\n * @Multi(\n *     a=1,\n *     b={\"k\": 2, \"l\": 3, \"m\" = 4}\n * )\n */\nclass C10 {}\n/** @A(a, b, c) */ class C11 {}\n/** @lower(a, b) */ class C12 {}\n/** @A(x={\"k\": 1}) @B(2) */ class C13 {}\n"},
		{"null, true and false options", map[string]any{
			"before_argument_assignments":     nil,
			"after_argument_assignments":      true,
			"around_parentheses":              false,
			"around_commas":                   false,
			"before_array_assignments_equals": false,
			"after_array_assignments_colon":   nil,
		}, "<?php\n/** @Spaced ( \"a\" ) */ class C1 {}\n/** @Args(a = 1, b= \"x\") */ class C2 {}\n/** @Foo(a= \"x\" , b= 1) */ class C3 {}\n/** @Route(\"/p\",name= \"r\",  x= 1) */ class C4 {}\n/** @Nested(x = {\"p\"= 1, \"q\"= 2}) */ class C5 {}\n/** @Str(v = \"has , comma = and ( paren\") */ class C6 {}\n/** @ORM\\Column(type= \"string\", nullable= true) */ class C7 {}\n/** @Empty() */ class C8 {}\n/**\n * @param string $x\n * @see method( a )\n */\nclass C9 {}\n/**\n * @Multi(\n *     a = 1,\n *     b = {\"k\" : 2, \"l\" :3, \"m\"= 4}\n * )\n */\nclass C10 {}\n/** @A(a,b,c) */ class C11 {}\n/** @lower(a , b) */ class C12 {}\n/** @A(x= {\"k\" :1}) @B( 2 ) */ class C13 {}\n"},
	}
	for _, c := range cases {
		f := DoctrineAnnotationSpaces{}.WithConfig(c.config).(fixerRule)
		got, _ := apply(t, f, in)
		if got != c.want {
			t.Errorf("%s:\n got=%q\nwant=%q", c.name, got, c.want)
		}
		if again, changed := apply(t, f, got); changed || again != got {
			t.Errorf("%s: not idempotent: %q", c.name, again)
		}
	}
	if got, _ := apply(t, DoctrineAnnotationSpaces{}, in); got != cases[0].want {
		t.Errorf("zero value is not the default: %q", got)
	}
}

func TestDoctrineAnnotationSpacesIgnoredTags(t *testing.T) {
	t.Parallel()
	const in = "<?php\n/** @Foo( 1 ) @Bar( 2 ) */\nclass C {}\n"
	f := DoctrineAnnotationSpaces{}.WithConfig(map[string]any{"ignored_tags": []any{"Bar"}}).(fixerRule)
	got, _ := apply(t, f, in)
	if want := "<?php\n/** @Foo(1) @Bar( 2 ) */\nclass C {}\n"; got != want {
		t.Errorf("got=%q want=%q", got, want)
	}
}
