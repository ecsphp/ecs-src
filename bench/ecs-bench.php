<?php

declare(strict_types=1);

use Symplify\EasyCodingStandard\Config\ECSConfig;
use Symplify\EasyCodingStandard\DependencyInjection\SimpleParameterProvider;
use Symplify\EasyCodingStandard\ValueObject\Option;

// Heavy config for speed benchmarking: PER-CS + all Symplify sets.
// Target path comes from the BENCH_TARGET env var (see bench/run.sh).
$target = getenv('BENCH_TARGET');
if ($target === false || $target === '') {
    $target = '/var/www/rector-src/src';
}

// Force single process so the benchmark measures the engine, not parallel
// worker orchestration across cores.
SimpleParameterProvider::setParameter(Option::PARALLEL, false);

return ECSConfig::configure()
    ->withPaths([$target])
    ->withPreparedSets(
        perCs: true,
        common: true,
        cleanCode: true,
        standaloneLine: true,
    );
