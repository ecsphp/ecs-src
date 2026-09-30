// Package config mirrors ECSConfig: paths to scan, skip patterns, active rules.
// A config can be built fluently in Go or loaded from a blink.json file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"blink/internal/fixer"
	"blink/internal/fixer/rules"
	"blink/internal/set"
)

type Config struct {
	Paths []string
	Skip  []string // filepath.Match globs tested against each path
	Rules []fixer.Fixer
	Jobs  int // parallel workers
	// PerPathRuleSkips disables a single rule on a subset of paths, keyed by the
	// rule's name (its PHP-CS-Fixer FQCN); mirrors ECS's per-path skip config.
	PerPathRuleSkips map[string][]string
}

// Configure returns a config seeded with the default psr12 + common sets,
// echoing ECSConfig::configure()->withPreparedSets(psr12: true, common: true).
func Configure() *Config {
	return &Config{
		Paths: []string{"."},
		Rules: set.Default(),
		Jobs:  runtime.NumCPU(),
	}
}

func (c *Config) WithPaths(paths ...string) *Config {
	if len(paths) > 0 {
		c.Paths = paths
	}
	return c
}

func (c *Config) WithSkip(patterns ...string) *Config {
	c.Skip = append(c.Skip, patterns...)
	return c
}

func (c *Config) WithRules(rs ...fixer.Fixer) *Config {
	c.Rules = rs
	return c
}

// file is the on-disk blink.json shape.
type file struct {
	Paths []string       `json:"paths"`
	Skip  []string       `json:"skip"`
	Sets  []string       `json:"sets"`
	Level map[string]int `json:"level"`
	Rules []string       `json:"rules"`
}

// Load reads and resolves a blink.json config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("config %s: %w", path, err)
	}
	return resolve(f)
}

// resolve turns a config file into a Config, collecting rules from sets, levels
// and explicit names, de-duplicated and kept in canonical (safest-first) order.
func resolve(f file) (*Config, error) {
	c := &Config{Paths: f.Paths, Skip: f.Skip, Jobs: runtime.NumCPU()}
	if len(c.Paths) == 0 {
		c.Paths = []string{"."}
	}

	wanted := map[string]bool{}
	selective := false

	for _, name := range f.Sets {
		fixers, ok := set.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown set %q", name)
		}
		selective = true
		for _, fx := range fixers {
			wanted[fx.Name()] = true
		}
	}

	for name, n := range f.Level {
		if name != "spaces" {
			return nil, fmt.Errorf("unknown level %q (only \"spaces\" is supported)", name)
		}
		selective = true
		for _, fx := range set.SpacesLevel(n) {
			wanted[fx.Name()] = true
		}
	}

	for _, name := range f.Rules {
		if _, ok := rules.ByName(name); !ok {
			return nil, fmt.Errorf("unknown rule %q", name)
		}
		selective = true
		wanted[name] = true
	}

	if !selective {
		c.Rules = set.Default()
		return c, nil
	}
	for _, fx := range rules.All() {
		if wanted[fx.Name()] {
			c.Rules = append(c.Rules, fx)
		}
	}
	return c, nil
}
