<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Caching;

use Nette\Utils\Json;
use ReflectionClass;
use Symplify\EasyCodingStandard\Application\Version\StaticVersionResolver;
use Symplify\EasyCodingStandard\Config\ECSConfig;
use Symplify\EasyCodingStandard\DependencyInjection\SimpleParameterProvider;
use Symplify\EasyCodingStandard\Exception\Configuration\FileNotFoundException;
use Webmozart\Assert\Assert;

/**
 * @see \Symplify\EasyCodingStandard\Tests\ChangedFilesDetector\FileHashComputer\FileHashComputerTest
 */
final class FileHashComputer
{
    public function computeConfig(string $filePath): string
    {
        $callable = require $filePath;
        Assert::isCallable($callable);

        $ecsConfig = new ECSConfig();
        $callable($ecsConfig);

        // hash the registered checkers, their configuration and the source of the local ones
        $checkerConfiguration = $ecsConfig->getCheckerConfiguration();
        $fileHash = sha1(
            Json::encode($checkerConfiguration) . $this->hashLocalCheckerFiles(array_keys($checkerConfiguration))
        );

        return sha1($fileHash . SimpleParameterProvider::hash() . StaticVersionResolver::PACKAGE_VERSION);
    }

    public function compute(string $filePath): string
    {
        $fileHash = md5_file($filePath);
        if (! $fileHash) {
            throw new FileNotFoundException(sprintf('File "%s" was not found', $fileHash));
        }

        return $fileHash;
    }

    /**
     * A checker installed in vendor/ only changes with its package, but a project's own checker can change while its
     * class name and configuration stay the same, so its code has to be part of the hash.
     *
     * @param array<class-string> $checkerClasses
     */
    private function hashLocalCheckerFiles(array $checkerClasses): string
    {
        $fileHashes = [];
        foreach ($checkerClasses as $checkerClass) {
            foreach ($this->resolveClassFileNames(new ReflectionClass($checkerClass)) as $fileName) {
                if (str_contains($fileName, DIRECTORY_SEPARATOR . 'vendor' . DIRECTORY_SEPARATOR)) {
                    continue;
                }

                $fileHashes[$fileName] = $this->compute($fileName);
            }
        }

        // contents only, so that the same project checked out elsewhere keeps its cache
        $fileHashes = array_values($fileHashes);
        sort($fileHashes);

        return sha1(implode('', $fileHashes));
    }

    /**
     * @param ReflectionClass<object> $reflectionClass
     * @return string[]
     */
    private function resolveClassFileNames(ReflectionClass $reflectionClass): array
    {
        $fileNames = [];
        do {
            $fileNames[] = $reflectionClass->getFileName();
            foreach ($reflectionClass->getTraits() as $traitReflectionClass) {
                $fileNames[] = $traitReflectionClass->getFileName();
            }

            $reflectionClass = $reflectionClass->getParentClass();
        } while ($reflectionClass instanceof ReflectionClass);

        return array_filter($fileNames, is_string(...));
    }
}
