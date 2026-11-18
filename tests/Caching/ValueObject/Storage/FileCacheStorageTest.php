<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Caching\ValueObject\Storage;

use Nette\Utils\FileSystem as UtilsFileSystem;
use PHPUnit\Framework\TestCase;
use Symfony\Component\Filesystem\Filesystem;
use Symplify\EasyCodingStandard\Caching\ValueObject\Storage\FileCacheStorage;

final class FileCacheStorageTest extends TestCase
{
    private string $cacheDirectory;

    private FileCacheStorage $fileCacheStorage;

    protected function setUp(): void
    {
        $this->cacheDirectory = sys_get_temp_dir() . '/ecs-file-cache-storage-' . uniqid();
        $this->fileCacheStorage = new FileCacheStorage($this->cacheDirectory, new Filesystem());
    }

    protected function tearDown(): void
    {
        UtilsFileSystem::delete($this->cacheDirectory);
    }

    public function testClearEmptiesTheDirectoryAndKeepsIt(): void
    {
        $this->fileCacheStorage->save('first', 'variable', 'data');
        $this->fileCacheStorage->save('second', 'variable', 'data');
        UtilsFileSystem::write($this->cacheDirectory . '/.hidden', '');

        $this->fileCacheStorage->clear();

        // the directory may be a mount point, which cannot be removed
        $this->assertDirectoryExists($this->cacheDirectory);
        $this->assertSame(['.', '..'], scandir($this->cacheDirectory));
        $this->assertNull($this->fileCacheStorage->load('first', 'variable'));
    }

    public function testClearWithoutDirectory(): void
    {
        $this->fileCacheStorage->clear();

        $this->assertDirectoryDoesNotExist($this->cacheDirectory);
    }
}
