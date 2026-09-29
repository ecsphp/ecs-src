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

	found, err := Find([]string{root}, nil)
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
}
