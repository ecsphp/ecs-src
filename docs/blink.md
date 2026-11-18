# Blink mode

`--blink` runs your `ecs.php` rules with the bundled Go binary instead of the PHP engine:

```bash
vendor/bin/ecs check --blink          # report
vendor/bin/ecs check --blink --fix    # rewrite in place
vendor/bin/ecs check --blink --commit # one git commit per rule that changed files
vendor/bin/ecs check src --blink      # limit to a path - pass it before the flags
```

`--no-diffs` and `--output-format=json` work as usual. Rules blink can't map are reported and skipped.

A plain `vendor/bin/ecs check` uses blink automatically when every rule in `ecs.php` is available in Go, and falls back to the PHP engine otherwise. Pass `--no-blink` to force the PHP engine anyway, e.g. to compare the speed.

`--commit` needs a clean working tree. Each commit is named `[Group] RuleName`.

## Inspect the config

ECS passes the resolved paths, indentation, rules and skips to blink as JSON. Print it with:

```bash
vendor/bin/ecs dump-config
```

## Binary

The binary ships in `bin/` as `blink-<platform>` for Linux, macOS and Windows, so no Go toolchain is needed. Lookup order:

1. `ECS_BLINK_BIN` environment variable
2. bundled `bin/blink-<platform>`
3. `vendor/bin/ecs-go`
4. `ecs-go` on the `PATH`

## Custom fixers

blink can't run PHP fixers. Port yours to Go and build your own binary with the `github.com/ecsphp/ecs-src/blink/extend` package:

```go
package main

import "github.com/ecsphp/ecs-src/blink/extend"

func main() {
	extend.Register(MyFixer{}, 0) // the PHP fixer's getPriority()
	extend.Main()
}
```

- `Name()` returns the PHP fixer's class name; keep the rule in `ecs.php`.
- `WithConfig` gets only options that differ from the PHP defaults.

Point ECS at it with `ECS_BLINK_BIN`. See [`blink/examples/customfixer`](../blink/examples/customfixer/main.go).
