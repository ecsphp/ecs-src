#!/usr/bin/env bash
# Regenerates cmd/list-universe/phpcs_sniffs.txt: every PHP_CodeSniffer sniff,
# fixable or report-only, as FQCNs. Reads the PHP_CodeSniffer installed by
# composer in the surrounding repo; rerun after bumping squizlabs/php_codesniffer.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STANDARDS="$ROOT/../vendor/squizlabs/php_codesniffer/src/Standards"
OUT="$ROOT/cmd/list-universe/phpcs_sniffs.txt"

[ -d "$STANDARDS" ] || { echo "PHP_CodeSniffer not found at $STANDARDS, run composer install" >&2; exit 1; }

find "$STANDARDS" -path '*/Sniffs/*' -name '*Sniff.php' ! -name 'Abstract*' \
    | sed -E "s#^$STANDARDS/#PHP_CodeSniffer/Standards/#; s#\.php\$##; s#/#\\\\#g" \
    | LC_ALL=C sort > "$OUT"

echo "Wrote $(wc -l < "$OUT") sniffs to $OUT"
