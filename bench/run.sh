#!/usr/bin/env bash
#
# Wall-clock benchmark: ECS (PHP engine) and --blink (Go engine) against the
# native tools they wrap - php-cs-fixer, PHP_CodeSniffer (phpcbf) and Laravel
# Pint - over the same real-world corpus, each in --fix mode on a fresh copy.
#
# Usage:
#   bench/run.sh [corpus.json] [path ...]
#
# corpus.json defaults to blink/parity/corpora/symfony.json. Any paths given
# after it override the corpus .paths (relative to the clone), e.g.
#   bench/run.sh blink/parity/corpora/laravel.json src
# runs over laravel's src only (its tests dir triggers a pathological
# php-cs-fixer fix loop that is not representative).
#
# ECS and --blink run in parallel across all cores; the native tools run at
# their default settings. Numbers are wall-clock seconds and vary with hardware.
set -euo pipefail

fail() { echo "bench: $*" >&2; exit 2; }

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
[ -x "$REPO_ROOT/bin/ecs" ] || fail "bin/ecs not found at $REPO_ROOT"
command -v jq >/dev/null || fail "jq is required"
command -v go >/dev/null || fail "go is required"
command -v composer >/dev/null || fail "composer is required"

CORPUS="${1:-$REPO_ROOT/blink/parity/corpora/symfony.json}"
[ -f "$CORPUS" ] || fail "corpus json not found: $CORPUS"
shift || true

name=$(jq -r '.name' "$CORPUS")
repo=$(jq -r '.repo' "$CORPUS")
ref=$(jq -r '.ref // "HEAD"' "$CORPUS")
if [ "$#" -gt 0 ]; then
    paths=("$@")
else
    mapfile -t paths < <(jq -r '.paths[]' "$CORPUS")
fi
[ "${#paths[@]}" -gt 0 ] || fail "no paths to check"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# 1. build blink from this checkout
echo "==> building blink" >&2
blink_bin="$WORKDIR/blink"
( cd "$REPO_ROOT/blink" && go build -o "$blink_bin" . ) || fail "blink build failed"

# 2. install the native tools in isolation - ECS scopes its own vendored
#    php-cs-fixer, so it cannot be run standalone from the repo vendor
echo "==> installing php-cs-fixer, php_codesniffer, pint" >&2
tools="$WORKDIR/tools"
mkdir -p "$tools"
composer --working-dir="$tools" require \
    friendsofphp/php-cs-fixer squizlabs/php_codesniffer laravel/pint \
    --no-interaction --quiet || fail "native tool install failed"
pcf="$tools/vendor/bin/php-cs-fixer"
phpcbf="$tools/vendor/bin/phpcbf"
pint="$tools/vendor/bin/pint"

# 3. the shared ECS config: corpus paths, skips and prepared sets
ecs_php="$WORKDIR/ecs.php"
{
    echo '<?php'
    echo 'declare(strict_types=1);'
    echo 'use Symplify\EasyCodingStandard\Config\ECSConfig;'
    echo 'return ECSConfig::configure()'
    sets_args=$(jq -r '.sets | map("\(.): true") | join(", ")' "$CORPUS")
    echo "    ->withPreparedSets(${sets_args})"
    skip_args=$(jq -r '(.skip // []) | map("'"'"'\(.)'"'"'") | join(", ")' "$CORPUS")
    [ -n "$skip_args" ] && echo "    ->withSkip([${skip_args}])"
    echo '    ->withParallel();'
} > "$ecs_php"

# 4. clone the corpus once
echo "==> cloning $repo ($ref)" >&2
src="$WORKDIR/corpus"
git clone --depth 1 --branch "$ref" "$repo" "$src" >/dev/null 2>&1 \
    || git clone --depth 1 "$repo" "$src" >/dev/null 2>&1 \
    || fail "clone failed: $repo"
rev=$(git -C "$src" rev-parse --short HEAD 2>/dev/null || echo "?")
file_count=$(cd "$src" && find "${paths[@]}" -name '*.php' 2>/dev/null | wc -l)

# abs_paths helper: resolve the corpus-relative paths inside a given tree
abs() { local tree="$1"; shift; for p in "$@"; do echo "$tree/$p"; done; }

# time a command, in seconds, over a fresh copy of the corpus paths
declare -A SECONDS_BY
measure() {
    local label="$1"; shift
    local tree="$WORKDIR/tree"
    rm -rf "$tree"; mkdir -p "$tree"
    for p in "${paths[@]}"; do mkdir -p "$tree/$(dirname "$p")"; cp -r "$src/$p" "$tree/$p"; done
    local start end
    start=$(date +%s.%N)
    "$@" >/dev/null 2>&1 || true
    end=$(date +%s.%N)
    SECONDS_BY[$label]=$(awk "BEGIN{printf \"%.1f\", $end - $start}")
    printf '    %-28s %6ss\n' "$label" "${SECONDS_BY[$label]}" >&2
}

tree="$WORKDIR/tree"
echo "==> $name @ $rev: $file_count PHP files in ${paths[*]}" >&2

measure "ECS (PHP)"      "$REPO_ROOT/bin/ecs" check --fix --no-progress-bar --memory-limit=1G --config "$ecs_php" $(abs "$tree" "${paths[@]}")
ECS_BLINK_BIN="$blink_bin" measure "--blink (Go)" "$REPO_ROOT/bin/ecs" check --fix --no-progress-bar --config "$ecs_php" --blink $(abs "$tree" "${paths[@]}")
measure "php-cs-fixer (@PER-CS,@Symfony)" "$pcf" fix $(abs "$tree" "${paths[@]}") --rules=@PER-CS,@Symfony --using-cache=no --no-interaction -q
measure "phpcbf (PSR12)" "$phpcbf" --standard=PSR12 --parallel="$(nproc)" $(abs "$tree" "${paths[@]}")
measure "pint (laravel preset)" "$pint" $(abs "$tree" "${paths[@]}")

# 5. markdown table on stdout
echo
echo "## $name @ $rev - $file_count PHP files (${paths[*]})"
echo
echo "| engine | time |"
echo "| --- | ---: |"
for label in "ECS (PHP)" "--blink (Go)" "php-cs-fixer (@PER-CS,@Symfony)" "phpcbf (PSR12)" "pint (laravel preset)"; do
    echo "| ${label} | ${SECONDS_BY[$label]:-?} s |"
done
