package rules

import (
	"testing"

	"blink/internal/fixer"
)

// aliasUpstreamCase is one PHP-CS-Fixer fixer test case: the input and what
// PHP-CS-Fixer turns it into (equal to the input when nothing changes).
type aliasUpstreamCase struct {
	name     string
	cfg      map[string]any
	input    string
	expected string
}

// aliasRunUpstreamCases checks each case like PHP-CS-Fixer's doTest: the input
// fixes to the expected output, and the expected output is left alone.
func aliasRunUpstreamCases(t *testing.T, base fixer.Fixer, cases []aliasUpstreamCase) {
	t.Helper()
	for _, c := range cases {
		f := base
		if c.cfg != nil {
			cf, ok := base.(fixer.ConfigurableFixer)
			if !ok {
				t.Fatalf("%s: %s is not configurable", c.name, base.Name())
			}
			f = cf.WithConfig(c.cfg)
		}
		got, changed := apply(t, f, c.input)
		if got != c.expected {
			t.Errorf("%s:\ninput:    %q\ngot:      %q\nexpected: %q", c.name, c.input, got, c.expected)
			continue
		}
		if changed != (c.input != c.expected) {
			t.Errorf("%s: changed=%v, want %v", c.name, changed, c.input != c.expected)
		}
		if again, changed2 := apply(t, f, c.expected); changed2 || again != c.expected {
			t.Errorf("%s: expected output is not stable: %q -> %q", c.name, c.expected, again)
		}
	}
}
