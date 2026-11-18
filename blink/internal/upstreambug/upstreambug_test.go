// Package upstreambug pins blink to PHP-CS-Fixer bugs it copies on purpose. Each
// case is testdata/cases/<name>/{input.php, expected.php, checkers.json, README.md}:
// expected.php is PHP-CS-Fixer's current, buggy output, also pinned on the PHP side
// by tests/Blink/UpstreamBugFixtureTest.php. When upstream fixes the bug, that test
// fails: update expected.php, then port the fix here until this test passes again.
package upstreambug

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/config"
	"github.com/ecsphp/ecs-src/blink/internal/runner"
)

func TestUpstreamBugs(t *testing.T) {
	cases, err := filepath.Glob("testdata/cases/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no upstream bug cases found")
	}

	for _, dir := range cases {
		t.Run(filepath.Base(dir), func(t *testing.T) {
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

			cfg, _, err := config.LoadECS(filepath.Join(dir, "checkers.json"))
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
				t.Errorf("blink must copy PHP-CS-Fixer here, see %s/README.md\n--- blink\n%s\n--- PHP-CS-Fixer (expected)\n%s", dir, got, expected)
			}
		})
	}
}
