#!/usr/bin/env bash
#
# Slevomat parity: run each fixable slevomat sniff alone over a real-world corpus,
# once with --blink and once with the PHP ECS engine, and fail when any file
# differs. Each sniff runs on its own, since blink runs rules one after another
# while PHP_CodeSniffer loops all sniffs together.
#
# Usage:
#   blink/parity/slevomat.sh <corpus.json> [sniff-regex] [--workdir DIR] [--keep]
#
# Reads "name", "repo", "ref", "paths" and "skip" from the corpus json. The
# optional regex picks sniffs from blink/slevomat-fixable-sniffs.txt, e.g.
# '\\Classes\\'. A file that differs after the batch run is rechecked with a
# fresh single-file ECS run, because PHP_CodeSniffer keeps fixer state between
# files; a recheck that ECS turns into invalid PHP is skipped, since blink's
# lint guard keeps the original there.
#
# Exit status: 0 when blink matches ECS, 1 when it differs, 2 on a setup error.
set -euo pipefail

# slevomat/coding-standard commit the blink ports and goldens follow
SLEVOMAT_REF=606a440f66317d2fa6a9eab5864c0bea1ecde35f

fail() { echo "slevomat parity: $*" >&2; exit 2; }

START=$(date +%s)
log() { printf '==> [%4ds] %s\n' "$(( $(date +%s) - START ))" "$*" >&2; }

CORPUS=""
FILTER=""
WORKDIR=""
KEEP=0
while [ $# -gt 0 ]; do
    case "$1" in
        --workdir) WORKDIR="$2"; shift 2 ;;
        --keep) KEEP=1; shift ;;
        -*) fail "unknown flag $1" ;;
        *) if [ -z "$CORPUS" ]; then CORPUS="$1"; else FILTER="$1"; fi; shift ;;
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
mapfile -t sniffs < <(grep -E -- "${FILTER:-.}" "$REPO_ROOT/blink/slevomat-fixable-sniffs.txt")
[ "${#sniffs[@]}" -gt 0 ] || fail "no sniff matches ${FILTER}"

if [ -z "$WORKDIR" ]; then
    WORKDIR="$(mktemp -d)"
    [ "$KEEP" -eq 1 ] || trap 'rm -rf "$WORKDIR"' EXIT
fi
mkdir -p "$WORKDIR"

log "building blink from $REPO_ROOT/blink"
blink_bin="$WORKDIR/blink"
( cd "$REPO_ROOT/blink" && go build -o "$blink_bin" . ) || fail "blink build failed"

# slevomat is no ECS dependency, so ECS autoloads a pinned checkout
slv="$WORKDIR/slevomat"
if [ ! -d "$slv/vendor" ]; then
    log "fetching slevomat/coding-standard $SLEVOMAT_REF"
    rm -rf "$slv"
    git init -q "$slv"
    git -C "$slv" fetch -q --depth 1 https://github.com/slevomat/coding-standard.git "$SLEVOMAT_REF" \
        && git -C "$slv" checkout -q FETCH_HEAD || fail "slevomat fetch failed"
    ( cd "$slv" && composer install -q --no-dev --no-plugins ) || fail "slevomat composer install failed"
fi

src="$WORKDIR/corpus"
if [ ! -d "$src" ]; then
    log "cloning $repo ($ref)"
    git clone --depth 1 --branch "$ref" "$repo" "$src" >/dev/null 2>&1 \
        || git clone --depth 1 "$repo" "$src" >/dev/null 2>&1 \
        || fail "clone failed: $repo"
fi
rev=$(git -C "$src" rev-parse --short HEAD 2>/dev/null || echo "?")
mapfile -t files < <(cd "$src" && find "${paths[@]}" -name '*.php' 2>/dev/null | sort)
log "corpus $name @ $rev: ${#files[@]} PHP files, ${#sniffs[@]} sniffs"

skip_args=$(jq -r '(.skip // []) | map("'"'"'\(.)'"'"'") | join(", ")' "$CORPUS")

# ecs_config <sniff> <tail>: an ECS config running one sniff, autoloading only
# slevomat and its phpdoc parser so PHP_CodeSniffer stays the ECS one
ecs_config() {
    cat <<PHP
<?php
declare(strict_types=1);
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
return \\Symplify\\EasyCodingStandard\\Config\\ECSConfig::configure()
    ->withRules([\\$1::class])
PHP
    [ -n "$skip_args" ] && echo "    ->withSkip([${skip_args}])"
    echo "    $2;"
}

# recheck <sniff-dir> <file>: prints the file when a fresh single-file ECS run
# still differs from blink and produces valid PHP
recheck() {
    local d f t
    d="$1"; f="$2"
    t=$(mktemp -d -p "$d" recheck.XXXXXX)
    mkdir -p "$t/tree/$(dirname "$f")"
    cp "$src/$f" "$t/tree/$f"
    sed "s#^    ->withCache(.*#    ->withCache('$t/cache');#" "$d/single.php" > "$t/ecs.php"
    "$REPO_ROOT/bin/ecs" check --fix --no-blink --clear-cache --no-progress-bar \
        --config "$t/ecs.php" "$t/tree/$f" >/dev/null 2>&1 || true
    if ! diff -q "$t/tree/$f" "$d/ours/$f" >/dev/null && php -l "$t/tree/$f" >/dev/null 2>&1; then
        echo "$f"
        diff -u --label "blink/$f" --label "ecs/$f" "$d/ours/$f" "$t/tree/$f" | head -n 40 > "$d/diff.$(echo "$f" | tr / _)"
    fi
    rm -rf "$t"
}
export -f recheck
export src REPO_ROOT

summary="$WORKDIR/summary.md"
{
    echo "## blink vs ECS slevomat parity: $name @ $rev"
    echo
    echo "| sniff | differing files |"
    echo "|---|---|"
} > "$summary"
details="$WORKDIR/details.md"
: > "$details"
failed=0
for sniff in "${sniffs[@]}"; do
    short="${sniff##*\\}"
    d="$WORKDIR/sniffs/$short"
    rm -rf "$d"; mkdir -p "$d"
    cp -r "$src" "$d/ours"; cp -r "$src" "$d/ecs"
    ecs_config "$sniff" '->withParallel()' > "$d/batch.php"
    ecs_config "$sniff" "->withCache('$d/cache')" > "$d/single.php"

    ours_paths=(); ecs_paths=()
    for p in "${paths[@]}"; do
        [ -d "$d/ours/$p" ] && ours_paths+=("$d/ours/$p")
        [ -d "$d/ecs/$p" ] && ecs_paths+=("$d/ecs/$p")
    done
    ECS_BLINK_BIN="$blink_bin" "$REPO_ROOT/bin/ecs" check --fix --clear-cache --no-progress-bar \
        --config "$d/batch.php" --blink "${ours_paths[@]}" >/dev/null 2>&1 || true
    "$REPO_ROOT/bin/ecs" check --fix --no-blink --clear-cache --no-progress-bar --memory-limit=1G \
        --config "$d/batch.php" "${ecs_paths[@]}" >/dev/null 2>&1 || true

    : > "$d/batch-differing"
    for f in "${files[@]}"; do
        diff -q "$d/ours/$f" "$d/ecs/$f" >/dev/null 2>&1 || echo "$f" >> "$d/batch-differing"
    done
    xargs -r -P "$(nproc)" -I{} bash -c 'recheck "$1" "$2"' _ "$d" {} < "$d/batch-differing" \
        | sort > "$d/differing"
    n=$(wc -l < "$d/differing")
    log "$short: $n differing ($(wc -l < "$d/batch-differing") before single-file recheck)"
    if [ "$n" -gt 0 ]; then
        failed=1
        echo "| \`$short\` | $n |" >> "$summary"
        while IFS= read -r f; do
            {
                echo "<details><summary><code>$short</code> <code>$f</code></summary>"
                echo
                echo '```diff'
                cat "$d/diff.$(echo "$f" | tr / _)"
                echo '```'
                echo '</details>'
                echo
            } >> "$details"
        done < <(head -n 5 "$d/differing")
    fi
    [ "$KEEP" -eq 1 ] || rm -rf "$d"
done

[ "$failed" -eq 0 ] && echo "| all ${#sniffs[@]} sniffs | 0 |" >> "$summary"
cat "$details" >> "$summary"
cat "$summary"
[ -n "${GITHUB_STEP_SUMMARY:-}" ] && cat "$summary" >> "$GITHUB_STEP_SUMMARY"

if [ "$failed" -eq 1 ]; then
    echo "::error::blink differs from ECS on slevomat sniffs over $name"
    exit 1
fi
exit 0
