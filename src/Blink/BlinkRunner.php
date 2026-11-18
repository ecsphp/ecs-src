<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Blink;

use Nette\Utils\FileSystem;
use Nette\Utils\Json;
use Symplify\EasyCodingStandard\Console\Output\ConsoleOutputFormatter;
use Symplify\EasyCodingStandard\Console\Output\JsonOutputFormatter;
use Symplify\EasyCodingStandard\Exception\ShouldNotHappenException;

/**
 * Experimental --blink mode: hands the run over to the "ecs-go" Go binary instead
 * of the PHP engine. The resolved ecs.php config (paths, rules, skips) is written
 * to a temp JSON file and passed to ecs-go via --ecs-config, so ecs-go maps the
 * ECS rules onto its own fixers. See docs/blink.md for the current limitations.
 *
 * @see \Symplify\EasyCodingStandard\Tests\Blink\BlinkRunnerTest
 */
final readonly class BlinkRunner
{
    public function __construct(
        private EcsGoBinaryLocator $ecsGoBinaryLocator,
    ) {
    }

    /**
     * @param array{paths: string[], rules: array<mixed>, skips: array<mixed>} $configData
     */
    public function run(
        array $configData,
        bool $isFixMode,
        bool $showDiffs = true,
        string $outputFormat = ConsoleOutputFormatter::NAME
    ): int {
        $binary = $this->ecsGoBinaryLocator->locate();

        $configPath = $this->writeConfig($configData);

        try {
            $arguments = $this->createArguments($binary, $configPath, $isFixMode, $showDiffs, $outputFormat);

            // string command, as the array form needs PHP 7.4 and the release is downgraded to 7.2
            $command = implode(' ', array_map(escapeshellarg(...), $arguments));

            // blink lints each fixed file with this PHP before writing it
            putenv('BLINK_PHP=' . PHP_BINARY);

            // inherit the terminal, so ecs-go detects a TTY and colors its output
            $process = proc_open($command, [STDIN, STDOUT, STDERR], $pipes);
            if (! is_resource($process)) {
                throw new ShouldNotHappenException(sprintf('Unable to start "%s"', $binary));
            }

            return proc_close($process);
        } finally {
            FileSystem::delete($configPath);
        }
    }

    /**
     * @return string[]
     */
    public function createArguments(
        string $binary,
        string $configPath,
        bool $isFixMode,
        bool $showDiffs = true,
        string $outputFormat = ConsoleOutputFormatter::NAME
    ): array {
        $arguments = [$binary, '--ecs-config', $configPath];

        // ecs-go reports by default; --fix rewrites in place, matching the
        // check-vs-fix split of ECS itself
        if ($isFixMode) {
            $arguments[] = '--fix';
        }

        if (! $showDiffs) {
            $arguments[] = '--no-diffs';
        }

        if ($outputFormat !== ConsoleOutputFormatter::NAME) {
            $arguments[] = '--output-format';
            $arguments[] = $outputFormat;
        }

        return $arguments;
    }

    /**
     * ecs-go writes the console and JSON reports; other formatters need the PHP engine
     */
    public function supportsOutputFormat(string $outputFormat): bool
    {
        return in_array($outputFormat, [ConsoleOutputFormatter::NAME, JsonOutputFormatter::getName()], true);
    }

    /**
     * @param array{paths: string[], rules: array<mixed>, skips: array<mixed>} $configData
     */
    private function writeConfig(array $configData): string
    {
        $configPath = tempnam(sys_get_temp_dir(), 'ecs-blink-') . '.json';
        file_put_contents($configPath, Json::encode($configData, Json::PRETTY));

        return $configPath;
    }
}
