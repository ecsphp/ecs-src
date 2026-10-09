// Command blink is a token-based coding-standard checker/fixer for PHP,
// modeled on symplify/easy-coding-standard.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"blink/internal/config"
	"blink/internal/fixer/rules"
	"blink/internal/reporter"
	"blink/internal/runner"
)

func main() {
	// A short-lived batch process: relax the GC so it collects far less during
	// the run instead of reclaiming memory the process is about to release anyway.
	debug.SetGCPercent(400)
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fix := false
	configPath := ""
	ecsConfigPath := ""
	var paths []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--fix":
			fix = true
		case a == "list-checkers":
			return listCheckers()
		case a == "-h" || a == "--help":
			usage()
			return 0
		case a == "--config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		case len(a) > 9 && a[:9] == "--config=":
			configPath = a[9:]
		case a == "--ecs-config":
			if i+1 < len(args) {
				ecsConfigPath = args[i+1]
				i++
			}
		case len(a) > 13 && a[:13] == "--ecs-config=":
			ecsConfigPath = a[13:]
		default:
			paths = append(paths, a)
		}
	}

	var cfg *config.Config
	var err error
	if ecsConfigPath != "" {
		var resolution *config.ECSResolution
		cfg, resolution, err = config.LoadECS(ecsConfigPath)
		if err == nil {
			reportECSResolution(os.Stderr, resolution)
		}
	} else {
		cfg, err = loadConfig(configPath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}
	// CLI paths override the config; runs across all CPU cores by default
	cfg.WithPaths(paths...)

	start := time.Now()
	progress := reporter.NewProgress(os.Stderr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	results, err := runner.Run(ctx, cfg, fix, progress)
	progress.Finish()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}

	code := reporter.Report(os.Stdout, results, fix)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	reporter.Footer(os.Stdout, progress.Total(), time.Since(start), mem.Sys)
	return code
}

// loadConfig uses an explicit --config path, else a blink.json in the working
// directory, else the built-in defaults (all rules).
func loadConfig(configPath string) (*config.Config, error) {
	if configPath != "" {
		return config.Load(configPath)
	}
	if _, err := os.Stat("blink.json"); err == nil {
		return config.Load("blink.json")
	}
	return config.Configure(), nil
}

// reportECSResolution prints, to w, only the ECS rules blink could not fully
// honour: rules with no blink fixer, and rules whose configuration blink does
// not model. A fully mapped config prints nothing. Per-path skips are
// intentional and not reported.
func reportECSResolution(w io.Writer, resolution *config.ECSResolution) {
	if len(resolution.Unsupported) == 0 && len(resolution.ConfigIgnored) == 0 {
		return
	}
	if len(resolution.Unsupported) > 0 {
		_, _ = fmt.Fprintf(w, "blink: %d unsupported ECS rule(s) (no blink fixer), skipped:\n", len(resolution.Unsupported))
		for _, class := range resolution.Unsupported {
			_, _ = fmt.Fprintf(w, "    - %s\n", class)
		}
	}
	if len(resolution.ConfigIgnored) > 0 {
		_, _ = fmt.Fprintf(w, "blink: %d configured ECS rule(s) applied with blink's built-in behaviour (config not modelled):\n", len(resolution.ConfigIgnored))
		for _, class := range resolution.ConfigIgnored {
			_, _ = fmt.Fprintf(w, "    - %s\n", class)
		}
	}
}

func listCheckers() int {
	fmt.Println("Registered checkers:")
	for _, r := range rules.All() {
		fmt.Printf("  - %s\n", r.Name())
	}
	return 0
}

func usage() {
	fmt.Print(`blink - token-based PHP coding standard tool

Usage:
  blink [paths...]            check paths (default: .)
  blink --fix [paths...]      fix paths in place
  blink --config FILE ...     use a blink.json config
  blink --ecs-config FILE ... use an ECS dump-config JSON (blink mode)
  blink list-checkers         list registered fixers

Loads blink.json from the working directory when present.
Runs across all CPU cores by default.
`)
}
