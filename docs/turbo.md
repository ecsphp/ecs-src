# Turbo mode (experimental)

`--turbo` hands the run over to the [`ecs-go`](https://github.com/TomasVotruba/ecs-go) Go binary instead of the PHP engine:

```bash
vendor/bin/ecs check --turbo          # report only (like a normal check)
vendor/bin/ecs check --turbo --fix    # rewrite files in place
vendor/bin/ecs check src --turbo      # limit to a path - pass the path before the flag
```

Pass any explicit path before the flags (`check src --turbo`), as with `--fix`; a path written after a boolean flag is consumed as that flag's value.

ECS resolves the configuration from `ecs.php` (paths, rules and skips) exactly as usual, dumps it to a temporary JSON file, and hands that file to ecs-go via `--ecs-config`. ecs-go maps each ECS rule onto its own fixer by class name, applies the shared paths and skips, and reports every rule it could not map. The check/fix split maps onto ecs-go directly:

- `--turbo` runs `ecs-go --ecs-config <config.json>` - reports without writing.
- `--turbo --fix` runs `ecs-go --ecs-config <config.json> --fix` - rewrites in place.

The exit code of `ecs-go` is passed through.

## Dumping the config

The JSON ecs-go consumes can also be inspected on its own:

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

The ecs-go binary is looked up in this order:

1. the `ECS_TURBO_BIN` environment variable, if it points to an existing file;
2. `vendor/bin/ecs-go`, if present;
3. `ecs-go` on the `PATH`.

## Prototype caveat

This is an RFC-stage prototype. ecs-go reads the rules and skips from your `ecs.php` (not just the paths), and because every ecs-go fixer is named by its PHP-CS-Fixer class, the rules map straight across by name. Every rule ecs-go has no fixer for is reported and skipped, and a configured rule runs with ecs-go's built-in behaviour (its configuration is not modelled yet) and is noted in the report. So a turbo run is never silently narrower than your config, but it is not yet equivalent to a full ECS run.

## Remaining delivery work

For `--turbo` to work out of the box, the ecs-go binary has to be installed alongside ECS. The planned path:

- require `tomasvotruba/ecs-go` as a dev dependency, which exposes `vendor/bin/ecs-go`;
- or ship ecs-go as downloadable per-OS binaries and point `ECS_TURBO_BIN` at one.

Until then, build ecs-go yourself and point `ECS_TURBO_BIN` at it, or put it on your `PATH`.
