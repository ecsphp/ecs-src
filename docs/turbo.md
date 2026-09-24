# Turbo mode (experimental)

`--turbo` hands the run over to the [`reco`](https://github.com/TomasVotruba/reco) Go binary instead of the PHP engine:

```bash
vendor/bin/ecs check --turbo          # report only (like a normal check)
vendor/bin/ecs check --turbo --fix    # rewrite files in place
vendor/bin/ecs check src --turbo      # limit to a path - pass the path before the flag
```

Pass any explicit path before the flags (`check src --turbo`), as with `--fix`; a path written after a boolean flag is consumed as that flag's value.

ECS resolves the configuration from `ecs.php` (paths, rules and skips) exactly as usual, dumps it to a temporary JSON file, and hands that file to reco via `--ecs-config`. reco maps each ECS rule to its native equivalent, applies the shared paths and skips, and reports every rule it could not map. The check/fix split maps onto reco directly:

- `--turbo` runs `reco run --ecs-config <config.json> --dry-run` - reports without writing.
- `--turbo --fix` runs `reco run --ecs-config <config.json>` - rewrites in place.

The exit code of `reco` is passed through.

## Dumping the config

The JSON reco consumes can also be inspected on its own:

```bash
vendor/bin/ecs dump-config        # print the resolved paths, rules and skips as JSON
```

The shape is:

```json
{
  "paths": ["/abs/src"],
  "rules": [
    { "class": "PhpCsFixer\\Fixer\\ArrayNotation\\ArraySyntaxFixer", "config": { "syntax": "short" } },
    { "class": "PhpCsFixer\\Fixer\\CastNotation\\LowercaseCastFixer", "config": {} }
  ],
  "skips": [
    { "path": "*/Legacy/*" },
    { "class": "PhpCsFixer\\Fixer\\Import\\OrderedImportsFixer" },
    { "class": "PhpCsFixer\\Fixer\\CastNotation\\LowercaseCastFixer", "paths": ["*/tests/*"] }
  ]
}
```

Configured fixer options are read off the fixer instance; sniff properties are not extracted yet.

## Binary resolution

The reco binary is looked up in this order:

1. the `ECS_TURBO_BIN` environment variable, if it points to an existing file;
2. `vendor/bin/reco`, if present;
3. `reco` on the `PATH`.

## Prototype caveat

This is an RFC-stage prototype. reco now reads the rules and skips from your `ecs.php` (not just the paths), but it only maps rules that have an exact native equivalent. Every unmapped rule is reported and skipped, and a rule configured in a way reco does not replicate is reported as unsupported rather than run under the wrong behaviour. So a turbo run covers a growing subset of your config, never silently more or less - but it is not yet equivalent to a full ECS run. The reco side lists what maps today.

## Remaining delivery work

For `--turbo` to work out of the box, the reco binary has to be installed alongside ECS. The planned path:

- ship reco as downloadable per-OS binaries via goreleaser releases;
- add a Composer post-install step that downloads the matching binary into `vendor/bin/reco`.

Until then, build reco yourself and point `ECS_TURBO_BIN` at it, or put it on your `PATH`.
