# blink

Fast, token-based PHP coding-standard checker and fixer - an [ECS](https://github.com/symplify/easy-coding-standard)-style
tool written in Go. Runs across all CPU cores by default. Lives under `blink/`
in the ECS repository and mirrors ECS's `psr12` + `common` output.

## Rule coverage

Share of the fixer universe blink implements - every PHP-CS-Fixer and Symplify
fixer, every PHP_CodeSniffer sniff (fixable or report-only), plus the sniffs ECS references from
its sets. Regenerate with `bin/rule-counts.sh`; CI keeps it current and fails
while any PHP-CS-Fixer fixer or PHP_CodeSniffer sniff is missing. After bumping
PHP_CodeSniffer, refresh the sniff list with `bin/phpcs-sniffs.sh`.

Every sniff blink implements must pass PHP_CodeSniffer's own test files
(`internal/fixer/rules/testdata/phpcs`): the upstream `.inc` inputs with the
output ECS produces for each. ECS runs sniffs with `phpcs:` annotations off, so
those expectations can differ from upstream's `.inc.fixed`. Regenerate them with
`bin/phpcs-fixtures.sh <PHP_CodeSniffer checkout>`.

<!-- rule-counts:start -->
| source | in blink | universe | coverage |
|---|---:|---:|---:|
| PHP-CS-Fixer | 282 | 282 | 100% |
| Symplify | 34 | 34 | 100% |
| PHP_CodeSniffer | 203 | 203 | 100% |
| **total** | **519** | **519** | **100%** |

blink runs 630 rules in all, its own extras included.
<!-- rule-counts:end -->

### slevomat sniffs

blink also ports the fixable [slevomat/coding-standard](https://github.com/slevomat/coding-standard)
sniffs, one category at a time. They are extras outside the universe above,
measured against every fixable slevomat sniff in `slevomat-fixable-sniffs.txt`
(refresh with `bin/slevomat-fixable-sniffs.sh` after a slevomat release).

<!-- slevomat-counts:start -->
| slevomat category | in blink | fixable sniffs | coverage |
|---|---:|---:|---:|
| Arrays | 5 | 5 | 100% |
| Attributes | 5 | 5 | 100% |
| Classes | 0 | 27 | 0% |
| Commenting | 13 | 13 | 100% |
| ControlStructures | 21 | 21 | 100% |
| Exceptions | 3 | 3 | 100% |
| Functions | 14 | 14 | 100% |
| Namespaces | 13 | 13 | 100% |
| Numbers | 1 | 1 | 100% |
| Operators | 4 | 4 | 100% |
| PHP | 9 | 9 | 100% |
| TypeHints | 15 | 15 | 100% |
| Variables | 1 | 1 | 100% |
| Whitespaces | 1 | 1 | 100% |
| **total** | **105** | **132** | **79%** |
<!-- slevomat-counts:end -->

## Build

Requires Go (see `go.mod` for the version).

```bash
cd blink
make build   # produces ./blink
```

## Usage

Check your code (reports a diff of what would change and the errors to fix by
hand, exit code 1 if issues):

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
  `per-cs` is exactly ECS's `SetList::PER_CS` (PHP-CS-Fixer's `@PER-CS`): the rules
  of `config/set/per-cs.php`, each with that file's configuration.
- `level` - gradual adoption: `{"spaces": N}` enables the first N rules of the
  spaces set (safest first), so you can raise coverage one step at a time.
- `rules` - enable individual fixers by name.
- `paths` / `skip` - files to scan and paths to ignore. Skips work like ECS's
  `withSkip()` paths: an entry without `*` must exist, a directory skips
  everything below it, and `*` also crosses `/`.

With no config file (or none of `sets`, `level` and `rules`), the default
`psr12` + `common` sets run. CLI path arguments override `paths`.

### Consuming an ECS config

`blink --ecs-config <file>` reads the JSON that ECS's `ecs dump-config` produces
(its resolved paths, rules and skips) and runs blink over it. Because every
blink fixer is named by its PHP-CS-Fixer class, the ECS rules map straight onto
blink fixers by class name. Any rule blink has no fixer for is reported and
skipped, so the run is never silently narrower than the ECS config. Rule
configuration is not modelled yet - a configured rule runs with blink's built-in
behaviour, which is noted in the report.

A sniff that only reports (such as GlobalKeywordSniff) prints ECS's table of
errors to fix by hand and fails the run, as ECS does.

## Parity with ECS

The `blink parity` CI workflow runs blink and the ECS in this repository over the
same real codebase (full rector-src, `psr12` + `common`) and fails if blink's
output differs from ECS on any file. It is the 1:1 parity gate: it stays red
until blink reproduces ECS exactly, and its job summary ranks the missing fixers
by how often ECS applies them on the sample.

## PSR-12

The `psr12` set implements the token-safe part of PHP-CS-Fixer's `@PSR-12`:
casing (keywords, constants, static references, casts), binary operator
spacing, parenthesis and language-construct spacing, `else if` -> `elseif`,
`declare` normalization, leading import slash removal, one import per
statement, blank lines before a namespace, no blank lines after a class
opening, and tab-to-space indentation.

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
