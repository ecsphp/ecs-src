package rules

import (
	"testing"

	"blink/internal/fixer"
)

// phpCsFixerCase is one PHP-CS-Fixer doTest() case: input must fix to
// expected, and expected must stay as it is.
type phpCsFixerCase struct {
	name     string
	expected string
	input    string
	config   map[string]any
}

func runPhpCsFixerCases(t *testing.T, base fixer.ConfigurableFixer, cases []phpCsFixerCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := base.WithConfig(c.config)
			if got, _ := apply(t, f, c.input); got != c.expected {
				t.Errorf("fix mismatch\ninput:\n%s\ngot:\n%s\nwant:\n%s", c.input, got, c.expected)
			}
			if c.input != c.expected {
				if got, changed := apply(t, f, c.expected); changed || got != c.expected {
					t.Errorf("expected output is not stable\ngot:\n%s\nwant:\n%s", got, c.expected)
				}
			}
		})
	}
}
