# blink

Fast, token-based PHP coding-standard checker and fixer - an [ECS](https://github.com/symplify/easy-coding-standard)-style
tool written in Go. Runs across all CPU cores by default. Lives under `blink/`
in the ECS repository and mirrors ECS's `psr12` + `common` output.

## Rule coverage

How many ECS rules blink implements. Regenerate with `bin/rule-counts.sh`;
CI keeps it current.

<!-- rule-counts:start -->
| tool | rules | of ECS |
|---|---:|---:|
| [ECS](https://github.com/symplify/easy-coding-standard) (baseline) | 209 | 100% |
| blink | 210 | 100% |
<!-- rule-counts:end -->

## Build

Requires Go (see `go.mod` for the version).

```bash
cd blink
make build   # produces ./blink
```

## Usage

Check your code (reports a diff of what would change, exit code 1 if issues):

```bash
./blink src tests
```

Fix in place:

```bash
./blink --fix src tests
```

List the active fixers:

```bash
./blink list-checkers
```

## Configuration

Drop a `blink.json` in your project root (auto-loaded, or point at one with
`--config`):

```json
{
    "paths": ["src", "tests"],
    "skip": ["*/Fixture/*"],
    "sets": ["spaces"],
    "level": {"spaces": 6}
}
```

- `sets` - enable a prepared set: `spaces`, `casing`, `psr12`, `per-cs`, `common`.
  `per-cs` mirrors PHP-CS-Fixer's `@PER-CS` (as ECS's `SetList::PER_CS`) and adds
  `single_line_empty_body` on top of the full set.
- `level` - gradual adoption: `{"spaces": N}` enables the first N rules of the
  spaces set (safest first), so you can raise coverage one step at a time.
- `rules` - enable individual fixers by name.
- `paths` / `skip` - files to scan and glob patterns to ignore.

With no config file, every fixer runs. CLI path arguments override `paths`.

### Consuming an ECS config

`blink --ecs-config <file>` reads the JSON that ECS's `ecs dump-config` produces
(its resolved paths, rules and skips) and runs blink over it. Because every
blink fixer is named by its PHP-CS-Fixer class, the ECS rules map straight onto
blink fixers by class name. Any rule blink has no fixer for is reported and
skipped, so the run is never silently narrower than the ECS config. Rule
configuration is not modelled yet - a configured rule runs with blink's built-in
behaviour, which is noted in the report.

## Parity with ECS

The `blink parity` CI workflow runs blink and the ECS in this repository over the
same real codebase (full rector-src, `psr12` + `common`) and fails if blink's
output differs from ECS on any file. It is the 1:1 parity gate: it stays red
until blink reproduces ECS exactly, and its job summary ranks the missing fixers
by how often ECS applies them on the sample.

## PSR-12

The `psr12` set implements the token-safe part of PHP-CS-Fixer's `@PSR-12`:
casing (keywords, constants, static references, casts), operator spacing
(assignment, arrow, comparison and logical operators), parenthesis and
language-construct spacing, `else if` -> `elseif`, `declare` normalization,
leading import slash removal, one import per statement, blank lines before a
namespace, no blank lines after a class opening, and tab-to-space indentation.

## What it looks like

```
1) src/Foo.php

    ---------- begin diff ----------
@@ Line 1 @@
 <?php
-    namespace App;
+namespace App;
-    $count=1;$total=2;
+    $count=1; $total=2;
    ----------- end diff -----------

Applied checkers:

 * PhpCsFixer\Fixer\NamespaceNotation\NoLeadingNamespaceWhitespaceFixer
 * PhpCsFixer\Fixer\Semicolon\SpaceAfterSemicolonFixer

 [WARNING] 1 error is fixable! Just add "--fix" to console command and rerun to apply.
```

## License

MIT
