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

// Find walks paths and returns every .php file not matching a skip glob.
func Find(paths []string, skip []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}

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
			if skipped(path, skip) || seen[path] {
				return nil
			}
			seen[path] = true
			out = append(out, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func skipped(path string, skip []string) bool {
	for _, pat := range skip {
		if ok, _ := filepath.Match(pat, path); ok {
			return true
		}
		if ok, _ := filepath.Match(pat, filepath.Base(path)); ok {
			return true
		}
		if strings.Contains(path, strings.Trim(pat, "*/")) && strings.Contains(pat, "*") {
			return true
		}
	}
	return false
}
