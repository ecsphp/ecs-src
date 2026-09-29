package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// esc doubles backslashes so an FQCN can be embedded in a JSON string literal.
func esc(s string) string {
	return strings.ReplaceAll(s, "\\", "\\\\")
}

const (
	knownRuleA  = `PhpCsFixer\Fixer\Alias\NoMixedEchoPrintFixer`
	knownRuleB  = `PhpCsFixer\Fixer\Alias\NoAliasFunctionsFixer`
	unknownRule = `PhpCsFixer\Fixer\Nonexistent\FooBarFixer`
)

func writeECS(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ecs-config.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadECSMapsRulesByClassName(t *testing.T) {
	c, resolution, err := LoadECS(writeECS(t, `{
		"paths": ["src"],
		"rules": [
			{"class": "`+esc(knownRuleA)+`", "config": {}},
			{"class": "`+esc(knownRuleB)+`", "config": {}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(c.Rules) != 2 {
		t.Fatalf("expected 2 mapped rules, got %d", len(c.Rules))
	}
	if resolution.Mapped != 2 || resolution.Total != 2 {
		t.Fatalf("resolution mapped/total = %d/%d, want 2/2", resolution.Mapped, resolution.Total)
	}
	if len(c.Paths) != 1 || c.Paths[0] != "src" {
		t.Fatalf("paths not read: %v", c.Paths)
	}
}

func TestLoadECSReportsUnknownRule(t *testing.T) {
	c, resolution, err := LoadECS(writeECS(t, `{
		"rules": [
			{"class": "`+esc(knownRuleA)+`", "config": {}},
			{"class": "`+esc(unknownRule)+`", "config": {}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	if len(c.Rules) != 1 {
		t.Fatalf("expected 1 mapped rule, got %d", len(c.Rules))
	}
	if len(resolution.Unsupported) != 1 || resolution.Unsupported[0] != unknownRule {
		t.Fatalf("unsupported = %v, want [%s]", resolution.Unsupported, unknownRule)
	}
}

func TestLoadECSDefaultsPathToDot(t *testing.T) {
	c, _, err := LoadECS(writeECS(t, `{"rules": []}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Paths) != 1 || c.Paths[0] != "." {
		t.Fatalf("default path should be '.', got %v", c.Paths)
	}
}

func TestLoadECSSkips(t *testing.T) {
	c, resolution, err := LoadECS(writeECS(t, `{
		"rules": [{"class": "`+esc(knownRuleA)+`", "config": {}}],
		"skips": [
			{"path": "*/Legacy/*"},
			{"paths": ["*/Generated/*"]},
			{"class": "`+esc(knownRuleA)+`"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	// the whole-class skip drops the only rule
	if len(c.Rules) != 0 {
		t.Fatalf("expected the class skip to drop the rule, got %d rules", len(c.Rules))
	}
	if len(c.Skip) != 2 {
		t.Fatalf("expected 2 skip globs, got %v", c.Skip)
	}
	if resolution.Mapped != 0 {
		t.Fatalf("mapped = %d, want 0 (rule was skipped)", resolution.Mapped)
	}
}

func TestLoadECSPerPathSkipIsCaveat(t *testing.T) {
	c, resolution, err := LoadECS(writeECS(t, `{
		"rules": [{"class": "`+esc(knownRuleA)+`", "config": {}}],
		"skips": [
			{"class": "`+esc(knownRuleA)+`", "paths": ["*/tests/*"]}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	// per-path skip does not drop the rule; it is reported as a caveat instead
	if len(c.Rules) != 1 {
		t.Fatalf("expected the rule to still run, got %d rules", len(c.Rules))
	}
	if len(resolution.PerPathSkips) != 1 || resolution.PerPathSkips[0] != knownRuleA {
		t.Fatalf("per-path skips = %v, want [%s]", resolution.PerPathSkips, knownRuleA)
	}
}

func TestLoadECSConfigIgnoredNote(t *testing.T) {
	_, resolution, err := LoadECS(writeECS(t, `{
		"rules": [{"class": "`+esc(knownRuleA)+`", "config": {"some": "option"}}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.ConfigIgnored) != 1 || resolution.ConfigIgnored[0] != knownRuleA {
		t.Fatalf("config-ignored = %v, want [%s]", resolution.ConfigIgnored, knownRuleA)
	}
}

func TestLoadECSDeduplicatesRules(t *testing.T) {
	c, resolution, err := LoadECS(writeECS(t, `{
		"rules": [
			{"class": "`+esc(knownRuleA)+`", "config": {}},
			{"class": "`+esc(knownRuleA)+`", "config": {}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Rules) != 1 || resolution.Mapped != 1 {
		t.Fatalf("expected dedup to 1 rule, got %d rules / mapped %d", len(c.Rules), resolution.Mapped)
	}
}

func TestLoadECSErrors(t *testing.T) {
	if _, _, err := LoadECS(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("want error for missing file")
	}
	if _, _, err := LoadECS(writeECS(t, `{not json`)); err == nil {
		t.Error("want error for invalid json")
	}
}
