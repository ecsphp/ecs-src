// Package runner ties finder, lexer, token stream and fixers together, echoing
// ECS's application layer. Files are processed by a pool of parallel workers.
package runner

import (
	"os"
	"sort"
	"sync"

	"blink/internal/config"
	"blink/internal/diff"
	"blink/internal/finder"
	"blink/internal/lexer"
	"blink/internal/tokens"
)

// FileResult records the change made to one file.
type FileResult struct {
	Path         string
	AppliedRules []string // rules that changed this file, in application order
	Diff         string   // unified diff (Original vs fixed)
	After        string   // fixed content
}

func (r FileResult) Changed() bool { return len(r.AppliedRules) > 0 }

// Run scans the config's paths and applies rules across Jobs workers. When
// write is true, changed files are written back to disk. Results are returned
// sorted by path so output is deterministic regardless of worker scheduling.
// Progress receives per-file progress updates during a run.
type Progress interface {
	Start(total int)
	Advance()
}

func Run(cfg *config.Config, write bool, prog Progress) ([]FileResult, error) {
	files, err := finder.Find(cfg.Paths, cfg.Skip)
	if err != nil {
		return nil, err
	}
	if prog != nil {
		prog.Start(len(files))
	}

	jobs := max(cfg.Jobs, 1)

	paths := make(chan string)
	results := make(chan FileResult)
	var firstErr error
	var errOnce sync.Once

	var wg sync.WaitGroup
	for range jobs {
		wg.Go(func() {
			for path := range paths {
				res, err := fixFile(cfg, path, write)
				if prog != nil {
					prog.Advance()
				}
				if err != nil {
					errOnce.Do(func() { firstErr = err })
					continue
				}
				if res.Changed() {
					results <- res
				}
			}
		})
	}

	go func() {
		for _, p := range files {
			paths <- p
		}
		close(paths)
		wg.Wait()
		close(results)
	}()

	var collected []FileResult
	for r := range results {
		collected = append(collected, r)
	}
	if firstErr != nil {
		return nil, firstErr
	}

	sort.Slice(collected, func(i, j int) bool { return collected[i].Path < collected[j].Path })
	return collected, nil
}

func fixFile(cfg *config.Config, path string, write bool) (FileResult, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return FileResult{}, err
	}
	original := string(src)

	stream := tokens.New(lexer.Lex(original))
	res := FileResult{Path: path}

	for _, rule := range cfg.Rules {
		if rule.Fix(stream) {
			res.AppliedRules = append(res.AppliedRules, rule.Name())
		}
	}

	res.After = stream.Render()
	if !res.Changed() {
		return res, nil
	}
	// The unified diff is only rendered in check (dry-run) mode; --fix reports a
	// count and rewrites the file, so computing the diff there is wasted work.
	if !write {
		res.Diff = diff.Unified(original, res.After)
	}

	if write {
		info, statErr := os.Stat(path)
		mode := os.FileMode(0o644)
		if statErr == nil {
			mode = info.Mode()
		}
		if err := os.WriteFile(path, []byte(res.After), mode); err != nil {
			return FileResult{}, err
		}
	}
	return res, nil
}
