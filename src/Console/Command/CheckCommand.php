<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Console\Command;

use Entropy\Console\Contract\CommandInterface;
use Entropy\Console\Contract\DefaultCommandInterface;
use Symplify\EasyCodingStandard\Application\EasyCodingStandardApplication;
use Symplify\EasyCodingStandard\Configuration\ConfigInitializer;
use Symplify\EasyCodingStandard\Configuration\ConfigurationFactory;
use Symplify\EasyCodingStandard\Console\ExitCode;
use Symplify\EasyCodingStandard\Console\Output\ConsoleOutputFormatter;
use Symplify\EasyCodingStandard\MemoryLimitter;
use Symplify\EasyCodingStandard\Reporter\ProcessedFileReporter;
use Symplify\EasyCodingStandard\Turbo\RecoConfigDumper;
use Symplify\EasyCodingStandard\Turbo\TurboRunner;

final readonly class CheckCommand implements CommandInterface, DefaultCommandInterface
{
    public function __construct(
        private ProcessedFileReporter $processedFileReporter,
        private MemoryLimitter $memoryLimitter,
        private ConfigInitializer $configInitializer,
        private EasyCodingStandardApplication $easyCodingStandardApplication,
        private ConfigurationFactory $configurationFactory,
        private TurboRunner $turboRunner,
        private RecoConfigDumper $recoConfigDumper,
    ) {
    }

    public function getName(): string
    {
        return 'check';
    }

    public function getDescription(): string
    {
        return 'Check coding standard in one or more directories';
    }

    /**
     * @param bool   $turbo        [EXPERIMENTAL] run the reco Go binary instead of the PHP engine
     * @param string $config       Path to config file
     * @param string $outputFormat Select output format
     * @param string $memoryLimit  Memory limit for check
     * @param string $port         [INTERNAL] parallel TCP port
     * @param string $identifier   [INTERNAL] parallel identifier
     * @param string ...$paths     The path(s) to be checked.
     *
     * @option $config
     * @option $outputFormat
     * @option $memoryLimit
     * @option $port
     * @option $identifier
     *
     * @api invoked via reflection by the Entropy console application
     *
     * @return ExitCode::*
     */
    public function run(
        bool $fix = false,
        bool $clearCache = false,
        bool $noProgressBar = false,
        bool $noErrorTable = false,
        bool $noDiffs = false,
        bool $debug = false,
        bool $turbo = false,
        string $config = '',
        string $outputFormat = ConsoleOutputFormatter::NAME,
        string $memoryLimit = '',
        string $port = '',
        string $identifier = '',
        string ...$paths,
    ): int {
        // create ecs.php config file if does not exist yet
        if (! $this->configInitializer->areSomeCheckersRegistered()) {
            $this->configInitializer->createConfig((string) getcwd());
            return ExitCode::SUCCESS;
        }

        $configuration = $this->configurationFactory->create(
            array_values($paths),
            $fix,
            $clearCache,
            $noProgressBar,
            $noErrorTable,
            $noDiffs,
            $outputFormat,
            $config !== '' ? $config : null,
            $port,
            $identifier,
            $memoryLimit !== '' ? $memoryLimit : null,
            $debug,
        );

        // experimental: hand the resolved config to the reco Go binary and skip the PHP engine
        if ($turbo) {
            $configData = $this->recoConfigDumper->dump($configuration->getSources());
            $turboExitCode = $this->turboRunner->run($configData, $fix);
            return $turboExitCode === ExitCode::SUCCESS ? ExitCode::SUCCESS : ExitCode::CHANGED_CODE_OR_FOUND_ERRORS;
        }

        $this->memoryLimitter->adjust($configuration);

        $errorsAndDiffs = $this->easyCodingStandardApplication->run($configuration);
        return $this->processedFileReporter->report($errorsAndDiffs, $configuration);
    }
}
