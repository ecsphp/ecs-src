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

// Find walks paths and returns every .php file not matching a skip path, plus the
// count of distinct .php files discovered before skips are applied. ECS counts
// every discovered file in its run summary (skipped ones included), so the caller
// reports this total while only processing the returned files.
func Find(paths []string, skip []string) ([]string, int, error) {
	var out []string
	seen := map[string]bool{}
	discovered := 0
	skipped := NewPathMatcher(skip)

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
			if skipped.Match(path) {
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

// Skipped reports whether path matches any of the skip paths (see PathMatcher).
func Skipped(path string, skip []string) bool {
	return NewPathMatcher(skip).Match(path)
}
