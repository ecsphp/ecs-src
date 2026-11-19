# Easy Coding Standard - Development Repository

Source for [Symplify EasyCodingStandard (ECS)](https://github.com/easy-coding-standard/ecs). Code here is scoped, downgraded and deployed to [easy-coding-standard/ecs](https://github.com/easy-coding-standard/ecs), the package `composer require symplify/easy-coding-standard` installs.

- **Using ECS?** See the [user documentation](build/target-repository/README.md).
- **Contributing?** Read on.

Requires PHP `>=8.4`.

```bash
composer install

composer test         # PHPUnit tests
composer lint         # phpstan + rector dry-run + check-cs
composer lint.fix     # auto-fix rector + coding standard
```

Please make sure `composer lint` and `composer test` pass before sending a pull request.

[`--blink` mode](docs/blink.md) runs the blink Go binary instead of the PHP engine - downloaded on first run, no Go toolchain needed:

| `--fix` on a fresh clone | Symfony `src` (13,103 files) | Laravel `src` (1,749 files) |
| --- | ---: | ---: |
| **ECS** | **2.9 s** | **1.0 s** |
| php-cs-fixer (`@PER-CS`, `@Symfony`) | 141.6 s | 25.8 s |
| PHP_CodeSniffer (`phpcbf`, PSR-12) | 133.7 s | 26.7 s |

If every rule in `ecs.php` is available in Go, a plain `vendor/bin/ecs check` runs blink instead of the PHP engine automatically - no `--blink` needed. Otherwise it falls back to PHP. Use `--no-blink` to force the PHP engine, e.g. to compare the speed.

Rules blink runs natively, per source. Regenerate with `blink/bin/rule-counts.sh`; CI fails when the table is stale.

<!-- parity-counts:start -->
| source | in blink | rules | coverage |
|---|---:|---:|---:|
| PHP-CS-Fixer | 282 | 282 | 100% |
| Symplify | 34 | 34 | 100% |
| PHP_CodeSniffer | 38 | 114 | 33% |
| slevomat | 19 | 132 | 14% |
| **total** | **373** | **562** | **66%** |
<!-- parity-counts:end -->
