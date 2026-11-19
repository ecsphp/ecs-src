#!/usr/bin/env bash
# Regenerates internal/fixer/rules/php_builtin_functions.txt: the lowercased
# names of PHP's built-in functions, which LowercasePHPFunctionsSniff reads from
# get_defined_functions() at runtime. Lists every loaded extension except the
# non-bundled PECL ones, so the result does not depend on a developer's setup.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
php -r '
$skip = ["xdebug", "mongodb", "rdkafka", "mailparse", "pcov", "imap", "apcu", "redis", "memcached", "imagick", "swoole", "grpc", "protobuf", "yaml", "igbinary", "msgpack", "amqp", "xhprof", "ast", "uopz"];
$names = [];
foreach (get_loaded_extensions() as $ext) {
    if (in_array(strtolower($ext), $skip, true)) {
        continue;
    }
    foreach (get_extension_funcs($ext) ?: [] as $fn) {
        $names[strtolower($fn)] = true;
    }
}
ksort($names, SORT_STRING);
echo implode("\n", array_keys($names)), "\n";
' > "$ROOT/internal/fixer/rules/php_builtin_functions.txt"
echo "Wrote $(wc -l < "$ROOT/internal/fixer/rules/php_builtin_functions.txt") functions"
