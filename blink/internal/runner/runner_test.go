package runner

import (
	"os"
	"path/filepath"
	"testing"

	"blink/internal/config"
)

func TestRunFixesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.php")
	dirty := "<?php $x = 1 ;   \n\n\n"
	if err := os.WriteFile(path, []byte(dirty), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Configure().WithPaths(dir)

	// check mode: reports but does not write
	results, err := Run(cfg, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Changed() {
		t.Fatalf("check: expected 1 changed file, got %+v", results)
	}
	if b, _ := os.ReadFile(path); string(b) != dirty {
		t.Fatal("check mode must not modify the file")
	}

	// fix mode: writes cleaned content
	if _, err := Run(cfg, true, nil); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	// default set is psr12 + common: the space before ";", the trailing
	// whitespace and the extra blank lines are fixed; inline code is left on the
	// "<?php" line (linebreak_after_opening_tag is not part of psr12 + common)
	want := "<?php $x = 1;\n"
	if string(b) != want {
		t.Fatalf("fix: got %q want %q", string(b), want)
	}

	// second check run is clean
	results, _ = Run(cfg, false, nil)
	if len(results) != 0 {
		t.Fatalf("expected clean after fix, got %+v", results)
	}
}
