// Package parityfixture runs blink's full configured pipeline over small fixtures
// that still diverge from PHP ECS, reproducing each divergence locally without
// cloning and fixing a whole corpus. A case is a standing bug: once blink matches
// ECS, the case is removed (the fix keeps its own unit test), so an empty set means
// no known divergences remain.
//
// Each case is testdata/cases/<name>/{input.php, expected.php}: input.php is fed
// through the same rule set the parity run uses (testdata/checkers.json, dumped via
// `bin/ecs dump-config`), and expected.php is the authoritative PHP ECS output. A
// mismatch means blink diverges from ECS on that case.
package parityfixture

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"blink/internal/config"
	"blink/internal/runner"
)

func TestParityFixtures(t *testing.T) {
	cases, err := filepath.Glob("testdata/cases/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		// only cases that still diverge from PHP ECS live here; a fix removes its
		// case (the fix keeps its own unit test), so an empty set is the goal state
		t.Skip("no outstanding parity divergences")
	}

	for _, dir := range cases {
		name := filepath.Base(dir)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input, err := os.ReadFile(filepath.Join(dir, "input.php"))
			if err != nil {
				t.Fatal(err)
			}
			expected, err := os.ReadFile(filepath.Join(dir, "expected.php"))
			if err != nil {
				t.Fatal(err)
			}

			tmp := filepath.Join(t.TempDir(), "subject.php")
			if err := os.WriteFile(tmp, input, 0o644); err != nil {
				t.Fatal(err)
			}

			cfg, _, err := config.LoadECS("testdata/checkers.json")
			if err != nil {
				t.Fatal(err)
			}
			cfg.WithPaths(tmp)
			if _, err := runner.Run(context.Background(), cfg, true, nil); err != nil {
				t.Fatal(err)
			}

			got, err := os.ReadFile(tmp)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(expected) {
				t.Errorf("blink output diverges from PHP ECS\n--- blink\n%s\n--- ecs (expected)\n%s", got, expected)
			}
		})
	}
}
