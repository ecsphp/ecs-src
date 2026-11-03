package finder

import (
	"os"
	"path/filepath"
	"slices"
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

	found, discovered, err := Find([]string{root}, []string{filepath.FromSlash("*/Fixture/*")})
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

// Expected values are ECS's SourceFinder and FileInfoMatcher run under PHP on
// the same tree.
func TestFindBuildsPathsLikeSymfonyFinder(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"src/Legacy/A.php", "src/LegacyOld/B.php", "src/C.php", "lib/D.php"} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("<?php echo 1;\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	abs := root + "/src"

	cases := []struct {
		cwd    string
		source string
		skip   string
		want   []string // kept files, in walk order
	}{
		{"", "./src", "src/Legacy", []string{"./src/C.php", "./src/LegacyOld/B.php"}},
		{"", "./src", "src/Leg", []string{"./src/C.php", "./src/Legacy/A.php", "./src/LegacyOld/B.php"}},
		{"", "./src", "Legacy/A.php", []string{"./src/C.php", "./src/LegacyOld/B.php"}},
		{"", "./src", "src/Legacy/*", []string{"./src/C.php", "./src/LegacyOld/B.php"}},
		// the prefix quirk: "src/Legacy" is a string prefix of "src/LegacyOld/B.php"
		{"", "src/", "src/Legacy", []string{"src/C.php"}},
		{"", "src/", "src/Leg", []string{"src/C.php"}},
		{"", "src", "src/Legacy", []string{"src/C.php"}},
		{"", "src", "Legacy/A.php", []string{"src/C.php", "src/LegacyOld/B.php"}},
		{"", "src//", "src/Legacy/*", []string{"src/C.php", "src/LegacyOld/B.php"}},
		{"", abs, "src/Legacy", []string{abs + "/C.php", abs + "/LegacyOld/B.php"}},
		{"", abs + "/", "src/Leg", []string{abs + "/C.php", abs + "/Legacy/A.php", abs + "/LegacyOld/B.php"}},
		{"lib", "../src", "../src/Legacy", []string{"../src/C.php", "../src/LegacyOld/B.php"}},
		{"lib", "../src", "src/Leg", []string{"../src/C.php", "../src/Legacy/A.php", "../src/LegacyOld/B.php"}},
		{"lib", ".", "Legacy/A.php", []string{"./D.php"}},
		{"lib", "./", "Legacy/A.php", []string{"./D.php"}},
	}
	for _, c := range cases {
		t.Run(c.cwd+"|"+c.source+"|"+c.skip, func(t *testing.T) {
			t.Chdir(filepath.Join(root, c.cwd))
			found, discovered, err := Find([]string{c.source}, []string{c.skip})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(found, c.want) {
				t.Fatalf("Find(%q, skip %q) = %q, want %q", c.source, c.skip, found, c.want)
			}
			wantDiscovered := 3
			if c.cwd == "lib" && (c.source == "." || c.source == "./") {
				wantDiscovered = 1
			}
			if discovered != wantDiscovered {
				t.Fatalf("discovered = %d, want %d", discovered, wantDiscovered)
			}
		})
	}
}
