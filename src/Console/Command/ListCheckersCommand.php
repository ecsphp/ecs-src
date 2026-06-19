<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Console\Command;

use Entropy\Console\Contract\CommandInterface;
use Nette\Utils\Json;
use PHP_CodeSniffer\Sniffs\Sniff;
use PhpCsFixer\Fixer\FixerInterface;
use Symplify\EasyCodingStandard\Console\ExitCode;
use Symplify\EasyCodingStandard\Console\Output\ConsoleOutputFormatter;
use Symplify\EasyCodingStandard\Console\Reporter\CheckerListReporter;
use Symplify\EasyCodingStandard\FixerRunner\Application\FixerFileProcessor;
use Symplify\EasyCodingStandard\Skipper\SkipCriteriaResolver\SkippedClassResolver;
use Symplify\EasyCodingStandard\SniffRunner\Application\SniffFileProcessor;

final readonly class ListCheckersCommand implements CommandInterface
{
    public function __construct(
        private SniffFileProcessor $sniffFileProcessor,
        private FixerFileProcessor $fixerFileProcessor,
        private CheckerListReporter $checkerListReporter,
        private SkippedClassResolver $skippedClassResolver
    ) {
    }

    public function getName(): string
    {
        return 'list-checkers';
    }

    public function getDescription(): string
    {
        return 'Shows loaded checkers';
    }

    /**
     * @param string $outputFormat Select output format
     * @param string $config       Path to config file
     *
     * @option $outputFormat
     * @option $config
     *
     * @api invoked via reflection by the Entropy console application
     *
     * @return ExitCode::*
     */
    public function run(string $outputFormat = ConsoleOutputFormatter::NAME, string $config = ''): int
    {
        // include skipped rules to avoid adding those too
        $skippedCheckers = $this->getSkippedCheckers();

        if ($outputFormat === 'json') {
            $data = [
                'sniffs' => $this->getSniffClasses(),
                'fixers' => $this->getFixerClasses(),
                'skipped-checkers' => $skippedCheckers,
            ];

            echo Json::encode($data, Json::PRETTY) . PHP_EOL;

            return ExitCode::SUCCESS;
        }

        $this->checkerListReporter->report($this->getSniffClasses(), 'from PHP_CodeSniffer');
        $this->checkerListReporter->report($this->getFixerClasses(), 'from PHP-CS-Fixer');
        $this->checkerListReporter->report($skippedCheckers, 'are skipped');

        return ExitCode::SUCCESS;
    }

    /**
     * @return array<class-string<FixerInterface>>
     */
    private function getFixerClasses(): array
    {
        $fixers = $this->fixerFileProcessor->getCheckers();
        return $this->getObjectClasses($fixers);
    }

    /**
     * @return array<class-string<Sniff>>
     */
    private function getSniffClasses(): array
    {
        $sniffs = $this->sniffFileProcessor->getCheckers();
        return $this->getObjectClasses($sniffs);
    }

    /**
     * @template TObject as Sniff|FixerInterface
     * @param TObject[] $checkers
     * @return array<class-string<TObject>>
     */
    private function getObjectClasses(array $checkers): array
    {
        $objectClasses = array_map(static fn (object $fixer): string => $fixer::class, $checkers);
        sort($objectClasses);

        return $objectClasses;
    }

    /**
     * @return string[]
     */
    private function getSkippedCheckers(): array
    {
        $skippedCheckers = [];
        foreach ($this->skippedClassResolver->resolve() as $checkerClass => $fileList) {
            // ignore specific skips
            if ($fileList !== null) {
                continue;
            }

            $skippedCheckers[] = $checkerClass;
        }

        return $skippedCheckers;
    }
}
