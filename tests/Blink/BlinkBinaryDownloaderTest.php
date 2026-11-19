<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Blink\BlinkBinaryDownloader;
use Symplify\EasyCodingStandard\Blink\Exception\EcsGoBinaryNotFoundException;

final class BlinkBinaryDownloaderTest extends TestCase
{
    private string $releaseDirectory;

    private string $targetFile;

    private BlinkBinaryDownloader $blinkBinaryDownloader;

    protected function setUp(): void
    {
        $releaseBaseDirectory = sys_get_temp_dir() . '/ecs-blink-release-' . uniqid();
        $this->releaseDirectory = $releaseBaseDirectory . '/blink-0123456789ab';
        mkdir($this->releaseDirectory, 0777, true);

        file_put_contents($this->releaseDirectory . '/blink-linux-amd64', 'binary');

        $this->targetFile = $releaseBaseDirectory . '/downloaded-blink';
        $this->blinkBinaryDownloader = new BlinkBinaryDownloader('file://' . $releaseBaseDirectory);
    }

    protected function tearDown(): void
    {
        $releaseBaseDirectory = dirname($this->releaseDirectory);

        array_map(unlink(...), (glob($this->releaseDirectory . '/*') ?: []));
        rmdir($this->releaseDirectory);

        if (is_file($this->targetFile)) {
            unlink($this->targetFile);
        }

        rmdir($releaseBaseDirectory);
    }

    public function testDownloadsVerifiedBinary(): void
    {
        file_put_contents(
            $this->releaseDirectory . '/checksums.txt',
            hash('sha256', 'other') . "  blink-darwin-arm64\n" . hash('sha256', 'binary') . "  blink-linux-amd64\n"
        );

        $this->blinkBinaryDownloader->download('blink-0123456789ab', 'blink-linux-amd64', $this->targetFile);

        $this->assertSame('binary', file_get_contents($this->targetFile));
        $this->assertTrue(is_executable($this->targetFile));
    }

    public function testRejectsChecksumMismatch(): void
    {
        file_put_contents($this->releaseDirectory . '/checksums.txt', hash('sha256', 'other') . "  blink-linux-amd64\n");

        try {
            $this->expectException(EcsGoBinaryNotFoundException::class);
            $this->blinkBinaryDownloader->download('blink-0123456789ab', 'blink-linux-amd64', $this->targetFile);
        } finally {
            $this->assertFileDoesNotExist($this->targetFile);
        }
    }

    public function testRejectsMissingChecksum(): void
    {
        file_put_contents($this->releaseDirectory . '/checksums.txt', hash('sha256', 'binary') . "  blink-darwin-arm64\n");

        $this->expectException(EcsGoBinaryNotFoundException::class);
        $this->blinkBinaryDownloader->download('blink-0123456789ab', 'blink-linux-amd64', $this->targetFile);
    }

    public function testFailsOnMissingRelease(): void
    {
        $this->expectException(EcsGoBinaryNotFoundException::class);
        $this->blinkBinaryDownloader->download('blink-missing', 'blink-linux-amd64', $this->targetFile);
    }
}
