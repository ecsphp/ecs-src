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

// Blink mode: ECS dumps its resolved ecs.php configuration to JSON (paths, rules,
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

// ECSResolution reports how an ECS config mapped onto blink, so a blink run is
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

// conflictingCheckerGroups are checker pairs that do the opposite of each other
// (e.g. Yoda vs no-Yoda). A config that loads both is contradictory, so blink
// rejects it, mirroring ECS's ConflictingCheckersCompilerPass. Classes are named
// exactly as ECS reports them in the dump-config "class" field.
var conflictingCheckerGroups = [][]string{
	{`Symplify\CodingStandard\Fixer\Spacing\StandaloneLineConstructorParamFixer`, `Symplify\CodingStandard\Fixer\Spacing\StandaloneLinePromotedPropertyFixer`},
	{`SlevomatCodingStandard\Sniffs\ControlStructures\DisallowYodaComparisonSniff`, `PhpCsFixer\Fixer\ControlStructure\YodaStyleFixer`},
	{`PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\LowerCaseConstantSniff`, `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\UpperCaseConstantSniff`},
	{`PhpCsFixer\Fixer\Casing\ConstantCaseFixer`, `PHP_CodeSniffer\Standards\Generic\Sniffs\PHP\UpperCaseConstantSniff`},
	{`SlevomatCodingStandard\Sniffs\TypeHints\DeclareStrictTypesSniff`, `PhpCsFixer\Fixer\LanguageConstruct\DeclareEqualNormalizeFixer`},
	{`SlevomatCodingStandard\Sniffs\TypeHints\DeclareStrictTypesSniff`, `PhpCsFixer\Fixer\PhpTag\BlankLineAfterOpeningTagFixer`},
	{`PHP_CodeSniffer\Standards\PSR12\Sniffs\Files\FileHeaderSniff`, `PhpCsFixer\Fixer\Phpdoc\NoBlankLinesAfterPhpdocFixer`},
	{`PHP_CodeSniffer\Standards\PSR2\Sniffs\Files\EndFileNewlineSniff`, `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\EndFileNoNewlineSniff`},
	{`PHP_CodeSniffer\Standards\Generic\Sniffs\Files\EndFileNewlineSniff`, `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\EndFileNoNewlineSniff`},
	{`PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\DisallowTabIndentSniff`, `PHP_CodeSniffer\Standards\Generic\Sniffs\WhiteSpace\DisallowSpaceIndentSniff`},
}

// checkConflictingCheckers rejects a config that loads both halves of any
// conflicting group, mirroring ECS's ConflictingCheckersCompilerPass. A class
// disabled via skip is not loaded and so does not count.
func checkConflictingCheckers(f ecsFile, skippedClasses map[string]bool) error {
	active := map[string]bool{}
	for _, rule := range f.Rules {
		if !skippedClasses[rule.Class] {
			active[rule.Class] = true
		}
	}
	for _, group := range conflictingCheckerGroups {
		both := true
		for _, class := range group {
			if !active[class] {
				both = false
				break
			}
		}
		if both {
			return fmt.Errorf(
				`checkers "%s" and "%s" mutually exclude each other; use only one of them or exclude the unwanted one with ->withSkip(...)`,
				group[0], group[1],
			)
		}
	}
	return nil
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

	if err := checkConflictingCheckers(f, skippedClasses); err != nil {
		return nil, nil, err
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

		// key on the resolved fixer name, so a deprecated class and its
		// successor (both resolving to one fixer) are not added twice
		if !seen[resolved.Name()] {
			seen[resolved.Name()] = true
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
