package runner

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"blink/internal/config"
	"blink/internal/fixer"
	"blink/internal/fixer/rules"
)

// TestRunPerRuleSkipPaths checks that a rule skipped on some paths stays off
// in every file ECS's skipper excludes: a "*" glob reaches into subfolders and
// a directory covers everything below it.
func TestRunPerRuleSkipPaths(t *testing.T) {
	const ruleName = `PhpCsFixer\Fixer\Alias\NoMixedEchoPrintFixer`
	rule, ok := rules.ByName(ruleName)
	if !ok {
		t.Fatalf("rule %s not found", ruleName)
	}

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"old/a.php", "old/sub/b.php", "web/handler/c.php", "web/other/d.php", "e.php"} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("<?php print 'x';\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name   string
		groups [][]string // one skip list per configured class resolving to the rule
		want   []string
	}{
		{"glob reaches subfolders", [][]string{{filepath.Join(root, "old/*.php")}}, []string{"e.php", "web/handler/c.php", "web/other/d.php"}},
		{"directory", [][]string{{filepath.Join(root, "web/handler")}}, []string{"e.php", "old/a.php", "old/sub/b.php", "web/other/d.php"}},
		{"no paths matched", [][]string{{filepath.Join(root, "nope")}}, []string{"e.php", "old/a.php", "old/sub/b.php", "web/handler/c.php", "web/other/d.php"}},
		// a deprecated alias and its successor both configured: the rule is off
		// only where both classes are skipped
		{"two classes", [][]string{{filepath.Join(root, "old")}, {filepath.Join(root, "*/b.php"), filepath.Join(root, "web")}}, []string{"e.php", "old/a.php", "web/handler/c.php", "web/other/d.php"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &config.Config{
				Paths:            []string{root},
				Rules:            []fixer.Fixer{rule},
				Jobs:             1,
				PerPathRuleSkips: map[string][][]string{ruleName: c.groups},
			}
			results, err := Run(context.Background(), cfg, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, r := range results {
				rel, _ := filepath.Rel(root, r.Path)
				got = append(got, filepath.ToSlash(rel))
			}
			sort.Strings(got)
			if len(got) != len(c.want) {
				t.Fatalf("changed %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("changed %v, want %v", got, c.want)
				}
			}
		})
	}
}
