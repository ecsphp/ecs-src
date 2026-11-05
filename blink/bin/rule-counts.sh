#!/usr/bin/env bash
# Rule coverage counter. Counts the rules blink implements against the full
# fixer universe and renders them into the README between the rule-counts
# markers. Run with --check in CI to fail when the README is stale.
#
#   bin/rule-counts.sh          update README in place
#   bin/rule-counts.sh --check  fail (exit 1) if README is out of date
#
# Sources:
#   universe - `list-universe`: all PHP-CS-Fixer + Symplify fixers plus the
#              PHP_CodeSniffer sniffs ECS references from its sets
#   blink    - `blink list-checkers`
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
README="$ROOT/README.md"
ALL_FIXERS="$ROOT/ecs-all-fixers.txt"
START="<!-- rule-counts:start -->"
END="<!-- rule-counts:end -->"

go build -C "$ROOT" -o /tmp/blink-count . >&2
/tmp/blink-count list-checkers > /tmp/blink-checkers.txt
# blink's own fixers only; the deprecated names listed after them are not rules of their own
go_count=$(awk '/^Deprecated/ {exit} /^[[:space:]]*- / {n++} END {print n+0}' /tmp/blink-checkers.txt)

# the universe blink is measured against (parity denominator)
go run -C "$ROOT" ./cmd/list-universe > /tmp/blink-universe.txt
ecs=$(grep -c . /tmp/blink-universe.txt)

# universe fixers blink has not ported yet; a deprecated name blink runs counts as ported
sed 's/^[[:space:]]*-[[:space:]]*//' /tmp/blink-checkers.txt | LC_ALL=C sort -u > /tmp/blink-impl.txt
missing=$(LC_ALL=C comm -23 <(LC_ALL=C sort -u /tmp/blink-universe.txt) /tmp/blink-impl.txt | sed -E 's#.*\\##')
missing_count=$(printf '%s' "$missing" | grep -c . || true)

# blink rules within the universe; blink's own extras never push parity past 100%
implemented=$(( ecs - missing_count ))
go_pct=$(( implemented * 100 / ecs ))

block=$(cat <<EOF
$START
| tool | rules | coverage |
|---|---:|---:|
| fixer universe (PHP-CS-Fixer + Symplify + in-set sniffs) | $ecs | 100% |
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
        echo "| tool | rules | coverage |"
        echo "|---|---:|---:|"
        echo "| fixer universe | $ecs | 100% |"
        echo "| blink | $go_count | ${go_pct}% |"
        echo ""
        echo "### Missing in blink ($missing_count of $ecs universe fixers)"
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
    if ! diff "$ALL_FIXERS" /tmp/blink-universe.txt >/dev/null; then
        echo "ecs-all-fixers.txt is stale. Run: bin/rule-counts.sh" >&2
        diff "$ALL_FIXERS" /tmp/blink-universe.txt || true
        exit 1
    fi
    echo "Rule counts up to date: universe $ecs, blink $go_count"
else
    printf '%s\n' "$new" > "$README"
    cp /tmp/blink-universe.txt "$ALL_FIXERS"
    echo "Updated README and ecs-all-fixers.txt: universe $ecs, blink $go_count"
fi
