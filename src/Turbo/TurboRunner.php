<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Turbo;

use Nette\Utils\Json;

/**
 * Experimental --turbo mode: hands the run over to the "reco" Go binary instead
 * of the PHP engine. The resolved ecs.php config (paths, rules, skips) is written
 * to a temp JSON file and passed to reco via --ecs-config, so reco maps the ECS
 * rules to its native ones. See docs/turbo.md for the current limitations.
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
     * @param array{paths: string[], rules: array<mixed>, skips: array<mixed>} $configData
     */
    public function run(array $configData, bool $isFixMode): int
    {
        $binary = $this->recoBinaryLocator->locate();

        $configPath = $this->writeConfig($configData);

        try {
            $arguments = $this->createArguments($binary, $configPath, $isFixMode);

            // symfony/process is in "replace"; passthru streams reco's output straight through
            $command = implode(' ', array_map(escapeshellarg(...), $arguments));

            $exitCode = 0;
            passthru($command, $exitCode);

            return $exitCode;
        } finally {
            @unlink($configPath);
        }
    }

    /**
     * @return string[]
     */
    public function createArguments(string $binary, string $configPath, bool $isFixMode): array
    {
        $arguments = [$binary, 'run', '--ecs-config', $configPath];

        // reco rewrites in place by default; --dry-run only reports, matching the
        // check-vs-fix split of ECS itself
        if (! $isFixMode) {
            $arguments[] = '--dry-run';
        }

        return $arguments;
    }

    /**
     * @param array{paths: string[], rules: array<mixed>, skips: array<mixed>} $configData
     */
    private function writeConfig(array $configData): string
    {
        $configPath = tempnam(sys_get_temp_dir(), 'ecs-turbo-') . '.json';
        file_put_contents($configPath, Json::encode($configData, Json::PRETTY));

        return $configPath;
    }
}
