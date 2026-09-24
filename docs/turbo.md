# Turbo mode (experimental)

`--turbo` hands the run over to the [`reco`](https://github.com/TomasVotruba/reco) Go binary instead of the PHP engine:

```bash
vendor/bin/ecs check --turbo          # report only (like a normal check)
vendor/bin/ecs check --turbo --fix    # rewrite files in place
vendor/bin/ecs check src --turbo      # limit to a path - pass the path before the flag
```

Pass any explicit path before the flags (`check src --turbo`), as with `--fix`; a path written after a boolean flag is consumed as that flag's value.

ECS resolves the paths from `ecs.php` (and any paths passed on the command line) exactly as usual, then invokes reco over them. The check/fix split maps onto reco directly:

- `--turbo` runs `reco run --dry-run <paths>` - reports without writing.
- `--turbo --fix` runs `reco run <paths>` - rewrites in place.

The exit code of `reco` is passed through.

## Binary resolution

The reco binary is looked up in this order:

1. the `ECS_TURBO_BIN` environment variable, if it points to an existing file;
2. `vendor/bin/reco`, if present;
3. `reco` on the `PATH`.

## Prototype caveat

This is an RFC-stage prototype. In turbo mode reco runs **its own native rule set**, not the sniffs and fixers configured in `ecs.php`. The rule selection in your config is ignored for now - only the paths are shared. Do not treat a turbo run as equivalent to a normal ECS run yet.

## Remaining delivery work

For `--turbo` to work out of the box, the reco binary has to be installed alongside ECS. The planned path:

- ship reco as downloadable per-OS binaries via goreleaser releases;
- add a Composer post-install step that downloads the matching binary into `vendor/bin/reco`.

Until then, build reco yourself and point `ECS_TURBO_BIN` at it, or put it on your `PATH`.
