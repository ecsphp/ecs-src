# Changelog

All notable changes to **Symplify EasyCodingStandard (ECS)** are documented here.

This log covers the last 10 releases (`13.0.2` → `13.1.6`). PR references `#1`–`#15`
belong to this repository (`easy-coding-standard/ecs-src`); higher numbers (`#321`–`#329`)
reference the original `symplify/easy-coding-standard` repository from before the split.

## [13.1.6] - 2026-06-09

### Changed
- Bump dependencies and raise the minimum dev requirement to **PHP 8.4**; move to `entropy/entropy` v3 for DI/console wiring (#15)

## [13.1.5] - 2026-05-30

### Internal
- Add CI test covering the scoped PHPUnit namespace fix from #8 (#12)

## [13.1.4] - 2026-05-30

### Changed
- Improve the deprecation message for `strict: true`: it now tells you exactly where to remove
  `strict: true` from the `->withPreparedSets(strict: true, ...)` call in `ecs.php` (#11)
- Improve the deprecation message for the `STRICT` set (#7)

## [13.1.3] - 2026-05-04

This release completes the move to the standalone `easy-coding-standard/ecs-src` repository.

### Fixed
- Scoped build: stop prefixing the lower-case `PHPUnit\Framework` namespace on the scoped build (#5)
- Scoped build: also strip the scoper prefix from `PHPUnit\Runner` and `PHPUnit\Util` (#6)

### Removed
- Remove the `scripts` command — out of scope for ECS (#1)

## [13.1.2] - 2026-05-04

### Internal
- Drop a no-longer-needed entry from the linter build

## [13.1.1] - 2026-05-03

### Internal
- Build/lock cleanup

## [13.1.0] - 2026-05-03

The main feature release of this window: gradual, step-by-step adoption levels and clearer
guidance away from dangerous sets.

### Added
- `withSpacesLevel(int $level)` — enable spacing rules incrementally, safest first, instead of
  flipping every spacing rule on at once. Level `0` turns on a single, safe rule; raise the level
  one PR at a time to grow coverage (#326)

  ```php
  // ecs.php
  use Symplify\EasyCodingStandard\Config\ECSConfig;

  return ECSConfig::configure()
      ->withSpacesLevel(5);
  ```

- `withArraysLevel()`, `withControlStructuresLevel()`, and `withDocblockLevel()` — the same
  gradual, leveled adoption for the array, control-structure, and docblock rule sets. Each starts
  with pure cleanup rules and ramps up to more invasive layout changes (#327)

  ```php
  // ecs.php
  use Symplify\EasyCodingStandard\Config\ECSConfig;

  return ECSConfig::configure()
      ->withArraysLevel(3)
      ->withControlStructuresLevel(2)
      ->withDocblockLevel(1);
  ```

  Example from the array set (`tests/Set/Array/Fixture/nested_array.php.inc`):

  ```diff
  -$test = ['key' => ['keyA' => 'valueA']];
  +$test = [
  +    'key' => [
  +        'keyA' => 'valueA',
  +    ],
  +];
  ```

### Changed
- Reconfigure `OrderedImportsFixer` to sort imports by `class`, `function`, then `const`,
  avoiding a mixed import order (#321)

  ```php
  // config/set/clean-code.php
  use PhpCsFixer\Fixer\Import\OrderedImportsFixer;
  use Symplify\EasyCodingStandard\Config\ECSConfig;

  return ECSConfig::configure()
      ->withConfiguredRule(OrderedImportsFixer::class, [
          'imports_order' => ['class', 'function', 'const'],
      ]);
  ```

- Bump dependencies (#328)

### Deprecated
- The `STRICT` set is now deprecated — it is dangerous to apply without context. Use Rector
  instead for safe, context-aware changes (#329)
- The `PHPUNIT` set is now deprecated — it never worked as intended (#329)

  ```diff
   // ecs.php — remove the deprecated flags
   return ECSConfig::configure()
       ->withPreparedSets(
  -        strict: true,
  -        phpunit: true,
           psr12: true,
       );
  ```

## [13.0.4] - 2026-01-05

### Internal
- Update tool versions for the bare run

## [13.0.3] - 2026-01-05

### Fixed
- Lock `nette/utils` to avoid a breaking transitive update

## [13.0.2] - 2026-01-05

### Fixed
- Lock `nette/utils` to avoid a breaking transitive update

[13.1.6]: https://github.com/easy-coding-standard/ecs-src/compare/13.1.5...13.1.6
[13.1.5]: https://github.com/easy-coding-standard/ecs-src/compare/13.1.4...13.1.5
[13.1.4]: https://github.com/easy-coding-standard/ecs-src/compare/13.1.3...13.1.4
[13.1.3]: https://github.com/easy-coding-standard/ecs-src/compare/13.1.2...13.1.3
[13.1.2]: https://github.com/easy-coding-standard/ecs-src/compare/13.1.1...13.1.2
[13.1.1]: https://github.com/easy-coding-standard/ecs-src/compare/13.1.0...13.1.1
[13.1.0]: https://github.com/easy-coding-standard/ecs-src/compare/13.0.4...13.1.0
[13.0.4]: https://github.com/easy-coding-standard/ecs-src/compare/13.0.3...13.0.4
[13.0.3]: https://github.com/easy-coding-standard/ecs-src/compare/13.0.2...13.0.3
[13.0.2]: https://github.com/easy-coding-standard/ecs-src/compare/13.0.1...13.0.2
