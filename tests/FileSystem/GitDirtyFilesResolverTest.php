<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\FileSystem;

use Iterator;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\FileSystem\GitDirtyFilesResolver;

final class GitDirtyFilesResolverTest extends TestCase
{
    private GitDirtyFilesResolver $gitDirtyFilesResolver;

    protected function setUp(): void
    {
        $this->gitDirtyFilesResolver = new GitDirtyFilesResolver();
    }

    /**
     * @param string[] $statusLines
     * @param string[] $expectedPaths
     */
    #[DataProvider('provideData')]
    public function test(array $statusLines, array $expectedPaths): void
    {
        $relativePaths = $this->gitDirtyFilesResolver->resolveRelativePaths($statusLines);
        $this->assertSame($expectedPaths, $relativePaths);
    }

    /**
     * @return Iterator<array{string[], string[]}>
     */
    public static function provideData(): Iterator
    {
        yield [[' M src/Foo.php'], ['src/Foo.php']];
        yield [['?? src/New.php'], ['src/New.php']];
        yield [['A  src/Staged.php'], ['src/Staged.php']];
        yield [['MM src/Both.php'], ['src/Both.php']];
        yield [['R  src/Old.php -> src/New.php'], ['src/New.php']];
        yield [['?? "src/special\\t.php"'], ['src/special\\t.php']];
        yield [['', ' M src/Foo.php'], ['src/Foo.php']];
        yield [[' M src/Foo.php', '?? src/Bar.php'], ['src/Foo.php', 'src/Bar.php']];
    }
}
