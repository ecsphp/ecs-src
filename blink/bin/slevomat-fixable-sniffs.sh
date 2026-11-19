#!/usr/bin/env bash
# Regenerates slevomat-fixable-sniffs.txt: every slevomat/coding-standard sniff
# that can fix code (calls addFixableError itself or through its abstract parent),
# as FQCNs. Rerun against a newer slevomat checkout to pick up new sniffs.
#
#   SLEVOMAT_DIR=/path/to/slevomat-checkout bin/slevomat-fixable-sniffs.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SNIFFS="${SLEVOMAT_DIR:-}/SlevomatCodingStandard/Sniffs"
OUT="$ROOT/slevomat-fixable-sniffs.txt"

[ -d "$SNIFFS" ] || { echo "Set SLEVOMAT_DIR to a slevomat/coding-standard checkout" >&2; exit 1; }

fixable=$(grep -rlE 'addFixable(Error|Warning)' "$SNIFFS" --include='*.php' | xargs -n1 basename | sed 's/\.php$//')
parents=$(printf '%s\n' "$fixable" | grep '^Abstract' | paste -sd'|' -)

find "$SNIFFS" -name '*Sniff.php' -print0 \
    | xargs -0 grep -lE "addFixable(Error|Warning)|extends (${parents:-NONE})\b" \
    | sed -E "s#^$SNIFFS/#SlevomatCodingStandard/Sniffs/#; s#\.php\$##; s#/#\\\\#g" \
    | LC_ALL=C sort > "$OUT"

echo "Wrote $(wc -l < "$OUT") fixable sniffs to $OUT"
