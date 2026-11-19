<?php

declare(strict_types=1);

// Regenerates the expected output of the slevomat sniff fixtures by running each
// sniff through ECS, so blink matches what ECS does (ECS ignores phpcs:disable
// comments, upstream's .fixed.php files assume them).
//
//   SLEVOMAT_DIR=/path/to/slevomat/coding-standard php blink/bin/slevomat-golden.php
//
// SLEVOMAT_DIR is a slevomat/coding-standard checkout with "composer install" run.
// A fixture ECS leaves unchanged gets no .fixed.php.

$slevomatDir = getenv('SLEVOMAT_DIR') ?: '';
if (! is_file($slevomatDir . '/SlevomatCodingStandard/Sniffs/TestCase.php')) {
    fwrite(STDERR, "Set SLEVOMAT_DIR to a slevomat/coding-standard checkout\n");
    exit(1);
}

$ecsBin = dirname(__DIR__, 2) . '/bin/ecs';
$dataDir = dirname(__DIR__) . '/internal/fixer/rules/testdata/slevomat';
$cases = json_decode((string) file_get_contents($dataDir . '/cases.json'), true, flags: JSON_THROW_ON_ERROR);

$autoload = sprintf(<<<'PHP'
<?php
spl_autoload_register(static function (string $class): void {
    foreach (['SlevomatCodingStandard\\' => %s, 'PHPStan\\PhpDocParser\\' => %s] as $prefix => $dir) {
        if (str_starts_with($class, $prefix)) {
            $file = $dir . str_replace('\\', '/', substr($class, strlen($prefix))) . '.php';
            if (is_file($file)) {
                require $file;
            }

            return;
        }
    }
}, true, true);
PHP
    , var_export($slevomatDir . '/SlevomatCodingStandard/', true), var_export($slevomatDir . '/vendor/phpstan/phpdoc-parser/src/', true));

$processes = [];
foreach ($cases as $case) {
    $workDir = sys_get_temp_dir() . '/slevomat-golden-' . md5($case['file'] . json_encode($case));
    @mkdir($workDir);
    $target = $workDir . '/' . basename((string) $case['file']);
    copy($dataDir . '/' . $case['file'], $target);
    file_put_contents($workDir . '/ecs.php', $autoload . sprintf(
        "\nreturn \\Symplify\\EasyCodingStandard\\Config\\ECSConfig::configure()->withPaths([%s])->withCache(%s)->withConfiguredRule(%s, %s);\n",
        var_export($target, true),
        var_export($workDir . '/cache', true),
        var_export($case['sniff'], true),
        var_export($case['config'] ?? [], true),
    ));

    $command = sprintf(
        'php %s check --fix --no-blink --clear-cache --no-progress-bar --config %s',
        escapeshellarg($ecsBin),
        escapeshellarg($workDir . '/ecs.php'),
    );
    $processes[] = [$case, $workDir, $target, proc_open($command, [
        1 => ['file', '/dev/null', 'w'],
        2 => ['pipe', 'w'],
    ], $pipes), $pipes];
}

$failed = false;
foreach ($processes as [$case, $workDir, $target, $process, $pipes]) {
    $stderr = stream_get_contents($pipes[2]);
    proc_close($process);

    $input = $dataDir . '/' . $case['file'];
    $expected = preg_replace('~\.php$~', '.fixed.php', $input);
    $output = (string) file_get_contents($target);
    if ($stderr !== '' && $output === file_get_contents($input)) {
        fwrite(STDERR, $case['file'] . ': ' . $stderr . "\n");
        $failed = true;
    }

    if ($output === file_get_contents($input)) {
        @unlink($expected);
    } else {
        file_put_contents($expected, $output);
    }

    exec('rm -rf ' . escapeshellarg($workDir));
}

exit($failed ? 1 : 0);
