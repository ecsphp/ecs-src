package finder

import (
	"os"
	"path/filepath"
	"testing"
)

// skipTree builds a small project and returns its root.
func skipTree(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"old/a.php",
		"old/sub/b.php",
		"old/sub/deep/c.php",
		"old2/d.php",
		"src/Web/Handler/H.php",
		"src/Web/HandlerTest/T.php",
		"src/Foo.php",
		"src/BarFoo.php",
		"src/Zoo.php",
	} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("<?php\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestSkippedMatchesECS mirrors ECS's FileInfoMatcher: exact path, prefix,
// suffix, fnmatch (where "*" crosses "/"), then "*/"-relative fnmatch. Rows
// marked "quirk" are ECS behaviour that looks surprising but is what the PHP
// engine does, so blink must do it too.
func TestSkippedMatchesECS(t *testing.T) {
	root := skipTree(t)
	t.Chdir(root)
	abs := func(rel string) string { return root + "/" + rel }

	cases := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		// a file given by its absolute path
		{"abs file itself", abs("src/Foo.php"), abs("src/Foo.php"), true},
		{"abs file other", abs("src/Foo.php"), abs("src/BarFoo.php"), false},

		// "*" crosses "/" in fnmatch
		{"glob same dir", abs("old/*.php"), abs("old/a.php"), true},
		{"glob subdir", abs("old/*.php"), abs("old/sub/b.php"), true},
		{"glob deep subdir", abs("old/*.php"), abs("old/sub/deep/c.php"), true},
		{"glob sibling dir", abs("old/*.php"), abs("old2/d.php"), false},
		{"glob unrelated", abs("old/*.php"), abs("src/Foo.php"), false},

		// "**" is two "*": it needs the "/" that follows it in the pattern
		{"double star subdir", abs("old/**/*.php"), abs("old/sub/b.php"), true},
		{"double star top level", abs("old/**/*.php"), abs("old/a.php"), false},

		// a directory skips everything below it
		{"abs dir child", abs("old"), abs("old/a.php"), true},
		{"abs dir deep child", abs("old"), abs("old/sub/deep/c.php"), true},
		{"abs dir unrelated", abs("old"), abs("src/Foo.php"), false},
		// quirk: plain string prefix, so a directory also skips a sibling it prefixes
		{"abs dir prefix sibling (quirk)", abs("old"), abs("old2/d.php"), true},
		{"abs dir prefix sibling dir (quirk)", abs("src/Web/Handler"), abs("src/Web/HandlerTest/T.php"), true},
		{"abs dir no prefix", abs("src/Web/Handler"), abs("src/Foo.php"), false},

		// trailing slash: the prefix then stops at the directory
		{"trailing slash child", abs("old") + "/", abs("old/sub/b.php"), true},
		{"trailing slash sibling", abs("old") + "/", abs("old2/d.php"), false},

		// relative patterns: "*/" fnmatch or suffix
		{"relative dir sibling", "old", abs("old2/d.php"), false},
		{"relative glob", "old/*.php", abs("old/sub/deep/c.php"), true},
		{"relative glob sibling", "old/*.php", abs("old2/d.php"), false},
		{"relative file suffix", "src/Foo.php", abs("src/Foo.php"), true},
		{"relative file other", "src/Foo.php", abs("src/Zoo.php"), false},
		// quirk: plain string suffix, so a bare name also skips a longer one
		{"bare name suffix (quirk)", "Foo.php", abs("src/BarFoo.php"), true},
		{"relative dir missing", "nope", abs("old/a.php"), false},

		// a leading or trailing "*" wraps the pattern in "*…*"
		{"leading star segment", "*/sub/*", abs("old/sub/b.php"), true},
		{"leading star no segment", "*/sub/*", abs("old/a.php"), false},
		{"trailing star prefix", abs("old") + "*", abs("old2/d.php"), true},
		{"trailing star relative", "src/Web*", abs("src/Web/Handler/H.php"), true},
		{"trailing star relative miss", "src/Web*", abs("src/Foo.php"), false},

		// fnmatch's "?" and bracket expressions
		{"question mark", abs("src/?oo.php"), abs("src/Zoo.php"), true},
		{"question mark miss", abs("src/?oo.php"), abs("src/BarFoo.php"), false},
		{"bracket", abs("src/[FZ]oo.php"), abs("src/Zoo.php"), true},
		{"bracket negated", abs("src/[!FZ]oo.php"), abs("src/Zoo.php"), false},
		{"bracket range", abs("src/[A-Z]oo.php"), abs("src/Foo.php"), true},

		{"empty pattern", "", abs("src/Foo.php"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Skipped(c.path, []string{c.pattern}); got != c.want {
				t.Errorf("Skipped(%q, [%q]) = %v, want %v", c.path, c.pattern, got, c.want)
			}
		})
	}
}

func TestFindSkipsDirectoriesAndGlobs(t *testing.T) {
	root := skipTree(t)
	cases := []struct {
		name string
		skip []string
		want []string
	}{
		{"glob crosses dirs", []string{filepath.Join(root, "old/*.php")}, []string{"old2/d.php", "src/BarFoo.php", "src/Foo.php", "src/Web/Handler/H.php", "src/Web/HandlerTest/T.php", "src/Zoo.php"}},
		{"directory", []string{filepath.Join(root, "src") + "/"}, []string{"old/a.php", "old/sub/b.php", "old/sub/deep/c.php", "old2/d.php"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			found, _, err := Find([]string{root}, c.skip)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, f := range found {
				rel, _ := filepath.Rel(root, f)
				got = append(got, filepath.ToSlash(rel))
			}
			if len(got) != len(c.want) {
				t.Fatalf("found %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("found %v, want %v", got, c.want)
				}
			}
		})
	}
}

// TestFnmatch pins fnmatch to PHP's fnmatch() with no flags (values taken from
// PHP): "*" and "?" cross "/", and bracket expressions follow glibc.
func TestFnmatch(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"*", "a/b/.c", true},
		{"a*b", "a/x/b", true},
		{"a?c", "a/c", true},
		{"[]]x", "]x", true},
		{"[!]]x", "]x", false},
		{"[!]]x", "ax", true},
		{"[^a]x", "ax", false},
		{"[a-]x", "-x", true},
		{"[[:digit:]]x", "7x", true},
		{"[[:alpha:]]x", "7x", false},
		{"a[bc", "a[bc", true},
		{"a[bc", "ab", false},
		{"*.php", "x.php.bak", false},
		{"*.php*", "x.php.bak", true},
		{"a**b", "ab", true},
		{"", "", true},
		{"", "a", false},
	}
	for _, c := range cases {
		if got := fnmatch(c.pattern, c.name); got != c.want {
			t.Errorf("fnmatch(%q, %q) = %v, want %v", c.pattern, c.name, got, c.want)
		}
	}
}
