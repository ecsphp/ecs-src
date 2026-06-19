<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\DependencyInjection;

use Symplify\EasyCodingStandard\Caching\ChangedFilesDetector;
use Symplify\EasyCodingStandard\Config\ECSConfig;

/**
 * @api
 */
final class EasyCodingStandardContainerFactory
{
    /**
     * @param string[] $argv
     */
    public function createFromArgv(array $argv): ECSConfig
    {
        $serviceContainerFactory = new ServiceContainerFactory();

        $inputConfigFiles = [];
        $rootECSConfig = getcwd() . DIRECTORY_SEPARATOR . 'ecs.php';

        $commandLineConfigFile = $this->resolveConfigFromArgv($argv);
        if ($commandLineConfigFile !== null && file_exists($commandLineConfigFile)) {
            // must be realpath, so container builder knows the location
            $inputConfigFiles[] = (string) realpath($commandLineConfigFile);
        } elseif (file_exists($rootECSConfig)) {
            $inputConfigFiles[] = $rootECSConfig;
        }

        $ecsConfig = $serviceContainerFactory->create($inputConfigFiles);
        $ecsConfig->boot();

        if ($inputConfigFiles !== []) {
            // for cache invalidation on config change
            /** @var ChangedFilesDetector $changedFilesDetector */
            $changedFilesDetector = $ecsConfig->make(ChangedFilesDetector::class);
            $changedFilesDetector->setUsedConfigs($inputConfigFiles);
        }

        return $ecsConfig;
    }

    /**
     * Resolve "--config <file>", "--config=<file>", "-c <file>" or "-c=<file>" from raw argv.
     *
     * @param string[] $argv
     */
    private function resolveConfigFromArgv(array $argv): ?string
    {
        foreach ($argv as $index => $arg) {
            if ($arg === '--config' || $arg === '-c') {
                return $argv[$index + 1] ?? null;
            }

            if (str_starts_with($arg, '--config=')) {
                return substr($arg, strlen('--config='));
            }

            if (str_starts_with($arg, '-c=')) {
                return substr($arg, strlen('-c='));
            }
        }

        return null;
    }
}
