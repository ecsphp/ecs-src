package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// upstreamCase is one case of a PHP-CS-Fixer fixer test's data provider:
// fixing input must give expected, and expected must be left as is. An empty
// input means expected is already fixed.
type upstreamCase struct {
	name     string
	cfg      map[string]any
	expected string
	input    string
}

// runUpstreamCases runs PHP-CS-Fixer's own test cases for a ported fixer.
func runUpstreamCases(t *testing.T, base fixer.Fixer, cases []upstreamCase) {
	t.Helper()
	for _, c := range cases {
		f := base
		if c.cfg != nil {
			configurable, ok := base.(fixer.ConfigurableFixer)
			if !ok {
				t.Fatalf("%s: %s takes no configuration", c.name, base.Name())
			}
			f = configurable.WithConfig(c.cfg)
		}
		if c.input != "" {
			if got, _ := apply(t, f, c.input); got != c.expected {
				t.Errorf("%s:\ninput:\n%s\ngot:\n%s\nwant:\n%s", c.name, c.input, got, c.expected)
			}
		}
		if got, changed := apply(t, f, c.expected); changed || got != c.expected {
			t.Errorf("%s: changed already fixed code (changed=%v):\n%s\ngot:\n%s", c.name, changed, c.expected, got)
		}
	}
}
