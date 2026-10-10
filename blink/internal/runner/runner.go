// Package runner ties finder, lexer, token stream and fixers together, echoing
// ECS's application layer. Files are processed by a pool of parallel workers.
package runner

import (
	"cmp"
	"context"
	"errors"
	"os"
	"slices"
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

// Progress receives per-file progress updates during a run.
type Progress interface {
	Start(total int)
	Advance()
}

// Run scans the config's paths and applies rules across Jobs workers, stopping
// early when ctx is cancelled. Every file error is collected (not just the
// first) and returned joined.
func Run(ctx context.Context, cfg *config.Config, write bool, prog Progress) ([]FileResult, error) {
	files, discovered, err := finder.Find(cfg.Paths, cfg.Skip)
	if err != nil {
		return nil, err
	}
	if prog != nil {
		// discovered includes skip-matched files, mirroring ECS's run summary
		prog.Start(discovered)
	}

	jobs := max(cfg.Jobs, 1)

	paths := make(chan string)
	results := make(chan FileResult)
	var mu sync.Mutex
	var errs []error

	var wg sync.WaitGroup
	for range jobs {
		wg.Go(func() {
			for path := range paths {
				res, err := fixFile(cfg, path, write)
				if prog != nil {
					prog.Advance()
				}
				if err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					continue
				}
				if res.Changed() {
					results <- res
				}
			}
		})
	}

	// Feed paths until exhausted or cancelled, then let the workers finish.
	go func() {
		defer close(paths)
		for _, p := range files {
			select {
			case paths <- p:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var collected []FileResult
	for r := range results {
		collected = append(collected, r)
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	slices.SortFunc(collected, func(a, b FileResult) int { return cmp.Compare(a.Path, b.Path) })
	return collected, nil
}

func fixFile(cfg *config.Config, path string, write bool) (FileResult, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return FileResult{}, err
	}
	original := string(src)

	stream := tokens.New(lexer.Lex(original))
	stream.SetPath(path)
	res := FileResult{Path: path}

	for _, rule := range cfg.Rules {
		if skipPatterns, ok := cfg.PerPathRuleSkips[rule.Name()]; ok && finder.Skipped(path, skipPatterns) {
			continue
		}
		if rule.Fix(stream) {
			res.AppliedRules = append(res.AppliedRules, rule.Name())
		}
	}

	res.After = stream.Render()
	// Rules that fight to a draw (one edits, another reverts) leave the rendered
	// file identical to the original; like PHP-CS-Fixer, report only real diffs.
	if res.After == original {
		res.AppliedRules = nil
	}
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
