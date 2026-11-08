# Benchmark

Wall-clock comparison of the two ECS engines - the PHP engine and the Go engine
(`--blink`) - against the native tools they wrap: php-cs-fixer, PHP_CodeSniffer
(`phpcbf`) and Laravel Pint.

```bash
bench/run.sh                                   # symfony src (default)
bench/run.sh blink/parity/corpora/laravel.json src
```

The corpus files live in `blink/parity/corpora/`. Each engine runs in `--fix`
mode over a fresh copy of the same files. ECS, `--blink` and `phpcbf` run across
all cores; php-cs-fixer and Pint run sequentially (as shipped). Numbers are
wall-clock seconds and vary with hardware and rule set.

Requires `go`, `jq` and `composer` on `PATH`. The native tools are installed
into a temporary directory, since ECS scopes its own vendored php-cs-fixer and
it cannot run standalone from the repo `vendor/`.

A reported time is only meaningful for a run that completed, so the benchmark
aborts (rather than recording a partial time) if any engine crashes - a signal,
exit 255, or a PHP/Symfony fatal-exception signature in its output.

> Laravel's `tests` dir triggers a pathological php-cs-fixer fix loop, so the
> laravel run is limited to `src`. php-cs-fixer and Pint also abort on
> `symfony/symfony` `src`: it carries a `@php-cs-fixer-ignore` annotation for a
> rule their ruleset does not enable, which their shared Runner treats as fatal.
