<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Skipper\Skipper;

use Symplify\EasyCodingStandard\Skipper\Matcher\FileInfoMatcher;
use Symplify\EasyCodingStandard\Skipper\SkipCriteriaResolver\SkippedClassAndCodesResolver;
use Symplify\EasyCodingStandard\Skipper\SkipCriteriaResolver\SkippedClassResolver;
use Symplify\EasyCodingStandard\Skipper\SkipCriteriaResolver\SkippedMessagesResolver;
use Symplify\EasyCodingStandard\Skipper\SkipCriteriaResolver\SkippedPathsResolver;

/**
 * @api
 * @see \Symplify\EasyCodingStandard\Tests\Skipper\Skipper\Skipper\SkipperTest
 */
final readonly class Skipper
{
    private const string FILE_ELEMENT = 'file_elements';

    public function __construct(
        private SkippedClassAndCodesResolver $skippedClassAndCodesResolver,
        private SkippedClassResolver $skippedClassResolver,
        private SkippedMessagesResolver $skippedMessagesResolver,
        private SkippedPathsResolver $skippedPathsResolver,
        private SkipSkipper $skipSkipper,
        private FileInfoMatcher $fileInfoMatcher,
    ) {
    }

    public function shouldSkipElement(string|object $element): bool
    {
        return $this->shouldSkipElementAndFilePath($element, __FILE__);
    }

    public function shouldSkipFilePath(string $filePath): bool
    {
        return $this->shouldSkipElementAndFilePath(self::FILE_ELEMENT, $filePath);
    }

    public function shouldSkipElementAndFilePath(string|object $element, string $filePath): bool
    {
        if ($this->shouldSkipClassAndCode($element, $filePath)) {
            return true;
        }

        if ($this->shouldSkipClass($element, $filePath)) {
            return true;
        }

        if ($this->shouldSkipMessage($element, $filePath)) {
            return true;
        }

        return $this->shouldSkipPath($filePath);
    }

    private function shouldSkipClassAndCode(string|object $element, string $filePath): bool
    {
        if (! is_string($element)) {
            return false;
        }

        // e.g. App\Category\ArraySniff.SomeCode
        if (substr_count($element, '.') !== 1) {
            return false;
        }

        $skippedClassAndCodes = $this->skippedClassAndCodesResolver->resolve();
        if (! array_key_exists($element, $skippedClassAndCodes)) {
            return false;
        }

        $skippedPaths = $skippedClassAndCodes[$element];
        if ($skippedPaths === null) {
            return true;
        }

        return $this->fileInfoMatcher->doesFileInfoMatchPatterns($filePath, $skippedPaths);
    }

    private function shouldSkipClass(string|object $element, string $filePath): bool
    {
        if (is_string($element) && ! class_exists($element) && ! interface_exists($element)) {
            return false;
        }

        $skippedClasses = $this->skippedClassResolver->resolve();
        return $this->skipSkipper->doesMatchSkip($element, $filePath, $skippedClasses);
    }

    private function shouldSkipMessage(string|object $element, string $filePath): bool
    {
        if (! is_string($element)) {
            return false;
        }

        if (substr_count($element, ' ') === 0) {
            return false;
        }

        $skippedMessages = $this->skippedMessagesResolver->resolve();
        if (! array_key_exists($element, $skippedMessages)) {
            return false;
        }

        $skippedPaths = $skippedMessages[$element];
        if ($skippedPaths === null) {
            return true;
        }

        return $this->fileInfoMatcher->doesFileInfoMatchPatterns($filePath, $skippedPaths);
    }

    private function shouldSkipPath(string $filePath): bool
    {
        $skippedPaths = $this->skippedPathsResolver->resolve();
        return $this->fileInfoMatcher->doesFileInfoMatchPatterns($filePath, $skippedPaths);
    }
}
