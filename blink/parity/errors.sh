#!/usr/bin/env bash
#
# Report-only parity: run every rule blink can report but not fix over a
# real-world corpus in check mode, once with --blink and once with the PHP ECS
# engine, and fail when the reported errors or the exit codes differ.
#
# Usage:
#   blink/parity/errors.sh <corpus.json> [--workdir DIR] [--keep]
#
# Reads "name", "repo", "ref", "paths" and "skip" from the corpus json; its sets
# and threshold are ignored, any difference fails.
#
# Exit status: 0 when blink matches ECS, 1 when it differs, 2 on a setup error.
set -euo pipefail

# slevomat/coding-standard commit the slevomat ports follow, as in slevomat.sh
SLEVOMAT_REF=606a440f66317d2fa6a9eab5864c0bea1ecde35f

fail() { echo "errors parity: $*" >&2; exit 2; }

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

name=$(jq -r '.name' "$CORPUS")
repo=$(jq -r '.repo' "$CORPUS")
ref=$(jq -r '.ref // "HEAD"' "$CORPUS")
mapfile -t paths < <(jq -r '.paths[]' "$CORPUS")

if [ -z "$WORKDIR" ]; then
    WORKDIR="$(mktemp -d)"
    [ "$KEEP" -eq 1 ] || trap 'rm -rf "$WORKDIR"' EXIT
fi
mkdir -p "$WORKDIR"

blink_bin="$WORKDIR/blink"
( cd "$REPO_ROOT/blink" && go build -o "$blink_bin" . ) || fail "blink build failed"
mapfile -t reporters < <(cd "$REPO_ROOT/blink" && go run ./cmd/list-reporters)
[ "${#reporters[@]}" -gt 0 ] || fail "no report-only rules found"

# slevomat is no ECS dependency, so ECS autoloads a pinned checkout
slv="$WORKDIR/slevomat"
if [ ! -d "$slv/vendor" ]; then
    rm -rf "$slv"
    git init -q "$slv"
    git -C "$slv" fetch -q --depth 1 https://github.com/slevomat/coding-standard.git "$SLEVOMAT_REF" \
        && git -C "$slv" checkout -q FETCH_HEAD || fail "slevomat fetch failed"
    ( cd "$slv" && composer install -q --no-dev --no-plugins ) || fail "slevomat composer install failed"
fi

src="$WORKDIR/corpus"
rm -rf "$src"
git clone --depth 1 --branch "$ref" "$repo" "$src" >/dev/null 2>&1 \
    || git clone --depth 1 "$repo" "$src" >/dev/null 2>&1 \
    || fail "clone failed: $repo"
rev=$(git -C "$src" rev-parse --short HEAD 2>/dev/null || echo "?")

abs_paths=()
for p in "${paths[@]}"; do
    [ -d "$src/$p" ] && abs_paths+=("$src/$p")
done
[ "${#abs_paths[@]}" -gt 0 ] || fail "corpus $name has no existing paths"

ecs_php="$WORKDIR/ecs.php"
{
    echo '<?php'
    echo 'declare(strict_types=1);'
    cat <<PHP
spl_autoload_register(static function (string \$class): void {
    foreach (['SlevomatCodingStandard\\\\' => '$slv/SlevomatCodingStandard/', 'PHPStan\\\\PhpDocParser\\\\' => '$slv/vendor/phpstan/phpdoc-parser/src/'] as \$prefix => \$dir) {
        if (str_starts_with(\$class, \$prefix)) {
            \$file = \$dir . str_replace('\\\\', '/', substr(\$class, strlen(\$prefix))) . '.php';
            if (is_file(\$file)) {
                require \$file;
            }
            return;
        }
    }
}, true, true);
PHP
    echo 'use Symplify\EasyCodingStandard\Config\ECSConfig;'
    echo 'return ECSConfig::configure()'
    echo '    ->withRules(['
    for r in "${reporters[@]}"; do echo "        \\$r::class,"; done
    echo '    ])'
    skip_args=$(jq -r '(.skip // []) | map("'"'"'\(.)'"'"'") | join(", ")' "$CORPUS")
    [ -n "$skip_args" ] && echo "    ->withSkip([${skip_args}])"
    echo '    ->withParallel();'
} > "$ecs_php"

# check mode never writes, so both engines read the same tree
set +e
ECS_BLINK_BIN="$blink_bin" "$REPO_ROOT/bin/ecs" check --no-progress-bar --output-format=json \
    --config "$ecs_php" --blink "${abs_paths[@]}" > "$WORKDIR/blink.json" 2>"$WORKDIR/blink.err"
blink_exit=$?
"$REPO_ROOT/bin/ecs" check --no-progress-bar --output-format=json --memory-limit=1G \
    --config "$ecs_php" "${abs_paths[@]}" > "$WORKDIR/ecs.json" 2>"$WORKDIR/ecs.err"
ecs_exit=$?
set -e

normalize() {
    jq -r --arg root "$src/" \
        '.files // {} | to_entries[] | .value.errors[]? | "\(.file_path | ltrimstr($root)):\(.line) \(.source_class) \(.message)"' \
        "$1" | LC_ALL=C sort
}
jq -e . "$WORKDIR/blink.json" >/dev/null 2>&1 || { cat "$WORKDIR/blink.err" >&2; fail "blink output is not json"; }
jq -e . "$WORKDIR/ecs.json" >/dev/null 2>&1 || { cat "$WORKDIR/ecs.err" >&2; fail "ECS output is not json"; }
normalize "$WORKDIR/blink.json" > "$WORKDIR/blink.txt"
normalize "$WORKDIR/ecs.json" > "$WORKDIR/ecs.txt"

errors_diff=$(diff -u --label blink --label ecs "$WORKDIR/blink.txt" "$WORKDIR/ecs.txt" || true)
blink_count=$(wc -l < "$WORKDIR/blink.txt")
ecs_count=$(wc -l < "$WORKDIR/ecs.txt")

emit() {
    echo "## blink vs ECS report-only errors: $name @ $rev"
    echo
    echo "| metric | blink | ECS |"
    echo "|---|---|---|"
    echo "| rules | ${#reporters[@]} | ${#reporters[@]} |"
    echo "| errors | $blink_count | $ecs_count |"
    echo "| exit code | $blink_exit | $ecs_exit |"
    if [ -n "$errors_diff" ]; then
        echo
        echo "### differing errors ('-' blink, '+' ECS)"
        echo
        echo '```diff'
        echo "$errors_diff" | head -n 300
        echo '```'
    fi
}
emit
[ -n "${GITHUB_STEP_SUMMARY:-}" ] && emit >> "$GITHUB_STEP_SUMMARY"

if [ -n "$errors_diff" ] || [ "$blink_exit" -ne "$ecs_exit" ]; then
    echo "::error::blink report-only errors differ from ECS on $name"
    exit 1
fi
exit 0
