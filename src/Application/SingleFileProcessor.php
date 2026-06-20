<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Application;

use Symplify\EasyCodingStandard\Caching\ChangedFilesDetector;
use Symplify\EasyCodingStandard\Skipper\Skipper\Skipper;
use Symplify\EasyCodingStandard\SniffRunner\ValueObject\Error\CodingStandardError;
use Symplify\EasyCodingStandard\ValueObject\Configuration;
use Symplify\EasyCodingStandard\ValueObject\Error\FileDiff;

final readonly class SingleFileProcessor
{
    public function __construct(
        private Skipper $skipper,
        private ChangedFilesDetector $changedFilesDetector,
        private FileProcessorCollector $fileProcessorCollector
    ) {
    }

    /**
     * @return array{file_diffs?: FileDiff[], coding_standard_errors?: CodingStandardError[]}
     */
    public function processFilePath(string $filePath, Configuration $configuration): array
    {
        if ($this->skipper->shouldSkipFilePath($filePath)) {
            return [];
        }

        $fileDiffs = [];
        $codingStandardErrors = [];

        $this->changedFilesDetector->addFilePath($filePath);
        $fileProcessors = $this->fileProcessorCollector->getFileProcessors();

        foreach ($fileProcessors as $fileProcessor) {
            if ($fileProcessor->getCheckers() === []) {
                continue;
            }

            $currentErrorsAndFileDiffs = $fileProcessor->processFile($filePath, $configuration);
            if ($currentErrorsAndFileDiffs === []) {
                continue;
            }

            $fileDiffs = [...$fileDiffs, ...($currentErrorsAndFileDiffs['file_diffs'] ?? [])];
            $codingStandardErrors = [
                ...$codingStandardErrors,
                ...($currentErrorsAndFileDiffs['coding_standard_errors'] ?? []),
            ];
        }

        // invalidate broken file, to analyse in next run too
        if ($fileDiffs !== [] || $codingStandardErrors !== []) {
            $this->changedFilesDetector->invalidateFilePath($filePath);
        }

        $errorsAndDiffs = [];
        if ($fileDiffs !== []) {
            $errorsAndDiffs['file_diffs'] = $fileDiffs;
        }

        if ($codingStandardErrors !== []) {
            $errorsAndDiffs['coding_standard_errors'] = $codingStandardErrors;
        }

        return $errorsAndDiffs;
    }
}
