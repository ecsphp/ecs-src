<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use Override;
use PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer;
use PhpCsFixer\Fixer\Import\FullyQualifiedStrictTypesFixer;
use Symplify\EasyCodingStandard\Blink\BlinkConfigDumper;
use Symplify\EasyCodingStandard\FixerRunner\Application\FixerFileProcessor;
use Symplify\EasyCodingStandard\Skipper\Matcher\FileInfoMatcher;
use Symplify\EasyCodingStandard\Skipper\Skipper\Skipper;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;

final class BlinkConfigDumperClassSkipTest extends AbstractTestCase
{
    private BlinkConfigDumper $blinkConfigDumper;

    #[Override]
    protected function setUp(): void
    {
        parent::setUp();

        $this->createContainerWithConfigs([__DIR__ . '/Source/interface-skip-ecs.php']);
        $this->blinkConfigDumper = $this->make(BlinkConfigDumper::class);
    }

    public function testDumpNamesCheckersCoveredByInterfaceSkip(): void
    {
        $skips = $this->blinkConfigDumper->dump([])['skips'];

        $this->assertContains([
            'class' => ArraySyntaxFixer::class,
            'paths' => [__DIR__ . '/Source/legacy'],
        ], $skips);
        $this->assertContains([
            'class' => FullyQualifiedStrictTypesFixer::class,
        ], $skips);
    }

    /**
     * The dumped skips, read by exact class, skip exactly where ECS's Skipper does.
     */
    public function testDumpSkipsWhatSkipperSkips(): void
    {
        $skips = $this->blinkConfigDumper->dump([])['skips'];
        $skipper = $this->make(Skipper::class);
        $fileInfoMatcher = $this->make(FileInfoMatcher::class);
        $fixers = $this->make(FixerFileProcessor::class)->getCheckers();
        $this->assertCount(4, $fixers);

        $files = ['legacy/A.php', 'generated/B.php', 'old/C.php', 'src/D.php'];

        $skippedSomewhere = 0;
        foreach ($fixers as $fixer) {
            foreach ($files as $file) {
                $filePath = __DIR__ . '/Source/' . $file;

                $dumpSkips = false;
                foreach ($skips as $skip) {
                    if (($skip['class'] ?? null) !== $fixer::class) {
                        continue;
                    }

                    if (! isset($skip['paths']) || $fileInfoMatcher->doesFileInfoMatchPatterns($filePath, $skip['paths'])) {
                        $dumpSkips = true;
                    }
                }

                $ecsSkips = $skipper->shouldSkipElementAndFilePath($fixer, $filePath);
                $this->assertSame($ecsSkips, $dumpSkips, sprintf('%s on %s', $fixer::class, $file));
                $skippedSomewhere += (int) $ecsSkips;
            }
        }

        // every skip in the config takes effect somewhere
        $this->assertGreaterThanOrEqual(8, $skippedSomewhere);
    }
}
