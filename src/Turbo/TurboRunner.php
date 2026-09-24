<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Turbo;

/**
 * Experimental --turbo mode: hands the run over to the "reco" Go binary instead
 * of the PHP engine. See docs/turbo.md for the current limitations.
 *
 * @see \Symplify\EasyCodingStandard\Tests\Turbo\TurboRunnerTest
 */
final readonly class TurboRunner
{
    public function __construct(
        private RecoBinaryLocator $recoBinaryLocator,
    ) {
    }

    /**
     * @param string[] $paths
     */
    public function run(array $paths, bool $isFixMode): int
    {
        $binary = $this->recoBinaryLocator->locate();
        $arguments = $this->createArguments($binary, $paths, $isFixMode);

        // symfony/process is in "replace"; passthru streams reco's output straight through
        $command = implode(' ', array_map(escapeshellarg(...), $arguments));

        $exitCode = 0;
        passthru($command, $exitCode);

        return $exitCode;
    }

    /**
     * @param string[] $paths
     * @return string[]
     */
    public function createArguments(string $binary, array $paths, bool $isFixMode): array
    {
        $arguments = [$binary, 'run'];

        // reco rewrites in place by default; --dry-run only reports, matching the
        // check-vs-fix split of ECS itself
        if (! $isFixMode) {
            $arguments[] = '--dry-run';
        }

        return array_merge($arguments, array_values($paths));
    }
}
