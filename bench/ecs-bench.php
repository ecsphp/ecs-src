<?php

declare(strict_types=1);

use Symplify\EasyCodingStandard\Config\ECSConfig;

// Heavy config for speed benchmarking: PER-CS + all Symplify sets.
// Target path comes from the BENCH_TARGET env var (see bench/run.sh).
$target = getenv('BENCH_TARGET');
if ($target === false || $target === '') {
    $target = '/var/www/rector-src/src';
}

return ECSConfig::configure()
    ->withPaths([$target])
    ->withPreparedSets(
        perCs: true,
        common: true,
        cleanCode: true,
        standaloneLine: true,
    );
