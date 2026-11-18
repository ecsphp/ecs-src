package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/config"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// breakingFixer turns every "1" into "1 ..." (no longer parses) and every "2"
// into "3" (still parses), to stand in for a buggy and a correct fixer.
type breakingFixer struct{}

func (breakingFixer) Name() string      { return "Test\\BreakingFixer" }
func (breakingFixer) SourceURL() string { return "" }
func (breakingFixer) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		switch s.At(i).Value {
		case "1":
			s.SetValue(i, "1 ...")
			changed = true
		case "2":
			s.SetValue(i, "3")
			changed = true
		}
	}
	return changed
}

func requirePHP(t *testing.T) string {
	t.Helper()
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("php not on PATH")
	}
	return php
}

func writeFile(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A fix that leaves a file unparseable is not written and is reported as an
// error, as PHP-CS-Fixer lints each file after fixing it.
func TestRunFixSkipsUnparseableResult(t *testing.T) {
	php := requirePHP(t)
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.php")
	fine := filepath.Join(dir, "fine.php")
	brokenSrc := "<?php\n\n$a = 1;\n"
	fineSrc := "<?php\n\n$a = 2;\n"
	writeFile(t, broken, brokenSrc)
	writeFile(t, fine, fineSrc)

	cfg := config.Configure().WithPaths(dir).WithRules(breakingFixer{})
	cfg.PHPBinary = php

	results, err := Run(context.Background(), cfg, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %+v", results)
	}

	brokenRes := results[0]
	if brokenRes.Changed() || len(brokenRes.Errors) != 1 {
		t.Fatalf("unparseable fix must be reported as one error, not a change: %+v", brokenRes)
	}
	e := brokenRes.Errors[0]
	if e.Checker != LintChecker || e.Line != 3 || !strings.Contains(e.Message, "Test\\BreakingFixer") {
		t.Fatalf("unexpected error %+v", e)
	}
	if b, _ := os.ReadFile(broken); string(b) != brokenSrc {
		t.Fatalf("unparseable fix must not be written, got %q", b)
	}

	if !results[1].Changed() {
		t.Fatalf("parseable fix must still apply: %+v", results[1])
	}
	if b, _ := os.ReadFile(fine); string(b) != "<?php\n\n$a = 3;\n" {
		t.Fatalf("parseable fix must be written, got %q", b)
	}
}

// A file that did not parse before fixing is not blamed on the fixers.
func TestRunFixWritesWhenOriginalUnparseable(t *testing.T) {
	php := requirePHP(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.php")
	writeFile(t, path, "<?php\n\n$a = 1 +;\n")

	cfg := config.Configure().WithPaths(dir).WithRules(breakingFixer{})
	cfg.PHPBinary = php

	results, err := Run(context.Background(), cfg, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Changed() || len(results[0].Errors) != 0 {
		t.Fatalf("want the fix applied without errors, got %+v", results)
	}
}

// Without a PHP binary there is no lint and the fix is written as before.
func TestRunFixWithoutPHPBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.php")
	writeFile(t, path, "<?php\n\n$a = 1;\n")

	cfg := config.Configure().WithPaths(dir).WithRules(breakingFixer{})

	results, err := Run(context.Background(), cfg, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Changed() {
		t.Fatalf("want the fix applied, got %+v", results)
	}
}

