package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// portConstructsCase is one PHP-CS-Fixer test case: src must fix to want with
// the rule configured by cfg (nil: the defaults), and want must stay as it is.
type portConstructsCase struct {
	name string
	cfg  map[string]any
	src  string
	want string
}

func runPortConstructsCases(t *testing.T, base fixer.Fixer, cases []portConstructsCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := base
			if c.cfg != nil {
				f = base.(fixer.ConfigurableFixer).WithConfig(c.cfg)
			}
			got, changed := apply(t, f, c.src)
			if got != c.want {
				t.Errorf("fix mismatch\ninput:\n%s\ngot:\n%s\nwant:\n%s", c.src, got, c.want)
			}
			if changed != (c.src != c.want) {
				t.Errorf("changed=%v for input:\n%s", changed, c.src)
			}
			if again, changed := apply(t, f, c.want); changed || again != c.want {
				t.Errorf("expected output is not stable\ngot:\n%s\nwant:\n%s", again, c.want)
			}
		})
	}
}
