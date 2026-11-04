package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/config"
	"github.com/ecsphp/ecs-src/blink/internal/fixer/rules"
)

// TestRunReportsSniffErrors keeps a file whose only finding is a non-fixable
// sniff error in the results, with the error on the original line, and leaves
// it untouched in fix mode.
func TestRunReportsSniffErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.php")
	src := "<?php\n\nfunction f()\n{\n    global $x;\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Configure().WithPaths(dir).WithRules(rules.GlobalKeyword{}, rules.ArraySyntax{})

	for _, write := range []bool{false, true} {
		results, err := Run(context.Background(), cfg, write, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].Changed() || len(results[0].Errors) != 1 || results[0].Errors[0].Line != 5 {
			t.Fatalf("write=%v: want one unchanged file with one error on line 5, got %+v", write, results)
		}
		if b, _ := os.ReadFile(path); string(b) != src {
			t.Fatalf("write=%v: the file must not change", write)
		}
	}
}
