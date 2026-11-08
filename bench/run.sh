#!/usr/bin/env bash
set -euo pipefail

# Speed benchmark for ECS: runs `bin/ecs check` (no fix) over a target project
# with a heavy config (PER-CS + all Symplify sets), cache cleared each run.
#
# Usage:
#   bench/run.sh [TARGET_PATH] [RUNS]
#
# Defaults:
#   TARGET_PATH  /var/www/rector-src/src
#   RUNS         3

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${1:-/var/www/rector-src/src}"
RUNS="${2:-3}"

if [[ ! -e "${TARGET}" ]]; then
    echo "Target does not exist: ${TARGET}" >&2
    exit 1
fi

export BENCH_TARGET="${TARGET}"

FILE_COUNT="$(find "${TARGET}" -name '*.php' | wc -l | tr -d ' ')"

echo "ECS speed benchmark"
echo "  target: ${TARGET} (${FILE_COUNT} PHP files)"
echo "  config: bench/ecs-bench.php (PER-CS + all Symplify sets)"
echo "  runs:   ${RUNS} (cache cleared before each)"
echo

total="0"
for i in $(seq 1 "${RUNS}"); do
    start="$(date +%s.%N)"
    php "${ROOT_DIR}/bin/ecs" check \
        --config "${ROOT_DIR}/bench/ecs-bench.php" \
        --clear-cache \
        --no-progress-bar \
        --no-error-table \
        --no-diffs \
        >/dev/null 2>&1 || true
    end="$(date +%s.%N)"
    elapsed="$(echo "${end} - ${start}" | bc)"
    printf "  run %d: %.2fs\n" "${i}" "${elapsed}"
    total="$(echo "${total} + ${elapsed}" | bc)"
done

avg="$(echo "scale=2; ${total} / ${RUNS}" | bc)"
echo
printf "  average: %ss over %d runs\n" "${avg}" "${RUNS}"
