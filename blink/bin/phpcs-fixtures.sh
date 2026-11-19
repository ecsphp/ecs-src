#!/usr/bin/env bash
# Regenerates internal/fixer/rules/testdata/phpcs: PHP_CodeSniffer's own sniff
# test inputs (*.inc from src/Standards/*/Tests) with the output ECS produces
# for each, so blink is held to ECS, not to raw PHPCS. ECS runs sniffs with
# annotations off, so phpcs:set / phpcs:ignore in the inputs have no effect.
#
#   bin/phpcs-fixtures.sh <PHP_CodeSniffer checkout at the composer-installed tag>
#
# Covers only the sniffs blink implements; rerun after porting a sniff.
# An input whose ECS output equals it gets no .fixed file.
set -euo pipefail

UPSTREAM="${1:?usage: bin/phpcs-fixtures.sh <PHP_CodeSniffer checkout>}/src/Standards"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ECS="$ROOT/../bin/ecs"
OUT="$ROOT/internal/fixer/rules/testdata/phpcs"
[ -d "$UPSTREAM" ] || { echo "no Standards dir in $UPSTREAM" >&2; exit 1; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
go build -C "$ROOT" -o "$work/blink" . >&2
sniffs=$("$work/blink" list-checkers | sed -n 's/^[[:space:]]*- \(PHP_CodeSniffer\\.*\)$/\1/p' | LC_ALL=C sort -u)

find "$OUT" -name "*.inc*" -delete 2>/dev/null || true

fixture() {
    local fqcn="$1" rel dir base out tmp
    rel=$(printf '%s' "$fqcn" | sed -E 's#^PHP_CodeSniffer\\Standards\\##; s#\\Sniffs\\#/Tests/#; s#\\#/#g; s#Sniff$#UnitTest#')
    dir=$(dirname "$rel"); base=$(basename "$rel")
    out="$OUT/${dir/\/Tests\//\/}"
    tmp="$work/${rel//\//_}"
    mkdir -p "$out" "$tmp"
    local inputs
    mapfile -t inputs < <(find "$UPSTREAM/$dir" -maxdepth 1 -regex ".*/$base\(\.[0-9]+\)?\.inc" | LC_ALL=C sort)
    [ "${#inputs[@]}" -gt 0 ] || { echo "no fixtures for $fqcn" >&2; return 1; }
    for f in "${inputs[@]}"; do
        cp "$f" "$out/"
        cp "$f" "$tmp/$(basename "$f").php"
    done
    printf '<?php\nreturn Symplify\\EasyCodingStandard\\Config\\ECSConfig::configure()->withRules([%s::class])->withCache(%s)->withoutParallel();\n' \
        "\\$fqcn" "'$tmp.cache'" > "$tmp.ecs.php"
    "$ECS" check --fix --no-progress-bar --no-error-table --config "$tmp.ecs.php" "$tmp" >/dev/null 2>&1 || true
    for f in "${inputs[@]}"; do
        local name; name=$(basename "$f")
        cmp -s "$f" "$tmp/$name.php" || cp "$tmp/$name.php" "$out/$name.fixed"
    done
}
export -f fixture
export UPSTREAM OUT ECS work

printf '%s\n' "$sniffs" | xargs -d '\n' -P "$(nproc)" -n1 bash -c 'fixture "$1"' _
echo "Wrote $(find "$OUT" -name '*.inc' | wc -l) inputs, $(find "$OUT" -name '*.fixed' | wc -l) fixed outputs to $OUT"
