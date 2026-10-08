#!/usr/bin/env bash
#
# Blink parity harness: run blink (--blink) and the PHP ECS engine over the same
# real-world corpus with the same prepared sets, then report every file whose
# blink output differs from ECS, grouped by the ECS checker that caused it.
#
# Usage:
#   blink/parity/run.sh <corpus.json> [--workdir DIR] [--keep]
#
# A corpus.json describes one codebase:
#   {
#     "name": "laravel",
#     "repo": "https://github.com/laravel/framework.git",
#     "ref": "HEAD",                      // optional, default HEAD
#     "paths": ["src", "tests"],          // dirs to check, relative to the clone
#     "skip":  ["*/node_modules/*"],       // optional ECS skip patterns
#     "sets":  ["psr12","perCs","common","standaloneLine","cleanCode"],
#     "max_diff_percent": 2.0              // gate: non-zero exit when exceeded
#   }
#
# Exit status: 0 when the differing share is within max_diff_percent, 1 when it
# is exceeded, 2 on a setup error. A $GITHUB_STEP_SUMMARY, when set, receives the
# same categorized report as a collapsible markdown table.
set -euo pipefail

fail() { echo "parity: $*" >&2; exit 2; }

CORPUS=""
WORKDIR=""
KEEP=0
while [ $# -gt 0 ]; do
    case "$1" in
        --workdir) WORKDIR="$2"; shift 2 ;;
        --keep) KEEP=1; shift ;;
        -*) fail "unknown flag $1" ;;
        *) CORPUS="$1"; shift ;;
    esac
done
[ -n "$CORPUS" ] && [ -f "$CORPUS" ] || fail "corpus json not found: $CORPUS"
command -v jq >/dev/null || fail "jq is required"
command -v go >/dev/null || fail "go is required"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
[ -x "$REPO_ROOT/bin/ecs" ] || fail "bin/ecs not found at $REPO_ROOT"

name=$(jq -r '.name' "$CORPUS")
repo=$(jq -r '.repo' "$CORPUS")
ref=$(jq -r '.ref // "HEAD"' "$CORPUS")
threshold=$(jq -r '.max_diff_percent' "$CORPUS")
mapfile -t paths < <(jq -r '.paths[]' "$CORPUS")
[ "${#paths[@]}" -gt 0 ] || fail "corpus $name has no paths"

if [ -z "$WORKDIR" ]; then
    WORKDIR="$(mktemp -d)"
    [ "$KEEP" -eq 1 ] || trap 'rm -rf "$WORKDIR"' EXIT
fi
mkdir -p "$WORKDIR"

# 1. build blink from this checkout
blink_bin="$WORKDIR/blink"
( cd "$REPO_ROOT/blink" && go build -o "$blink_bin" . ) || fail "blink build failed"

# 2. the shared ECS config: the corpus paths, skips and prepared sets
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
    echo '    ->withoutParallel();'
} > "$ecs_php"

# 3. fetch the corpus once, then two identical copies
src="$WORKDIR/corpus"
rm -rf "$src"
git clone --depth 1 --branch "$ref" "$repo" "$src" >/dev/null 2>&1 \
    || git clone --depth 1 "$repo" "$src" >/dev/null 2>&1 \
    || fail "clone failed: $repo"
rev=$(git -C "$src" rev-parse --short HEAD 2>/dev/null || echo "?")
ours="$WORKDIR/ours"; ecs="$WORKDIR/ecs"
rm -rf "$ours" "$ecs"; cp -r "$src" "$ours"; cp -r "$src" "$ecs"

abs_paths_ours=(); abs_paths_ecs=(); abs_paths_src=()
for p in "${paths[@]}"; do
    [ -d "$ours/$p" ] && abs_paths_ours+=("$ours/$p")
    [ -d "$ecs/$p" ] && abs_paths_ecs+=("$ecs/$p")
    [ -d "$src/$p" ] && abs_paths_src+=("$src/$p")
done

# 4. fix one tree with blink, the other with the PHP engine
ECS_BLINK_BIN="$blink_bin" "$REPO_ROOT/bin/ecs" check --fix --no-progress-bar \
    --config "$ecs_php" --blink "${abs_paths_ours[@]}" >/dev/null 2>&1 || true
"$REPO_ROOT/bin/ecs" check --fix --no-progress-bar \
    --config "$ecs_php" "${abs_paths_ecs[@]}" >/dev/null 2>&1 || true

# 5. the checkers ECS applied on the pristine tree, as JSON, for attribution
ecs_json="$WORKDIR/ecs.json"
"$REPO_ROOT/bin/ecs" check --no-progress-bar --output-format=json \
    --config "$ecs_php" "${abs_paths_src[@]}" > "$ecs_json" 2>/dev/null || echo '{}' > "$ecs_json"

# 6. diff the two trees, attribute each differing file to the ECS checkers that
#    touched it on the pristine sample
report="$WORKDIR/report.txt"
: > "$report"
total=0; differ=0
declare -A by_checker
while IFS= read -r f; do
    total=$((total + 1))
    diff -q "$ours/$f" "$ecs/$f" >/dev/null 2>&1 && continue
    differ=$((differ + 1))
    abs="$src/$f"
    checkers=$(jq -r --arg k "$abs" '.files[$k].diffs[]?.applied_checkers[]?' "$ecs_json" 2>/dev/null \
        | sed -E 's#.*\\##' | sort -u | paste -sd, - || true)
    echo "--- $f${checkers:+  [${checkers}]}" >> "$report"
    for c in $(echo "$checkers" | tr ',' ' '); do by_checker[$c]=$(( ${by_checker[$c]:-0} + 1 )); done
done < <(cd "$ours" && find "${paths[@]}" -name '*.php' 2>/dev/null | sort)

pct=$(awk "BEGIN{ if ($total>0) printf \"%.2f\", $differ*100/$total; else print \"0.00\" }")

emit() {
    echo "## blink vs ECS parity: $name @ $rev"
    echo
    echo "| metric | value |"
    echo "|---|---|"
    echo "| files | $total |"
    echo "| differing | $differ |"
    echo "| diff | ${pct}% (max ${threshold}%) |"
    echo
    echo "### differing files by ECS checker"
    echo
    echo "| count | checker |"
    echo "|---|---|"
    for c in "${!by_checker[@]}"; do echo "${by_checker[$c]} $c"; done | sort -rn \
        | while read -r n c; do echo "| $n | \`$c\` |"; done
}
emit
[ -n "${GITHUB_STEP_SUMMARY:-}" ] && emit >> "$GITHUB_STEP_SUMMARY"
echo
echo "report: $report"
echo "identical=$((total - differ)) differing=$differ of $total (${pct}% diff, max ${threshold}%)"

awk "BEGIN{ exit !($differ*100/$total > $threshold) }" && {
    echo "::error::blink differs from ECS on ${pct}% of $name files ($differ of $total), over ${threshold}%"
    exit 1
}
exit 0
