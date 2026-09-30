package config

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"

	"blink/internal/fixer"
	"blink/internal/fixer/rules"
)

// Turbo mode: ECS dumps its resolved ecs.php configuration to JSON (paths, rules,
// skips) with `ecs dump-config`, and blink consumes it here. Every blink fixer
// is named by its PHP-CS-Fixer FQCN, which is exactly the class ECS reports, so
// the rules map by name without a translation table.

// ecsConfiguredRule is one ECS sniff or fixer: a fully qualified class name and
// its configuration.
type ecsConfiguredRule struct {
	Class  string         `json:"class"`
	Config map[string]any `json:"config"`
}

// ecsSkip is one ECS skip entry: a path to exclude, a rule to disable, or a rule
// disabled only on some paths.
type ecsSkip struct {
	Path  string   `json:"path"`
	Class string   `json:"class"`
	Paths []string `json:"paths"`
}

// ecsFile is the on-disk shape of `ecs dump-config` output.
type ecsFile struct {
	PHPVersion string              `json:"php_version"`
	Paths      []string            `json:"paths"`
	Rules      []ecsConfiguredRule `json:"rules"`
	Skips      []ecsSkip           `json:"skips"`
}

// ECSResolution reports how an ECS config mapped onto blink, so a turbo run is
// never silently narrower than the config it stands in for.
type ECSResolution struct {
	// Total is the number of ECS rules in the config.
	Total int
	// Mapped is the number of rules resolved to a blink fixer.
	Mapped int
	// Unsupported are rule classes blink has no fixer for.
	Unsupported []string
	// ConfigIgnored are mapped rules that carried configuration; blink applies
	// its built-in behaviour for the fixer rather than the ECS options.
	ConfigIgnored []string
	// PerPathSkips are class-on-path skips applied project-wide, since blink
	// cannot yet skip a single rule on a subset of files.
	PerPathSkips []string
}

// LoadECS reads an `ecs dump-config` JSON file and resolves it to a blink
// config, returning a report of what could not be mapped.
func LoadECS(path string) (*Config, *ECSResolution, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var f ecsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, nil, fmt.Errorf("ECS config %s: %w", path, err)
	}

	return resolveECS(f)
}

// resolveECS turns an ECS dump-config into a Config plus a mapping report.
func resolveECS(f ecsFile) (*Config, *ECSResolution, error) {
	config := &Config{Paths: f.Paths, Jobs: runtime.NumCPU()}
	if len(config.Paths) == 0 {
		config.Paths = []string{"."}
	}

	resolution := &ECSResolution{Total: len(f.Rules)}

	skippedClasses := map[string]bool{}
	for _, skip := range f.Skips {
		switch {
		case skip.Class != "" && skip.Path == "" && len(skip.Paths) == 0:
			skippedClasses[skip.Class] = true
		case skip.Class != "":
			// A rule disabled only on some paths: skip that rule on those paths,
			// mirroring ECS, instead of dropping it project-wide.
			if config.PerPathRuleSkips == nil {
				config.PerPathRuleSkips = map[string][]string{}
			}
			if skip.Path != "" {
				config.PerPathRuleSkips[skip.Class] = append(config.PerPathRuleSkips[skip.Class], skip.Path)
			}
			config.PerPathRuleSkips[skip.Class] = append(config.PerPathRuleSkips[skip.Class], skip.Paths...)
			resolution.PerPathSkips = append(resolution.PerPathSkips, skip.Class)
		default:
			if skip.Path != "" {
				config.Skip = append(config.Skip, skip.Path)
			}
			config.Skip = append(config.Skip, skip.Paths...)
		}
	}

	seen := map[string]bool{}
	for _, rule := range f.Rules {
		if skippedClasses[rule.Class] {
			continue
		}

		resolved, ok := rules.ByName(rule.Class)
		if !ok {
			resolution.Unsupported = append(resolution.Unsupported, rule.Class)
			continue
		}

		if len(rule.Config) > 0 {
			if configurable, ok := resolved.(fixer.ConfigurableFixer); ok {
				resolved = configurable.WithConfig(rule.Config)
			} else {
				resolution.ConfigIgnored = append(resolution.ConfigIgnored, rule.Class)
			}
		}

		if !seen[rule.Class] {
			seen[rule.Class] = true
			resolution.Mapped++
			config.Rules = append(config.Rules, resolved)
		}
	}

	// apply fixers in the same canonical (priority) order as the standalone path,
	// since the ECS dump lists them in set order, not execution order
	rules.CanonicalOrder(config.Rules)

	sort.Strings(resolution.Unsupported)
	sort.Strings(resolution.ConfigIgnored)
	sort.Strings(resolution.PerPathSkips)

	return config, resolution, nil
}
