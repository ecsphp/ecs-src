// Command check-sources verifies that every registered fixer links to a
// reachable PHP-CS-Fixer source file on GitHub. It is run in CI so a renamed or
// removed upstream rule fails the build.
//
// Rather than one HEAD request per fixer (164 of them, which GitHub throttles
// with 503/429), it fetches the recursive file tree of each source repository
// in a single GitHub API call and checks every fixer path against that set.
// The last known-good tree is cached in sources_cache.json (committed and
// embedded) and used as a fallback when GitHub is unreachable. Run with
// -refresh to regenerate that committed cache.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"blink/internal/fixer"
	"blink/internal/fixer/rules"
)

//go:embed sources_cache.json
var embeddedCache []byte

const maxAttempts = 4

// sourceBases are the GitHub locations whose file trees back the fixer sources.
var sourceBases = []string{
	fixer.SourceBase,
	fixer.SymplifySourceBase,
	fixer.EcsSrcSourceBase,
}

// cache maps each source base URL to the set of file paths that exist under it,
// relative to the base (e.g. "Alias/NoAliasFunctionsFixer.php").
type cache map[string][]string

// treeResponse is the subset of the GitHub git-trees API we read.
type treeResponse struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
	Truncated bool `json:"truncated"`
}

// parseBase splits a source base URL into the GitHub owner, repo, branch, and
// the path prefix within the repo (e.g. "src/Fixer/").
func parseBase(base string) (owner, repo, branch, prefix string, err error) {
	rest, ok := strings.CutPrefix(base, "https://github.com/")
	if !ok {
		return "", "", "", "", fmt.Errorf("unexpected base URL %q", base)
	}
	parts := strings.SplitN(rest, "/", 5)
	if len(parts) < 5 || parts[2] != "blob" {
		return "", "", "", "", fmt.Errorf("cannot parse base URL %q", base)
	}
	return parts[0], parts[1], parts[3], parts[4], nil
}

// fetchTree returns the file paths under base, relative to the base prefix, by
// reading the repository's recursive git tree in one request.
func fetchTree(client *http.Client, base string) ([]string, error) {
	owner, repo, branch, prefix, err := parseBase(base)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, branch)

	var body []byte
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "blink-check-sources")
		req.Header.Set("Accept", "application/vnd.github+json")
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, lastErr = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK && lastErr == nil {
				lastErr = nil
				break
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("%s -> %d", url, resp.StatusCode)
			}
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				return nil, lastErr
			}
		}
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}

	var tree treeResponse
	if err := json.Unmarshal(body, &tree); err != nil {
		return nil, err
	}
	if tree.Truncated {
		return nil, fmt.Errorf("git tree for %s/%s truncated; cannot verify sources", owner, repo)
	}

	var paths []string
	for _, entry := range tree.Tree {
		if entry.Type != "blob" {
			continue
		}
		rel, ok := strings.CutPrefix(entry.Path, prefix)
		if ok && strings.HasSuffix(rel, ".php") {
			paths = append(paths, rel)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

// buildCache fetches the tree for every source base.
func buildCache(client *http.Client) (cache, error) {
	result := cache{}
	for _, base := range sourceBases {
		paths, err := fetchTree(client, base)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", base, err)
		}
		result[base] = paths
	}
	return result, nil
}

// refresh regenerates the committed cache file from live GitHub data.
func refresh(client *http.Client, path string) error {
	built, err := buildCache(client)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(built, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n", path)
	return nil
}

// sets turns a cache into fast lookup sets keyed by source base.
func sets(c cache) map[string]map[string]bool {
	out := make(map[string]map[string]bool, len(c))
	for base, paths := range c {
		set := make(map[string]bool, len(paths))
		for _, p := range paths {
			set[p] = true
		}
		out[base] = set
	}
	return out
}

func main() {
	refreshFlag := flag.Bool("refresh", false, "regenerate the committed sources_cache.json from live GitHub data")
	cachePath := flag.String("cache", "cmd/check-sources/sources_cache.json", "path to the sources cache file")
	flag.Parse()

	client := &http.Client{Timeout: 30 * time.Second}

	if *refreshFlag {
		if err := refresh(client, *cachePath); err != nil {
			fmt.Fprintf(os.Stderr, "refresh failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Prefer live data so a renamed or removed upstream rule is caught, and fall
	// back to the committed cache when GitHub is unreachable.
	current, err := buildCache(client)
	if err != nil {
		fmt.Printf("live fetch failed (%v); using committed cache\n", err)
		if err := json.Unmarshal(embeddedCache, &current); err != nil {
			fmt.Fprintf(os.Stderr, "cannot read committed cache: %v\n", err)
			os.Exit(1)
		}
	}
	lookup := sets(current)

	failed := false
	for _, f := range rules.All() {
		url := f.SourceURL()
		base, rel := splitSource(url)
		set, ok := lookup[base]
		switch {
		case !ok:
			fmt.Printf("FAIL %s: no source tree for %s\n", f.Name(), base)
			failed = true
		case !set[rel]:
			fmt.Printf("FAIL %s: %s missing from upstream\n", f.Name(), url)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
	fmt.Printf("All %d fixer sources reachable.\n", len(rules.All()))
}

// splitSource splits a fixer SourceURL into its known base and the path relative
// to that base.
func splitSource(url string) (base, rel string) {
	for _, b := range sourceBases {
		if after, ok := strings.CutPrefix(url, b); ok {
			return b, after
		}
	}
	return url, ""
}
