package config

import (
	"os"
	"path/filepath"
	"testing"

	"blink/internal/fixer/rules"
	"blink/internal/set"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "blink.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadDefaultsToDefaultSet(t *testing.T) {
	c, err := Load(write(t, `{}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Rules) != len(set.Default()) {
		t.Fatalf("empty config should enable the default psr12+common set, got %d", len(c.Rules))
	}
	if len(c.Paths) != 1 || c.Paths[0] != "." {
		t.Fatalf("default path should be '.', got %v", c.Paths)
	}
}

func TestLoadSet(t *testing.T) {
	c, err := Load(write(t, `{"sets":["spaces"],"paths":["src"],"skip":["*/Fixture/*"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Rules) != len(rules.SpacingFixers()) {
		t.Fatalf("spaces set should enable the spacing rules, got %d", len(c.Rules))
	}
	if len(c.Paths) != 1 || c.Paths[0] != "src" {
		t.Fatalf("paths not read: %v", c.Paths)
	}
	if len(c.Skip) != 1 {
		t.Fatalf("skip not read: %v", c.Skip)
	}
}

func TestLoadLevelIsPrefix(t *testing.T) {
	c, err := Load(write(t, `{"level":{"spaces":3}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Rules) != 3 {
		t.Fatalf("level 3 should give 3 rules, got %d", len(c.Rules))
	}
	// resolve keeps canonical (All) order; the 3 chosen must be the first 3
	// spacing rules, appearing in All in the same relative order
	spacing := rules.SpacingFixers()
	wantNames := map[string]bool{spacing[0].Name(): true, spacing[1].Name(): true, spacing[2].Name(): true}
	for _, f := range c.Rules {
		if !wantNames[f.Name()] {
			t.Fatalf("level 3 selected unexpected rule %s", f.Name())
		}
	}
}

func TestLoadExplicitRule(t *testing.T) {
	name := rules.All()[0].Name()
	c, err := Load(write(t, `{"rules":["`+jsonEscape(name)+`"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Rules) != 1 || c.Rules[0].Name() != name {
		t.Fatalf("explicit rule not selected: %v", c.Rules)
	}
}

func TestLoadUnknownErrors(t *testing.T) {
	cases := []string{
		`{"sets":["nope"]}`,
		`{"level":{"psr12":1}}`,
		`{"rules":["Nope\\Fixer"]}`,
	}
	for _, body := range cases {
		if _, err := Load(write(t, body)); err == nil {
			t.Errorf("expected error for %s", body)
		}
	}
}

// jsonEscape escapes backslashes for embedding a PHP FQCN in a JSON literal.
func jsonEscape(s string) string {
	out := make([]rune, 0, len(s)*2)
	for _, r := range s {
		if r == '\\' {
			out = append(out, '\\', '\\')
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
