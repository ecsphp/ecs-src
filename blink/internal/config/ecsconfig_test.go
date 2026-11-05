package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// esc doubles backslashes so an FQCN can be embedded in a JSON string literal.
func esc(s string) string {
	return strings.ReplaceAll(s, "\\", "\\\\")
}

const (
	knownRuleA  = `PhpCsFixer\Fixer\Alias\NoMixedEchoPrintFixer`
	knownRuleB  = `PhpCsFixer\Fixer\Alias\NoAliasFunctionsFixer`
	unknownRule = `PhpCsFixer\Fixer\Nonexistent\FooBarFixer`
	// unconfigurableRule takes no options in blink
	unconfigurableRule = `PhpCsFixer\Fixer\Semicolon\NoSinglelineWhitespaceBeforeSemicolonsFixer`
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
		"rules": [{"class": "`+esc(unconfigurableRule)+`", "config": {"some": "option"}}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.ConfigIgnored) != 1 || resolution.ConfigIgnored[0] != unconfigurableRule {
		t.Fatalf("config-ignored = %v, want [%s]", resolution.ConfigIgnored, unconfigurableRule)
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

func TestLoadECSRejectsConflictingCheckers(t *testing.T) {
	const yoda = `PhpCsFixer\Fixer\ControlStructure\YodaStyleFixer`
	const noYoda = `SlevomatCodingStandard\Sniffs\ControlStructures\DisallowYodaComparisonSniff`

	// both halves of a conflicting group loaded -> rejected
	_, _, err := LoadECS(writeECS(t, `{
		"rules": [
			{"class": "`+esc(yoda)+`", "config": {}},
			{"class": "`+esc(noYoda)+`", "config": {}}
		]
	}`))
	if err == nil {
		t.Fatal("expected conflicting checkers to be rejected")
	}
	if !strings.Contains(err.Error(), "mutually exclude") {
		t.Fatalf("unexpected error: %v", err)
	}

	// skipping one half resolves the conflict
	if _, _, err := LoadECS(writeECS(t, `{
		"rules": [
			{"class": "`+esc(yoda)+`", "config": {}},
			{"class": "`+esc(noYoda)+`", "config": {}}
		],
		"skips": [{"class": "`+esc(noYoda)+`"}]
	}`)); err != nil {
		t.Fatalf("skipping one half should resolve the conflict, got %v", err)
	}

	// only one half loaded -> fine
	if _, _, err := LoadECS(writeECS(t, `{
		"rules": [{"class": "`+esc(yoda)+`", "config": {}}]
	}`)); err != nil {
		t.Fatalf("single checker should load, got %v", err)
	}
}

func TestLoadECSResolvesDeprecatedAlias(t *testing.T) {
	deprecated := `PhpCsFixer\Fixer\ControlStructure\NoUnneededCurlyBracesFixer`
	successor := `PhpCsFixer\Fixer\ControlStructure\NoUnneededBracesFixer`

	// the deprecated class name still maps, to the successor fixer
	c, resolution, err := LoadECS(writeECS(t, `{
		"rules": [{"class": "`+esc(deprecated)+`", "config": {}}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Mapped != 1 || len(resolution.Unsupported) != 0 {
		t.Fatalf("deprecated alias not mapped: mapped=%d unsupported=%v", resolution.Mapped, resolution.Unsupported)
	}
	if len(c.Rules) != 1 || c.Rules[0].Name() != successor {
		t.Fatalf("alias did not resolve to successor, got %v", c.Rules)
	}

	// the deprecated name and its successor collapse to a single fixer
	c, resolution, err = LoadECS(writeECS(t, `{
		"rules": [
			{"class": "`+esc(deprecated)+`", "config": {}},
			{"class": "`+esc(successor)+`", "config": {}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Mapped != 1 || len(c.Rules) != 1 {
		t.Fatalf("alias and successor should collapse to one rule, mapped=%d rules=%d", resolution.Mapped, len(c.Rules))
	}
}

func TestLoadECSCountsDistinctClasses(t *testing.T) {
	// ECS sets overlap, so the same class often appears several times. Duplicate
	// entries must not inflate Total, or a fully mapped config looks narrower
	// than it is (mapped 158 of 180 when the 22 extra are duplicates).
	c, resolution, err := LoadECS(writeECS(t, `{
		"rules": [
			{"class": "`+esc(knownRuleA)+`", "config": {}},
			{"class": "`+esc(knownRuleA)+`", "config": {}},
			{"class": "`+esc(knownRuleA)+`", "config": {}},
			{"class": "`+esc(knownRuleB)+`", "config": {}}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Total != 2 || resolution.Mapped != 2 {
		t.Fatalf("distinct classes miscounted: mapped/total = %d/%d, want 2/2", resolution.Mapped, resolution.Total)
	}
	if len(c.Rules) != 2 {
		t.Fatalf("expected 2 distinct rules, got %d", len(c.Rules))
	}
}

// A fixer whose options depend on key order gets the config as dumped.
func TestLoadECSKeepsConfigKeyOrder(t *testing.T) {
	for _, c := range []struct{ replacements, want string }{
		{`{"foo-bar": "qux", "foo": "baz"}`, "@qux"},
		{`{"foo": "baz", "foo-bar": "qux"}`, "@baz-bar"},
	} {
		cfg, _, err := LoadECS(writeECS(t, `{
			"rules": [{"class": "`+esc(`PhpCsFixer\Fixer\Phpdoc\GeneralPhpdocTagRenameFixer`)+`", "config": {"replacements": `+c.replacements+`}}]
		}`))
		if err != nil {
			t.Fatal(err)
		}
		s := tokens.New(lexer.Lex("<?php\n/** @foo-bar */\nfunction f() {}"))
		cfg.Rules[0].Fix(s)
		if want := "<?php\n/** " + c.want + " */\nfunction f() {}"; s.Render() != want {
			t.Fatalf("replacements %s: got %q, want %q", c.replacements, s.Render(), want)
		}
	}
}

// No path list skips the rule everywhere, a list on those paths, an empty list nowhere.
func TestLoadECSClassSkipPathLists(t *testing.T) {
	cases := []struct {
		name      string
		skip      string
		wantRules int
		wantPaths []string
	}{
		{"no paths", `{"class": "` + esc(knownRuleA) + `"}`, 0, nil},
		{"null paths", `{"class": "` + esc(knownRuleA) + `", "paths": null}`, 0, nil},
		{"some paths", `{"class": "` + esc(knownRuleA) + `", "paths": ["/p/old/*.php", "/p/src/Web"]}`, 1, []string{"/p/old/*.php", "/p/src/Web"}},
		{"empty paths", `{"class": "` + esc(knownRuleA) + `", "paths": []}`, 1, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, _, err := LoadECS(writeECS(t, `{
				"rules": [{"class": "`+esc(knownRuleA)+`", "config": {}}],
				"skips": [`+c.skip+`]
			}`))
			if err != nil {
				t.Fatal(err)
			}
			if len(cfg.Rules) != c.wantRules {
				t.Fatalf("rules = %d, want %d", len(cfg.Rules), c.wantRules)
			}
			groups, ok := cfg.PerPathRuleSkips[knownRuleA]
			if ok != (c.wantPaths != nil) {
				t.Fatalf("per-path skips = %v (present %v), want %v", groups, ok, c.wantPaths)
			}
			if !ok {
				return
			}
			if len(groups) != 1 || !slices.Equal(groups[0], c.wantPaths) {
				t.Fatalf("per-path skips = %v, want [%v]", groups, c.wantPaths)
			}
		})
	}
}

// ECS runs a deprecated alias and its successor as two checkers, so the shared
// fixer is skipped only where both are (checked with the PHP engine).
func TestLoadECSPerPathSkipOnDeprecatedAlias(t *testing.T) {
	deprecated := `PhpCsFixer\Fixer\ControlStructure\NoUnneededCurlyBracesFixer`
	successor := `PhpCsFixer\Fixer\ControlStructure\NoUnneededBracesFixer`
	rule := func(class string) string { return `{"class": "` + esc(class) + `", "config": {}}` }
	skip := func(class string, paths string) string {
		return `{"class": "` + esc(class) + `", "paths": ` + paths + `}`
	}

	cases := []struct {
		name  string
		rules []string
		skips []string
		want  [][]string // nil: the fixer has no per-path skip
	}{
		{"alias skipped", []string{rule(deprecated)}, []string{skip(deprecated, `["/p/a"]`)}, [][]string{{"/p/a"}}},
		{"successor configured too", []string{rule(deprecated), rule(successor)}, []string{skip(deprecated, `["/p/a"]`)}, nil},
		{"both skipped", []string{rule(deprecated), rule(successor)}, []string{skip(deprecated, `["/p/a"]`), skip(successor, `["/p/*/A.php"]`)}, [][]string{{"/p/a"}, {"/p/*/A.php"}}},
		{"skip on unconfigured successor", []string{rule(deprecated)}, []string{skip(successor, `["/p/a"]`)}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, _, err := LoadECS(writeECS(t, `{"rules": [`+strings.Join(c.rules, ",")+`], "skips": [`+strings.Join(c.skips, ",")+`]}`))
			if err != nil {
				t.Fatal(err)
			}
			if len(cfg.Rules) != 1 || cfg.Rules[0].Name() != successor {
				t.Fatalf("rules = %v, want only %s", cfg.Rules, successor)
			}
			got, ok := cfg.PerPathRuleSkips[successor]
			if ok != (c.want != nil) || !slices.EqualFunc(got, c.want, slices.Equal[[]string]) {
				t.Fatalf("per-path skips = %q (present %v), want %q", got, ok, c.want)
			}
			if len(cfg.PerPathRuleSkips) > 1 || (c.want == nil && len(cfg.PerPathRuleSkips) > 0) {
				t.Fatalf("per-path skips keyed off the fixer: %q", cfg.PerPathRuleSkips)
			}
		})
	}
}

// fixWith runs the rules an ECS config resolves to over code.
func fixWith(t *testing.T, rules []string, config, code string) string {
	t.Helper()
	entries := make([]string, 0, len(rules))
	for _, class := range rules {
		entries = append(entries, `{"class": "`+esc(class)+`", "config": `+config+`}`)
	}
	cfg, resolution, err := LoadECS(writeECS(t, `{"rules": [`+strings.Join(entries, ",")+`]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(resolution.Unsupported) != 0 || len(resolution.ConfigIgnored) != 0 {
		t.Fatalf("unsupported %v, config ignored %v", resolution.Unsupported, resolution.ConfigIgnored)
	}
	s := tokens.New(lexer.Lex(code))
	for _, rule := range cfg.Rules {
		rule.Fix(s)
		s.ClearEmptyTokens()
	}
	return s.Render()
}

// Each deprecated PHP-CS-Fixer class runs its successor the way the deprecated
// fixer's proxy configures it.
func TestLoadECSRunsDeprecatedAliasesAsTheirProxy(t *testing.T) {
	cases := []struct {
		class, config, in, want string
	}{
		{
			`PhpCsFixer\Fixer\ArrayNotation\NoTrailingCommaInSinglelineArrayFixer`, `{}`,
			"<?php\n$a = [1, 2,];\nfoo(1, 2,);\nlist($b, $c,) = $d;\n",
			"<?php\n$a = [1, 2];\nfoo(1, 2,);\nlist($b, $c,) = $d;\n",
		},
		{
			`PhpCsFixer\Fixer\ControlStructure\NoTrailingCommaInListCallFixer`, `{}`,
			"<?php\n$a = [1, 2,];\nfoo(1, 2,);\nlist($b, $c,) = $d;\n",
			"<?php\n$a = [1, 2,];\nfoo(1, 2,);\nlist($b, $c) = $d;\n",
		},
		{
			`PhpCsFixer\Fixer\FunctionNotation\NoTrailingCommaInSinglelineFunctionCallFixer`, `{}`,
			"<?php\n$a = [1, 2,];\nfoo(1, 2,);\nlist($b, $c,) = $d;\n",
			"<?php\n$a = [1, 2,];\nfoo(1, 2);\nlist($b, $c) = $d;\n",
		},
		{
			`PhpCsFixer\Fixer\FunctionNotation\FunctionTypehintSpaceFixer`, `{}`,
			"<?php\nclass A { public int  $p; }\nfunction f(int  $a) {}\n",
			"<?php\nclass A { public int  $p; }\nfunction f(int $a) {}\n",
		},
		{
			`PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAfterConstructFixer`, `{}`,
			"<?php\nforeach ($a  as  $b) {}\nreturn  1;\n",
			"<?php\nforeach ($a  as $b) {}\nreturn 1;\n",
		},
		{
			`PhpCsFixer\Fixer\LanguageConstruct\SingleSpaceAfterConstructFixer`, `{"constructs": ["echo"]}`,
			"<?php\necho  1;\nreturn  2;\n",
			"<?php\necho 1;\nreturn  2;\n",
		},
		{
			`PhpCsFixer\Fixer\Operator\NewWithBracesFixer`, `{}`,
			"<?php\n$a = new Foo;\n",
			"<?php\n$a = new Foo();\n",
		},
		{
			`PhpCsFixer\Fixer\Operator\NewWithBracesFixer`, `{"named_class": false}`,
			"<?php\n$a = new Foo;\n",
			"<?php\n$a = new Foo;\n",
		},
		{
			`PhpCsFixer\Fixer\Whitespace\CompactNullableTypehintFixer`, `{}`,
			"<?php\nfunction f(?  int $a) {}\n",
			"<?php\nfunction f(?int $a) {}\n",
		},
		{
			`PhpCsFixer\Fixer\Whitespace\NoSpacesInsideParenthesisFixer`, `{}`,
			"<?php\nfoo( 1, 2 );\n",
			"<?php\nfoo(1, 2);\n",
		},
		{
			`PhpCsFixer\Fixer\Basic\CurlyBracesPositionFixer`, `{}`,
			"<?php\nclass A {\n}\n",
			"<?php\nclass A\n{\n}\n",
		},
	}
	for _, c := range cases {
		if got := fixWith(t, []string{c.class}, c.config, c.in); got != c.want {
			t.Errorf("%s %s:\ngot  %q\nwant %q", c.class, c.config, got, c.want)
		}
	}
}

// Two deprecated classes proxying one successor with different options both
// keep theirs, rather than collapsing into whichever comes first.
func TestLoadECSKeepsEachDeprecatedAliasConfiguration(t *testing.T) {
	got := fixWith(t, []string{
		`PhpCsFixer\Fixer\ArrayNotation\NoTrailingCommaInSinglelineArrayFixer`,
		`PhpCsFixer\Fixer\ControlStructure\NoTrailingCommaInListCallFixer`,
	}, `{}`, "<?php\n$a = [1, 2,];\nfoo(1, 2,);\nlist($b, $c,) = $d;\n")
	if want := "<?php\n$a = [1, 2];\nfoo(1, 2,);\nlist($b, $c) = $d;\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
