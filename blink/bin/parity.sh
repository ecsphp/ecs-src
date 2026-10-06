#!/usr/bin/env bash
# Blink-vs-PHP parity check on a real project, run locally.
#
# Runs the same ECS config both ways on two identical copies of a project:
#   bin/ecs check --fix --blink   (Go binary)
#   bin/ecs check --fix           (PHP engine)
# then diffs the trees. The files whose output differs are written to
# parity.<project>.txt. Pass --only to re-check just those files from the last
# run instead of the whole tree - the tight loop while closing a single gap.
#
#   blink/bin/parity.sh laravel          full run, refresh parity.laravel.txt
#   blink/bin/parity.sh mautic           full run, refresh parity.mautic.txt
#   blink/bin/parity.sh laravel --only   re-check only the files from last run
#
# The project is cloned once into $TMPDIR and reused; pass --fresh to re-clone.
set -euo pipefail

PROJECT="${1:-}"
MODE="${2:-}"

case "$PROJECT" in
    laravel) REPO="https://github.com/laravel/framework.git"; SRC_PATHS="src tests" ;;
    mautic)  REPO="https://github.com/mautic/mautic.git";     SRC_PATHS="app plugins" ;;
    *) echo "usage: blink/bin/parity.sh <laravel|mautic> [--only|--fresh]" >&2; exit 2 ;;
esac

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BLINK_DIR="$ROOT/blink"
WORK="${TMPDIR:-/tmp}/ecs-parity"
CHECKOUT="$WORK/$PROJECT"
OURS="$WORK/$PROJECT-ours"
ECS="$WORK/$PROJECT-ecs"
BLINK_BIN="$WORK/blink"
PARITY_FILE="$BLINK_DIR/parity.$PROJECT.txt"

mkdir -p "$WORK"

go build -C "$BLINK_DIR" -o "$BLINK_BIN" .

if [ "$MODE" = "--fresh" ]; then
    rm -rf "$CHECKOUT"
fi
if [ ! -d "$CHECKOUT/.git" ]; then
    rm -rf "$CHECKOUT"
    git clone --depth 1 "$REPO" "$CHECKOUT"
fi

# withPaths entries for the ECS config, one __DIR__ line per source path
paths_php=""
for p in $SRC_PATHS; do
    paths_php="$paths_php            __DIR__ . '/$p',
"
done

# the config lives inside the checkout so __DIR__ resolves against it; applies
# every prepared set, no per-fixer skips, to exercise the full rule surface
cat > "$CHECKOUT/ecs.php" <<PHP
<?php

declare(strict_types=1);

use Symplify\EasyCodingStandard\Config\ECSConfig;

return ECSConfig::configure()
    ->withPaths([
$paths_php        ])
    ->withSkip([
        '*/node_modules/*',
        '*/Fixture/*',
    ])
    ->withPreparedSets(
        psr12: true,
        perCs: true,
        common: true,
        standaloneLine: true,
        cleanCode: true,
    )
    ->withoutParallel();
PHP

# which files to check: the whole tree, or just last run's divergent list
if [ "$MODE" = "--only" ]; then
    [ -s "$PARITY_FILE" ] || { echo "no $PARITY_FILE from a previous full run" >&2; exit 2; }
    mapfile -t REL_FILES < "$PARITY_FILE"
else
    mapfile -t REL_FILES < <(cd "$CHECKOUT" && find $SRC_PATHS -name '*.php' | sort)
fi

# fresh identical copies for each engine
rm -rf "$OURS" "$ECS"
cp -r "$CHECKOUT" "$OURS"
cp -r "$CHECKOUT" "$ECS"

# in --only mode, hand each engine the explicit file list (resolved per copy);
# a full run uses the config paths so the whole tree is covered
if [ "$MODE" = "--only" ]; then
    ours_paths=(); ecs_paths=()
    for f in "${REL_FILES[@]}"; do
        ours_paths+=("$OURS/$f")
        ecs_paths+=("$ECS/$f")
    done
    ECS_BLINK_BIN="$BLINK_BIN" "$ROOT/bin/ecs" check --fix --no-progress-bar --config "$OURS/ecs.php" --blink "${ours_paths[@]}" || true
    "$ROOT/bin/ecs" check --fix --no-progress-bar --config "$ECS/ecs.php" "${ecs_paths[@]}" || true
else
    ECS_BLINK_BIN="$BLINK_BIN" "$ROOT/bin/ecs" check --fix --no-progress-bar --config "$OURS/ecs.php" --blink || true
    "$ROOT/bin/ecs" check --fix --no-progress-bar --config "$ECS/ecs.php" || true
fi

total=0
differing=()
for f in "${REL_FILES[@]}"; do
    total=$((total + 1))
    if ! diff -q "$OURS/$f" "$ECS/$f" >/dev/null 2>&1; then
        differing+=("$f")
    fi
done
differ=${#differing[@]}
same=$((total - differ))

pct=0
diff_pct=0
if [ "$total" -gt 0 ]; then
    pct=$(awk "BEGIN{printf \"%.2f\", $same * 100 / $total}")
    diff_pct=$(awk "BEGIN{printf \"%.2f\", $differ * 100 / $total}")
fi

# record the divergent files so the next --only run checks just these
if [ "$differ" -gt 0 ]; then
    printf '%s\n' "${differing[@]}" > "$PARITY_FILE"
else
    : > "$PARITY_FILE"
fi

for f in "${differing[@]}"; do
    echo "--- $f (< PHP, > blink)"
    diff "$ECS/$f" "$OURS/$f" | sed 's/^/    /' | head -60 || true
done

echo "identical=$same differing=$differ of $total (${pct}% parity, ${diff_pct}% diff) -> $PARITY_FILE"
