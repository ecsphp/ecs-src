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

	"github.com/ecsphp/ecs-src/blink/internal/config"
	"github.com/ecsphp/ecs-src/blink/internal/diff"
	"github.com/ecsphp/ecs-src/blink/internal/finder"
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// FileResult records the change made to one file.
type FileResult struct {
	Path         string
	AppliedRules []string // rules that changed this file, in application order
	Diff         string   // unified diff (Original vs fixed)
	After        string   // fixed content
	// Errors are the errors found in the original contents, by line.
	Errors []fixer.CodingStandardError
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

	// prepare each rule's skip paths once, not per file
	ruleSkips := make(map[string][]*finder.PathMatcher, len(cfg.PerPathRuleSkips))
	for rule, groups := range cfg.PerPathRuleSkips {
		for _, patterns := range groups {
			ruleSkips[rule] = append(ruleSkips[rule], finder.NewPathMatcher(patterns))
		}
	}

	paths := make(chan string)
	results := make(chan FileResult)
	var mu sync.Mutex
	var errs []error

	var wg sync.WaitGroup
	for range jobs {
		wg.Go(func() {
			for path := range paths {
				res, err := fixFile(cfg, ruleSkips, path, write)
				if prog != nil {
					prog.Advance()
				}
				if err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					continue
				}
				if res.Changed() || len(res.Errors) > 0 {
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

func fixFile(cfg *config.Config, ruleSkips map[string][]*finder.PathMatcher, path string, write bool) (FileResult, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return FileResult{}, err
	}
	original := string(src)

	stream := tokens.New(lexer.Lex(original))
	stream.SetPath(path)
	res := FileResult{Path: path}

	for _, rule := range cfg.Rules {
		reporter, ok := rule.(fixer.ErrorReporter)
		if !ok {
			continue
		}
		if skippedFor(ruleSkips[rule.Name()], path) || !isCandidate(rule, stream) {
			continue
		}
		res.Errors = append(res.Errors, reporter.Errors(stream)...)
	}
	slices.SortStableFunc(res.Errors, func(a, b fixer.CodingStandardError) int { return cmp.Compare(a.Line, b.Line) })

	for _, rule := range cfg.Rules {
		if skippedFor(ruleSkips[rule.Name()], path) {
			continue
		}
		// a deprecated class configured beside its successor runs as a second
		// instance of the same fixer, which is listed once
		if isCandidate(rule, stream) && rule.Fix(stream) && !slices.Contains(res.AppliedRules, rule.Name()) {
			res.AppliedRules = append(res.AppliedRules, rule.Name())
		}
		// ECS does this after every fixer that changed the file. Blink rules may
		// clear tokens without reporting a change, so do it after every rule.
		stream.ClearEmptyTokens()
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
	// Both check and --fix list the per-file diff, matching ECS.
	res.Diff = diff.Unified(original, res.After)

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

// skippedFor reports whether every configured class behind a rule is skipped on
// path (see config.Config.PerPathRuleSkips); a rule without skips is not.
func skippedFor(matchers []*finder.PathMatcher, path string) bool {
	if len(matchers) == 0 {
		return false
	}
	for _, m := range matchers {
		if !m.Match(path) {
			return false
		}
	}
	return true
}

// isCandidate reports whether rule may concern the file: false only for a
// fixer.Candidate that rules the file out, as PHP-CS-Fixer's isCandidate.
func isCandidate(rule fixer.Fixer, s *tokens.Stream) bool {
	c, ok := rule.(fixer.Candidate)
	return !ok || c.IsCandidate(s)
}
