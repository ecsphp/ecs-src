package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"blink/internal/config"
	"blink/internal/fixer/rules"
)

// Two rules that fight to a draw must not be reported as a change: UnaryOperatorSpaces
// (full) strips the space after "!" and NotOperatorWithSuccessorSpace adds it back,
// so the rendered file is identical to the original. PHP-CS-Fixer reports only real
// diffs, so blink must too (otherwise the file shows up with an empty diff).
func TestRunNetZeroNotReported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.php")
	src := "<?php\nif (! $a) {\n    echo 1;\n}\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Configure().WithPaths(dir).WithRules(
		rules.UnaryOperatorSpaces{Full: true},
		rules.NotOperatorWithSuccessorSpace{},
	)

	results, err := Run(context.Background(), cfg, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("net-zero change must not be reported, got %+v", results)
	}
	if b, _ := os.ReadFile(path); string(b) != src {
		t.Fatal("check mode must not modify the file")
	}
}
