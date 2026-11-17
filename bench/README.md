# Benchmark

Wall-clock comparison of the two ECS engines - the PHP engine and the Go engine
(`--blink`) - against the native tools they wrap: php-cs-fixer and
PHP_CodeSniffer (`phpcbf`).

```bash
bench/run.sh                                   # symfony src (default)
bench/run.sh blink/parity/corpora/laravel.json src
```

The corpus files live in `blink/parity/corpora/`. Each engine runs in `--fix`
mode over a fresh copy of the same files. ECS, `--blink` and `phpcbf` run across
all cores; php-cs-fixer runs sequentially (as shipped). Numbers are
wall-clock seconds and vary with hardware and rule set.

Each run also writes `bench/results/<corpus>.json` with the corpus revision,
file count, ECS revision, date, core count and per-engine seconds.

Requires `go`, `jq` and `composer` on `PATH`. The native tools are installed
into a temporary directory, since ECS scopes its own vendored php-cs-fixer and
it cannot run standalone from the repo `vendor/`.

A reported time is only meaningful for a run that completed, so the benchmark
aborts (rather than recording a partial time) if any engine crashes - a signal,
exit 255, or a PHP/Symfony fatal-exception signature in its output.

> Laravel's `tests` dir triggers a pathological php-cs-fixer fix loop, so the
> laravel run is limited to `src`. php-cs-fixer also aborts on
> `symfony/symfony` `src`: `Uid/BinaryUtil.php` carries a `@php-cs-fixer-ignore`
> annotation for a rule php-cs-fixer does not know, which its Runner treats as
> fatal. Files listed in the corpus `bench_remove` are deleted from the clone, so
> every engine runs on the same set.

## Before/after for blink performance work

```bash
bench/blink-before-after.sh                          # origin/main vs this checkout
bench/blink-before-after.sh origin/main ../corpus common 7
```

Builds blink at the base ref and from this checkout, runs both in check mode
over the same corpus (default: this repo's `vendor/`) and set, and prints
median wall and CPU seconds. Runs alternate between the two builds, and the
script fails if their reports differ.
