package rules

import "testing"

// keyword-spelled class-constant names must not be lowercased (sweep finding)
func TestLowercaseKeywordsKeepsConstantName(t *testing.T) {
	src := "<?php class A {\n    public const string ARRAY = 'x';\n}"
	if got, changed := apply(t, LowercaseKeywords{}, src); changed || got != src {
		t.Fatalf("constant name ARRAY must not change: %q", got)
	}
}

// a method named like a keyword ("match") is a function decl, not a control body
func TestBracesPositionKeywordNamedMethod(t *testing.T) {
	got, changed := apply(t, BracesPosition{}, "<?php class A\n{\n    public function match($x): bool {\n    }\n}")
	want := "<?php class A\n{\n    public function match($x): bool\n    {\n    }\n}"
	if !changed || got != want {
		t.Fatalf("keyword-named method brace should go to next line\n got: %q\nwant: %q", got, want)
	}
}

// nullable return/param types must not be treated as ternaries (sweep finding)
func TestTernaryLeavesNullableTypes(t *testing.T) {
	for _, src := range []string{
		"<?php function f(): ?string\n{\n}",
		"<?php function f(?int $x): ?array\n{\n}",
	} {
		if got, changed := apply(t, TernaryOperatorSpaces{}, src); changed || got != src {
			t.Fatalf("nullable type must not change: in=%q got=%q", src, got)
		}
	}
}
