package set

import (
	"testing"

	"blink/internal/fixer"
	"blink/internal/lexer"
	"blink/internal/tokens"
)

// member returns the fixer of fixers whose short class name is short.
func member(t *testing.T, fixers []fixer.Fixer, short string) fixer.Fixer {
	t.Helper()
	for _, f := range fixers {
		if shortName(f.Name()) == short {
			return f
		}
	}
	t.Fatalf("%s is not a member", short)
	return nil
}

func fix(f fixer.Fixer, src string) string {
	s := tokens.New(lexer.Lex(src))
	f.Fix(s)
	return s.Render()
}

// TestSetsKeepECSRuleConfiguration checks that a set runs its members with the
// options ECS's prepared set gives them, not with PHP-CS-Fixer's defaults.
func TestSetsKeepECSRuleConfiguration(t *testing.T) {
	cases := []struct {
		set     string
		fixers  []fixer.Fixer
		rule    string
		in, out string
	}{
		{"common", Common(), "YodaStyleFixer", "<?php\nif (null === $x) {}", "<?php\nif ($x === null) {}"},
		{"default", Default(), "YodaStyleFixer", "<?php\nif (null === $x) {}", "<?php\nif ($x === null) {}"},
	}
	for _, c := range cases {
		if got := fix(member(t, c.fixers, c.rule), c.in); got != c.out {
			t.Errorf("%s set, %s: got %q want %q", c.set, c.rule, got, c.out)
		}
	}
}
