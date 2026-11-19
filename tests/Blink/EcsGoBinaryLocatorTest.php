<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Blink\BlinkBinaryDownloader;
use Symplify\EasyCodingStandard\Blink\EcsGoBinaryLocator;
use Symplify\EasyCodingStandard\Blink\Exception\EcsGoBinaryNotFoundException;

final class EcsGoBinaryLocatorTest extends TestCase
{
    private EcsGoBinaryLocator $ecsGoBinaryLocator;

    protected function setUp(): void
    {
        $this->ecsGoBinaryLocator = new EcsGoBinaryLocator(new BlinkBinaryDownloader());
    }

    public function testEnvironmentOverrideWins(): void
    {
        putenv('ECS_BLINK_BIN=' . __FILE__);

        $this->assertSame(__FILE__, $this->ecsGoBinaryLocator->locate());

        putenv('ECS_BLINK_BIN');
    }

    public function testDownloadsReleaseBinaryOnce(): void
    {
        putenv('ECS_BLINK_BIN');

        $releaseBaseDirectory = sys_get_temp_dir() . '/ecs-blink-release-' . uniqid();
        $binDirectory = $releaseBaseDirectory . '/bin';
        $releaseDirectory = $releaseBaseDirectory . '/blink-0123456789ab';
        mkdir($binDirectory, 0777, true);
        mkdir($releaseDirectory);

        $arch = in_array(strtolower(php_uname('m')), ['aarch64', 'arm64'], true) ? 'arm64' : 'amd64';
        $suffix = PHP_OS_FAMILY === 'Windows' ? '.exe' : '';
        $platform = strtolower(PHP_OS_FAMILY) . '-' . $arch . $suffix;

        file_put_contents($binDirectory . '/blink.version', "blink-0123456789ab\n");
        file_put_contents($releaseDirectory . '/blink-' . $platform, 'binary');
        file_put_contents($releaseDirectory . '/checksums.txt', hash('sha256', 'binary') . '  blink-' . $platform);

        $ecsGoBinaryLocator = new EcsGoBinaryLocator(
            new BlinkBinaryDownloader('file://' . $releaseBaseDirectory),
            $binDirectory
        );
        $releaseBinary = $binDirectory . '/blink-0123456789ab-' . $platform;

        try {
            $this->assertSame($releaseBinary, $ecsGoBinaryLocator->locate());
            $this->assertTrue(is_executable($releaseBinary));

            // cached: a second run must not need the release anymore
            unlink($releaseDirectory . '/blink-' . $platform);
            $this->assertSame($releaseBinary, $ecsGoBinaryLocator->locate());
        } finally {
            array_map(unlink(...), [...(glob($binDirectory . '/*') ?: []), ...(glob($releaseDirectory . '/*') ?: [])]);
            rmdir($binDirectory);
            rmdir($releaseDirectory);
            rmdir($releaseBaseDirectory);
        }
    }

    public function testFindsBinaryOnPath(): void
    {
        putenv('ECS_BLINK_BIN');
        $originalPath = (string) getenv('PATH');
        putenv('PATH=' . __DIR__ . '/Source/bin');

        $this->assertSame(__DIR__ . '/Source/bin/blink', $this->ecsGoBinaryLocator->locate());

        putenv('PATH=' . $originalPath);
    }

    public function testThrowsWhenNoBinaryFound(): void
    {
        putenv('ECS_BLINK_BIN');
        $originalPath = (string) getenv('PATH');
        putenv('PATH=');

        try {
            $this->expectException(EcsGoBinaryNotFoundException::class);
            $this->ecsGoBinaryLocator->locate();
        } finally {
            putenv('PATH=' . $originalPath);
        }
    }
}
