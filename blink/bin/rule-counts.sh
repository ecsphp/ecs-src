#!/usr/bin/env bash
# Rule coverage counter. Counts the rules blink implements against the ECS
# baseline and renders them into the README between the rule-counts markers.
# Run with --check in CI to fail when the README is stale.
#
#   bin/rule-counts.sh          update README in place
#   bin/rule-counts.sh --check  fail (exit 1) if README is out of date
#
# Sources:
#   ECS   - tracked baseline from ecs-rules.txt ("N rules configured")
#   blink - `blink list-checkers`
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
README="$ROOT/README.md"
RULES_TXT="$ROOT/ecs-rules.txt"
ALL_FIXERS="$ROOT/ecs-all-fixers.txt"
START="<!-- rule-counts:start -->"
END="<!-- rule-counts:end -->"

ecs=$(grep -oE '[0-9]+ rules configured in ECS' "$RULES_TXT" | grep -oE '^[0-9]+')

go build -C "$ROOT" -o /tmp/blink-count . >&2
go_count=$(/tmp/blink-count list-checkers | grep -cE '^[[:space:]]*- ')

# rules ECS configures (across its sets) that blink does not implement yet
missing=""
if [ -f "$ALL_FIXERS" ]; then
    /tmp/blink-count list-checkers | sed 's/^[[:space:]]*-[[:space:]]*//' | LC_ALL=C sort -u > /tmp/blink-impl.txt
    missing=$(LC_ALL=C comm -23 <(LC_ALL=C sort -u "$ALL_FIXERS") /tmp/blink-impl.txt | sed -E 's#.*\\##')
fi
missing_count=$(printf '%s' "$missing" | grep -c . || true)

go_pct=$(( go_count * 100 / ecs ))

block=$(cat <<EOF
$START
| tool | rules | of ECS |
|---|---:|---:|
| [ECS](https://github.com/symplify/easy-coding-standard) (baseline) | $ecs | 100% |
| blink | $go_count | ${go_pct}% |
$END
EOF
)

# Write the block back into README between the markers.
new=$(awk -v start="$START" -v end="$END" -v block="$block" '
  $0 ~ start {print block; skip=1; next}
  $0 ~ end {skip=0; next}
  !skip {print}
' "$README")

# Emit to the CI job summary so every PR shows the counters and the backlog.
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
        echo "### Rule coverage"
        echo "| tool | rules | of ECS |"
        echo "|---|---:|---:|"
        echo "| ECS (baseline) | $ecs | 100% |"
        echo "| blink | $go_count | ${go_pct}% |"
        echo ""
        echo "### Missing in blink ($missing_count of $(LC_ALL=C sort -u "$ALL_FIXERS" 2>/dev/null | grep -c . || echo 0) ECS fixers)"
        if [ -n "$missing" ]; then
            printf '%s\n' "$missing" | sed 's/^/- `/; s/$/`/'
        else
            echo "None - full coverage."
        fi
    } >> "$GITHUB_STEP_SUMMARY"
fi

# Always print the missing list to stdout too (visible in CI logs).
echo "Missing in blink ($missing_count):"
if [ -n "$missing" ]; then
    printf '  %s\n' $missing
fi

if [ "${1:-}" = "--check" ]; then
    if ! diff <(printf '%s\n' "$new") "$README" >/dev/null; then
        echo "README rule counts are stale. Run: bin/rule-counts.sh" >&2
        diff "$README" <(printf '%s\n' "$new") || true
        exit 1
    fi
    echo "Rule counts up to date: ECS $ecs, blink $go_count"
else
    printf '%s\n' "$new" > "$README"
    echo "Updated README: ECS $ecs, blink $go_count"
fi
