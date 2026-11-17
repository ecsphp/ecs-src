#!/usr/bin/env bash
#
# Before/after benchmark for blink performance work: builds blink from a base
# ref and from this checkout, runs both in check mode over the same corpus and
# set, and prints median wall and CPU (user+sys) seconds. Runs alternate between
# the two builds so machine load drifts hit both equally. The reports of both
# builds must be identical - a speedup that changes output is a bug.
#
# Usage:
#   bench/blink-before-after.sh [base-ref] [corpus-dir] [set] [runs]
#
# Defaults: base-ref origin/main, corpus-dir vendor/ of this repo, set common
# (the heaviest prepared set, docblock rules included), 5 runs. CPU seconds
# are the steadier number on a busy machine; wall seconds depend on free cores.
set -euo pipefail

fail() { echo "bench: $*" >&2; exit 2; }

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_REF="${1:-origin/main}"
CORPUS="$(realpath "${2:-$REPO_ROOT/vendor}")"
SET="${3:-common}"
RUNS="${4:-5}"

command -v go >/dev/null || fail "go is required"
[ -d "$CORPUS" ] || fail "corpus dir not found: $CORPUS"
git -C "$REPO_ROOT" rev-parse --verify --quiet "$BASE_REF^{commit}" >/dev/null || fail "unknown base ref: $BASE_REF"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

echo "==> building blink at $BASE_REF (before) and this checkout (after)" >&2
mkdir -p "$WORKDIR/base"
git -C "$REPO_ROOT" archive "$BASE_REF" blink | tar -x -C "$WORKDIR/base"
( cd "$WORKDIR/base/blink" && go build -o "$WORKDIR/before" . ) || fail "base build failed"
( cd "$REPO_ROOT/blink" && go build -o "$WORKDIR/after" . ) || fail "build failed"

printf '{"paths": ["%s"], "sets": ["%s"]}\n' "$CORPUS" "$SET" > "$WORKDIR/blink.json"

# run <name> <iteration>: one check run, appending "wall cpu" to <name>.times
run() {
    local start end
    start=$(date +%s.%N)
    /usr/bin/time -f "%U %S" -o "$WORKDIR/$1.rusage" \
        "$WORKDIR/$1" --config "$WORKDIR/blink.json" > "$WORKDIR/$1.out.$2" 2>&1 || true
    end=$(date +%s.%N)
    read -r user sys < <(tail -n 1 "$WORKDIR/$1.rusage")
    echo "$(echo "$end - $start" | bc) $(echo "$user + $sys" | bc)" >> "$WORKDIR/$1.times"
}

echo "==> $RUNS runs each, set $SET, corpus $CORPUS" >&2
for i in $(seq "$RUNS"); do
    run before "$i"
    run after "$i"
done

# the summary line carries run time and memory, so it is left out of the compare
grep -v ' · ' "$WORKDIR/before.out.1" > "$WORKDIR/before.report"
grep -v ' · ' "$WORKDIR/after.out.1" > "$WORKDIR/after.report"
if ! cmp -s "$WORKDIR/before.report" "$WORKDIR/after.report"; then
    diff "$WORKDIR/before.report" "$WORKDIR/after.report" | head -40 >&2
    fail "reports differ between before and after"
fi

median() { sort -n | awk '{ a[NR] = $1 } END { print (NR % 2) ? a[(NR + 1) / 2] : (a[NR / 2] + a[NR / 2 + 1]) / 2 }'; }

before_wall=$(cut -d' ' -f1 "$WORKDIR/before.times" | median)
before_cpu=$(cut -d' ' -f2 "$WORKDIR/before.times" | median)
after_wall=$(cut -d' ' -f1 "$WORKDIR/after.times" | median)
after_cpu=$(cut -d' ' -f2 "$WORKDIR/after.times" | median)

echo
printf '%-8s %10s %10s\n' "" "wall (s)" "cpu (s)"
printf '%-8s %10.2f %10.2f\n' before "$before_wall" "$before_cpu"
printf '%-8s %10.2f %10.2f\n' after "$after_wall" "$after_cpu"
printf '%-8s %9.1f%% %9.1f%%\n' change \
    "$(echo "($after_wall - $before_wall) * 100 / $before_wall" | bc -l)" \
    "$(echo "($after_cpu - $before_cpu) * 100 / $before_cpu" | bc -l)"
echo "reports identical ($(wc -l < "$WORKDIR/after.report") lines)"
