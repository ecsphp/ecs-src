<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Turbo;

/**
 * Resolves the "reco" Go binary that powers the experimental --turbo mode.
 *
 * @see \Symplify\EasyCodingStandard\Tests\Turbo\RecoBinaryLocatorTest
 */
final class RecoBinaryLocator
{
    private const string ENV_OVERRIDE = 'ECS_TURBO_BIN';

    public function locate(): string
    {
        $envBinary = getenv(self::ENV_OVERRIDE);
        if (is_string($envBinary) && $envBinary !== '' && is_file($envBinary)) {
            return $envBinary;
        }

        $vendorBinary = getcwd() . '/vendor/bin/reco';
        if (is_file($vendorBinary)) {
            return $vendorBinary;
        }

        // fall back to the binary on PATH
        return 'reco';
    }
}
