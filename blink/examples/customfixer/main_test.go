package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ecsphp/ecs-src/blink/extend"
)

func TestRenameFunction(t *testing.T) {
	t.Parallel()
	fixer := RenameFunction{}.WithConfig(map[string]any{"functions": map[string]any{"sizeof": "count"}})
	s := extend.Lex(`<?php
$a = sizeof($b) + \SIZEOF($c);
$d = $e->sizeof($f) + Foo::sizeof($g) + Foo\sizeof($h) + $i & sizeof($j);
function sizeof($k) {}
function &sizeof($l) {}
#[sizeof(1)]
class M {}
`)
	if !fixer.Fix(s) {
		t.Fatal("Fix reported no change")
	}
	want := `<?php
$a = count($b) + \count($c);
$d = $e->sizeof($f) + Foo::sizeof($g) + Foo\sizeof($h) + $i & count($j);
function sizeof($k) {}
function &sizeof($l) {}
#[sizeof(1)]
class M {}
`
	if got := s.Render(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TestOutOfTreeBuild builds this example as its own module, as a project would.
func TestOutOfTreeBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a blink binary")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command not found")
	}
	blinkDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	blinkGoMod, err := os.ReadFile(filepath.Join(blinkDir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	goVersion := regexp.MustCompile(`(?m)^go \S+$`).Find(blinkGoMod)

	module := t.TempDir()
	goMod := "module example.com/acme-blink\n\n" + string(goVersion) + "\n\n" +
		"require github.com/ecsphp/ecs-src/blink v0.0.0\n\n" +
		"replace github.com/ecsphp/ecs-src/blink => " + strconv.Quote(filepath.ToSlash(blinkDir)) + "\n"
	writeFile(t, filepath.Join(module, "go.mod"), goMod)
	writeFile(t, filepath.Join(module, "main.go"), string(source))

	binary := filepath.Join(t.TempDir(), "acme-blink")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command(goBin, "build", "-o", binary, ".")
	build.Dir = module
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	out, err := exec.Command(binary, "list-checkers").Output()
	if err != nil {
		t.Fatalf("list-checkers: %v", err)
	}
	name := RenameFunction{}.Name()
	if !strings.Contains(string(out), name) {
		t.Errorf("list-checkers does not list %s:\n%s", name, out)
	}

	project := t.TempDir()
	php := filepath.Join(project, "a.php")
	writeFile(t, php, "<?php\n\n$a = sizeof($b);\n")
	config, err := json.Marshal(map[string]any{
		"paths": []string{project},
		"rules": []map[string]any{
			{"class": name, "config": map[string]any{"functions": map[string]any{"sizeof": "count"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(t.TempDir(), "ecs-config.json")
	writeFile(t, configPath, string(config))

	out, err = exec.Command(binary, "--ecs-config", configPath, "--fix", project).CombinedOutput()
	if err != nil {
		t.Fatalf("--fix: %v\n%s", err, out)
	}
	fixed, err := os.ReadFile(php)
	if err != nil {
		t.Fatal(err)
	}
	if want := "<?php\n\n$a = count($b);\n"; string(fixed) != want {
		t.Errorf("fixed file:\n%s\nwant:\n%s", fixed, want)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
