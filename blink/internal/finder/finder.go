// Package finder collects .php files from configured paths, honoring skips.
package finder

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// alwaysSkipDirs are dependency directories never worth scanning.
var alwaysSkipDirs = map[string]bool{
	"vendor":       true,
	"node_modules": true,
	".git":         true,
}

// Find walks paths and returns every .php file not matching a skip glob, plus the
// count of distinct .php files discovered before skip globs are applied. ECS counts
// every discovered file in its run summary (skipped ones included), so the caller
// reports this total while only processing the returned files.
func Find(paths []string, skip []string) ([]string, int, error) {
	var out []string
	seen := map[string]bool{}
	discovered := 0

	for _, p := range paths {
		err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if alwaysSkipDirs[d.Name()] {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".php") {
				return nil
			}
			if seen[path] {
				return nil
			}
			seen[path] = true
			discovered++
			if skipped(path, skip) {
				return nil
			}
			out = append(out, path)
			return nil
		})
		if err != nil {
			return nil, 0, err
		}
	}
	return out, discovered, nil
}

// Skipped reports whether path matches any of the skip globs, using the same
// matching as the file walk (full-path glob, basename glob, or a "*fragment*"
// substring). Reused for per-path rule skips.
func Skipped(path string, skip []string) bool {
	return skipped(path, skip)
}

func skipped(path string, skip []string) bool {
	for _, pat := range skip {
		if ok, _ := filepath.Match(pat, path); ok {
			return true
		}
		if ok, _ := filepath.Match(pat, filepath.Base(path)); ok {
			return true
		}
		// "*/Fixture/*" matches the path segment "/Fixture/", not the substring
		// "Fixture" (which would wrongly skip e.g. "DataFixtures")
		if inner := strings.Trim(pat, "*"); inner != "" && strings.Contains(pat, "*") && strings.Contains(path, inner) {
			return true
		}
	}
	return false
}
