#!/usr/bin/env bash
# Rule coverage counter. Counts the rules blink implements against the full
# fixer universe and renders them into the README between the rule-counts
# markers. Run with --check in CI to fail when the README is stale.
#
#   bin/rule-counts.sh          update README in place
#   bin/rule-counts.sh --check  fail (exit 1) if README is out of date
#
# Both modes fail (exit 1) while any PHP-CS-Fixer fixer or fixable
# PHP_CodeSniffer sniff is missing in blink.
#
# Sources:
#   universe - `list-universe`: all PHP-CS-Fixer + Symplify fixers, all fixable
#              PHP_CodeSniffer sniffs, plus the sniffs ECS references from its sets
#   blink    - `blink list-checkers`
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
README="$ROOT/README.md"
ALL_FIXERS="$ROOT/ecs-all-fixers.txt"
START="<!-- rule-counts:start -->"
END="<!-- rule-counts:end -->"
SLEVOMAT="$ROOT/slevomat-fixable-sniffs.txt"
SLEVOMAT_START="<!-- slevomat-counts:start -->"
SLEVOMAT_END="<!-- slevomat-counts:end -->"
# the repository README carries one combined parity table
ROOT_README="$ROOT/../README.md"
PARITY_START="<!-- parity-counts:start -->"
PARITY_END="<!-- parity-counts:end -->"

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
missing_required=$(LC_ALL=C comm -23 <(LC_ALL=C sort -u /tmp/blink-universe.txt) /tmp/blink-impl.txt \
    | grep -E '^(PhpCsFixer|PHP_CodeSniffer)\\' || true)

# blink rules within the universe; blink's own extras never push parity past 100%
implemented=$(( ecs - missing_count ))
go_pct=$(( implemented * 100 / ecs ))

# per-source coverage rows: universe rules with a given FQCN prefix vs those blink implements
coverage_row() {
    local label="$1" prefix="$2" total done
    total=$(grep -c "^${prefix}" /tmp/blink-universe.txt || true)
    done=$(LC_ALL=C comm -12 <(LC_ALL=C sort -u /tmp/blink-universe.txt) /tmp/blink-impl.txt | grep -c "^${prefix}" || true)
    echo "| $label | $done | $total | $(( total > 0 ? done * 100 / total : 100 ))% |"
}

table=$(cat <<EOF
| source | in blink | universe | coverage |
|---|---:|---:|---:|
$(coverage_row "PHP-CS-Fixer" 'PhpCsFixer\\')
$(coverage_row "Symplify" 'Symplify\\')
$(coverage_row "PHP_CodeSniffer" 'PHP_CodeSniffer\\')
| **total** | **$implemented** | **$ecs** | **${go_pct}%** |
EOF
)

block=$(cat <<EOF
$START
$table

blink runs $go_count rules in all, its own extras included.
$END
EOF
)

# slevomat sniffs are blink-only extras, measured on their own against every
# fixable slevomat sniff (slevomat-fixable-sniffs.txt), per category
slevomat_done=$(LC_ALL=C comm -12 <(LC_ALL=C sort -u "$SLEVOMAT") /tmp/blink-impl.txt)
slevomat_total=$(grep -c . "$SLEVOMAT")
slevomat_done_count=$(printf '%s' "$slevomat_done" | grep -c . || true)
slevomat_rows=$(cut -d'\' -f3 "$SLEVOMAT" | LC_ALL=C sort -u | while read -r category; do
    total=$(grep -cF "\\Sniffs\\${category}\\" "$SLEVOMAT")
    done=$(printf '%s\n' "$slevomat_done" | grep -cF "\\Sniffs\\${category}\\" || true)
    echo "| $category | $done | $total | $(( done * 100 / total ))% |"
done)

slevomat_table=$(cat <<EOF
| slevomat category | in blink | fixable sniffs | coverage |
|---|---:|---:|---:|
$slevomat_rows
| **total** | **$slevomat_done_count** | **$slevomat_total** | **$(( slevomat_done_count * 100 / slevomat_total ))%** |
EOF
)

slevomat_block=$(cat <<EOF
$SLEVOMAT_START
$slevomat_table
$SLEVOMAT_END
EOF
)

all_done=$(( implemented + slevomat_done_count ))
all_total=$(( ecs + slevomat_total ))
parity_block=$(cat <<EOF
$PARITY_START
| source | in blink | rules | coverage |
|---|---:|---:|---:|
$(coverage_row "PHP-CS-Fixer" 'PhpCsFixer\\')
$(coverage_row "Symplify" 'Symplify\\')
$(coverage_row "PHP_CodeSniffer" 'PHP_CodeSniffer\\')
| slevomat | $slevomat_done_count | $slevomat_total | $(( slevomat_done_count * 100 / slevomat_total ))% |
| **total** | **$all_done** | **$all_total** | **$(( all_done * 100 / all_total ))%** |
$PARITY_END
EOF
)
new_root=$(awk -v start="$PARITY_START" -v end="$PARITY_END" -v block="$parity_block" '
  $0 ~ start {print block; skip=1; next}
  $0 ~ end {skip=0; next}
  !skip {print}
' "$ROOT_README")

# Write the blocks back into README between their markers.
new=$(awk -v start="$START" -v end="$END" -v block="$block" \
    -v sstart="$SLEVOMAT_START" -v send="$SLEVOMAT_END" -v sblock="$slevomat_block" '
  $0 ~ start {print block; skip=1; next}
  $0 ~ end {skip=0; next}
  $0 ~ sstart {print sblock; skip=1; next}
  $0 ~ send {skip=0; next}
  !skip {print}
' "$README")

# Emit to the CI job summary so every PR shows the counters and the backlog.
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
        echo "### Rule coverage"
        echo "$table"
        echo ""
        echo "### slevomat coverage"
        echo "$slevomat_table"
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
    if ! diff <(printf '%s\n' "$new_root") "$ROOT_README" >/dev/null; then
        echo "Repository README parity table is stale. Run: bin/rule-counts.sh" >&2
        diff "$ROOT_README" <(printf '%s\n' "$new_root") || true
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
    printf '%s\n' "$new_root" > "$ROOT_README"
    cp /tmp/blink-universe.txt "$ALL_FIXERS"
    echo "Updated README and ecs-all-fixers.txt: universe $ecs, blink $go_count"
fi

if [ -n "$missing_required" ]; then
    echo "PHP-CS-Fixer fixers and PHP_CodeSniffer sniffs not implemented in blink:" >&2
    printf '  %s\n' $missing_required >&2
    exit 1
fi
