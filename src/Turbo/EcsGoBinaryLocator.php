<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Turbo;

use Symplify\EasyCodingStandard\Turbo\Exception\EcsGoBinaryNotFoundException;

/**
 * Resolves the "ecs-go" Go binary that powers the experimental --turbo mode.
 *
 * @see \Symplify\EasyCodingStandard\Tests\Turbo\EcsGoBinaryLocatorTest
 */
final class EcsGoBinaryLocator
{
    private const string ENV_OVERRIDE = 'ECS_TURBO_BIN';

    public function locate(): string
    {
        $envBinary = getenv(self::ENV_OVERRIDE);
        if (is_string($envBinary) && $envBinary !== '' && is_file($envBinary)) {
            return $envBinary;
        }

        $vendorBinary = getcwd() . '/vendor/bin/ecs-go';
        if (is_file($vendorBinary)) {
            return $vendorBinary;
        }

        $pathBinary = $this->findOnPath('ecs-go');
        if ($pathBinary !== null) {
            return $pathBinary;
        }

        throw new EcsGoBinaryNotFoundException(sprintf(
            'The ecs-go binary for --turbo was not found in "%s" env, "vendor/bin/ecs-go" or on PATH. Build it from https://github.com/TomasVotruba/ecs-go and point "%s" to it.',
            self::ENV_OVERRIDE,
            self::ENV_OVERRIDE,
        ));
    }

    private function findOnPath(string $binaryName): ?string
    {
        $path = (string) getenv('PATH');

        foreach (explode(PATH_SEPARATOR, $path) as $directory) {
            if ($directory === '') {
                continue;
            }

            $candidate = $directory . DIRECTORY_SEPARATOR . $binaryName;
            if (is_file($candidate) && is_executable($candidate)) {
                return $candidate;
            }
        }

        return null;
    }
}
