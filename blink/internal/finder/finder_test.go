package finder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSkipsDependencyDirs(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"src/a.php":          "<?php",
		"src/sub/b.php":      "<?php",
		"vendor/c.php":       "<?php",
		"vendor/pkg/d.php":   "<?php",
		"node_modules/e.php": "<?php",
		".git/f.php":         "<?php",
		"src/notphp.txt":     "x",
	}
	for rel, body := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	found, discovered, err := Find([]string{root}, nil)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]bool{}
	for _, f := range found {
		rel, _ := filepath.Rel(root, f)
		got[rel] = true
	}

	want := []string{filepath.Join("src", "a.php"), filepath.Join("src", "sub", "b.php")}
	if len(got) != len(want) {
		t.Fatalf("found %v, want %v", found, want)
	}
	for _, w := range want {
		if !got[w] {
			t.Fatalf("missing %s in %v", w, found)
		}
	}
	// dependency dirs and non-.php files are not discovered either
	if discovered != len(want) {
		t.Fatalf("discovered = %d, want %d", discovered, len(want))
	}
}

func TestFindCountsSkippedFiles(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"src/a.php", "src/Fixture/b.php", "src/Fixture/c.php"} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("<?php"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	found, discovered, err := Find([]string{root}, []string{"*/Fixture/*"})
	if err != nil {
		t.Fatal(err)
	}
	// only a.php is processed, but all three .php files count toward the total
	if len(found) != 1 {
		t.Fatalf("found %v, want 1 file", found)
	}
	if discovered != 3 {
		t.Fatalf("discovered = %d, want 3 (skipped files still counted)", discovered)
	}
}

func TestSkippedSegmentNotSubstring(t *testing.T) {
	skip := []string{"*/Fixture/*", "*/node_modules/*"}
	cases := map[string]bool{
		"/app/Fixture/X.php":          true, // real Fixture segment
		"/app/node_modules/pkg/a.php": true,
		"/app/DataFixtures/ORM/X.php": false, // "Fixture" only as a substring
		"/app/InstallFixtures/X.php":  false,
		"/app/src/Fixtures.php":       false,
	}
	for path, want := range cases {
		if got := Skipped(path, skip); got != want {
			t.Errorf("Skipped(%q) = %v, want %v", path, got, want)
		}
	}
}
