// Package finder collects .php files from configured paths, honoring skips.
package finder

import (
	"io/fs"
	"os"
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
//
// Skips match paths as strings, so paths are built like Symfony Finder does:
// the source as typed, trailing "/" trimmed, then "/" and the path below it.
func Find(paths []string, skip []string) ([]string, int, error) {
	var out []string
	seen := map[string]bool{}
	discovered := 0
	skipped := NewPathMatcher(skip)

	visit := func(path string) {
		if !strings.HasSuffix(path, ".php") || seen[path] {
			return
		}
		seen[path] = true
		discovered++
		if !skipped.Match(path) {
			out = append(out, path)
		}
	}

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, 0, err
		}
		if !info.IsDir() {
			visit(p)
			continue
		}
		if err := walk(finderBase(p), visit); err != nil {
			return nil, 0, err
		}
	}
	return out, discovered, nil
}

// finderBase is the prefix Symfony Finder puts before the paths below dir: dir
// with trailing "/" trimmed (except the root "/" itself), then "/".
func finderBase(dir string) string {
	if dir == "/" {
		return dir
	}
	return strings.TrimRight(dir, "/") + "/"
}

// walk visits every file below base (which ends in "/") in lexical order,
// building each path by appending to base. Like Finder without followLinks, it
// does not descend into symlinked directories, and it never enters a
// dependency directory below the source.
func walk(base string, visit func(path string)) error {
	entries, err := os.ReadDir(base)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := base + entry.Name()
		if entry.IsDir() {
			if alwaysSkipDirs[entry.Name()] {
				continue
			}
			if err := walk(path+"/", visit); err != nil {
				return err
			}
			continue
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			// Finder's files() keeps a link only when it resolves to a file
			if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
				continue
			}
		}
		visit(path)
	}
	return nil
}

// Skipped reports whether path matches any of the skip paths (see PathMatcher).
func Skipped(path string, skip []string) bool {
	return NewPathMatcher(skip).Match(path)
}
