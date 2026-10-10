package rules

import "testing"

func TestClassDefinitionSingleLine(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass Foo\nextends Bar\n{\n}\n"

	// default: a multiline header is left untouched
	if got, changed := apply(t, ClassDefinition{}, src); changed || got != src {
		t.Fatalf("default: changed=%v got=%q want unchanged", changed, got)
	}

	// single_line=true: the header collapses onto one line, brace kept below
	cfg := ClassDefinition{}.WithConfig(map[string]any{"single_line": true})
	got, changed := apply(t, cfg, src)
	if want := "<?php\nclass Foo extends Bar\n{\n}\n"; !changed || got != want {
		t.Fatalf("single_line=true: changed=%v got=%q want=%q", changed, got, want)
	}
}

// A class-name reference the lexer tags as a class-like keyword (here "Enum" in
// "Enum::MODE_ADD") is not a declaration, so single_line must not flatten the code
// that follows it.
func TestClassDefinitionIgnoresKeywordNamedReference(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass Foo\n{\n    public function run(): void\n    {\n        $x = [\n" +
		"            Enum::MODE_ADD,\n            Enum::MODE_EDIT,\n        ];\n    }\n\n" +
		"    public function other(): void\n    {\n    }\n}\n"

	cfg := ClassDefinition{}.WithConfig(map[string]any{"single_line": true})
	if got, changed := apply(t, cfg, src); changed || got != src {
		t.Fatalf("single_line must not touch an Enum:: reference: changed=%v got=%q", changed, got)
	}
}

func TestClassDefinitionSpaceBeforeParenthesis(t *testing.T) {
	t.Parallel()
	src := "<?php $x = new class() {};"

	// default: anonymous class left untouched
	if got, changed := apply(t, ClassDefinition{}, src); changed || got != src {
		t.Fatalf("default: changed=%v got=%q want unchanged", changed, got)
	}

	// space_before_parenthesis=true: one space before the constructor "("
	cfg := ClassDefinition{}.WithConfig(map[string]any{"space_before_parenthesis": true})
	got, changed := apply(t, cfg, src)
	if want := "<?php $x = new class () {};"; !changed || got != want {
		t.Fatalf("space_before_parenthesis=true: changed=%v got=%q want=%q", changed, got, want)
	}
}
