package rules

import "testing"

func TestNoUnneededControlParenthesesConfig(t *testing.T) {
	t.Parallel()
	src := "<?php function f() { echo ($a); return ($b); }"
	got, _ := apply(t, NoUnneededControlParentheses{}, src)
	if want := "<?php function f() { echo $a; return $b; }"; got != want {
		t.Fatalf("default got=%q want=%q", got, want)
	}
	f := NoUnneededControlParentheses{}.WithConfig(map[string]any{"statements": []any{"return"}})
	got, _ = apply(t, f.(fixerRule), src)
	if want := "<?php function f() { echo ($a); return $b; }"; got != want {
		t.Fatalf("configured got=%q want=%q", got, want)
	}
}

// PHP-CS-Fixer throws on this input, so there is no expected output to match.
func TestNoUnneededControlParenthesesCloneAfterBlockWithoutCase(t *testing.T) {
	t.Parallel()
	src := "<?php if ($a): { } clone ($b); endif;"
	got, changed := apply(t, NoUnneededControlParentheses{}, src)
	if changed || got != src {
		t.Fatalf("changed=%v got=%q", changed, got)
	}
}

// PHP-CS-Fixer removes these parentheses, which breaks most of them, so its output is no reference.
func TestNoUnneededControlParenthesesCloneInTernaryKeepsPairOfOtherOperands(t *testing.T) {
	t.Parallel()
	operands := []string{"$o->b", "$a::$b", "FOO", "K::X", "__CLASS__", "1", "enum", "new Foo", "new Foo()", "new $a", "(new Foo)"}
	for _, operand := range operands {
		for _, next := range []string{"->b", "?->b", "::b", "[0]", "()", " and $g", " + 1", " instanceof Foo", " ?: 1"} {
			src := "<?php\n$f ? clone (" + operand + ")" + next + " : 1;\n"
			if got, changed := apply(t, NoUnneededControlParentheses{}, src); changed || got != src {
				t.Errorf("%s changed=%v got=%q", src, changed, got)
			}
		}
	}
}
