// Command list-universe prints the full set of fixers blink is measured
// against, one FQCN per line. It is the denominator for the README parity
// counter: every PHP-CS-Fixer fixer, every Symplify coding-standard fixer, and
// the PHP_CodeSniffer sniffs ECS references from its sets. blink is a subset of
// this universe, so coverage never exceeds 100%.
//
// The fixer lists come from the committed GitHub source-tree cache
// (cmd/check-sources/sources_cache.json); the sniff list is scraped from the
// ECS set definitions in the surrounding monorepo.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"blink/internal/fixer"
)

func main() {
	cachePath := flag.String("cache", "cmd/check-sources/sources_cache.json", "path to the source-tree cache")
	setsFlag := flag.String("sets", "../config/set,../src/Config/Level", "comma-separated dirs scanned for referenced PHP_CodeSniffer sniffs")
	flag.Parse()

	universe := map[string]bool{}

	cache, err := loadCache(*cachePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, path := range cache[fixer.SourceBase] {
		if fqcn := fixerFQCN(path, `PhpCsFixer\Fixer\`); fqcn != "" {
			universe[fqcn] = true
		}
	}
	for _, path := range cache[fixer.EcsSrcSourceBase] {
		if fqcn := fixerFQCN(path, `Symplify\CodingStandard\Fixer\`); fqcn != "" {
			universe[fqcn] = true
		}
	}

	for dir := range strings.SplitSeq(*setsFlag, ",") {
		for fqcn := range sniffsIn(strings.TrimSpace(dir)) {
			universe[fqcn] = true
		}
	}

	names := make([]string, 0, len(universe))
	for name := range universe {
		names = append(names, name)
	}
	sort.Strings(names)

	if _, err := os.Stdout.WriteString(strings.Join(names, "\n") + "\n"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadCache(path string) (map[string][]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cache: %w", err)
	}
	var cache map[string][]string
	if err := json.Unmarshal(raw, &cache); err != nil {
		return nil, fmt.Errorf("parse cache: %w", err)
	}
	return cache, nil
}

// fixerFQCN turns a cached source path ("Alias/ArrayPushFixer.php") into its
// FQCN under prefix, skipping abstract base classes and files outside a
// category subdirectory.
func fixerFQCN(path, prefix string) string {
	if !strings.HasSuffix(path, "Fixer.php") {
		return ""
	}
	class := filepath.Base(path)
	if strings.HasPrefix(class, "Abstract") {
		return ""
	}
	rel := strings.TrimSuffix(path, ".php")
	if !strings.Contains(rel, "/") {
		return ""
	}
	return prefix + strings.ReplaceAll(rel, "/", `\`)
}

var sniffRe = regexp.MustCompile(`PHP_CodeSniffer\\(?:Standards\\)?[A-Za-z0-9\\]+Sniff`)

// sniffsIn scrapes the PHP_CodeSniffer sniff FQCNs referenced under dir,
// dropping the bare Sniff interface.
func sniffsIn(dir string) map[string]bool {
	found := map[string]bool{}
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".php") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range sniffRe.FindAllString(string(raw), -1) {
			if m == `PHP_CodeSniffer\Sniffs\Sniff` {
				continue
			}
			found[m] = true
		}
		return nil
	})
	return found
}
