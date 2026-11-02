<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use Override;
use PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer;
use PhpCsFixer\Fixer\Casing\LowercaseKeywordsFixer;
use PhpCsFixer\Fixer\Import\FullyQualifiedStrictTypesFixer;
use stdClass;
use Symplify\EasyCodingStandard\Blink\BlinkConfigDumper;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;

final class BlinkConfigDumperTest extends AbstractTestCase
{
    private BlinkConfigDumper $blinkConfigDumper;

    #[Override]
    protected function setUp(): void
    {
        parent::setUp();

        $this->createContainerWithConfigs([__DIR__ . '/Source/configured-ecs.php']);
        $this->blinkConfigDumper = $this->make(BlinkConfigDumper::class);
    }

    public function testDumpPassesPathsThrough(): void
    {
        $data = $this->blinkConfigDumper->dump(['src', 'tests']);

        $this->assertSame(['src', 'tests'], $data['paths']);
    }

    public function testDumpExtractsConfiguredFixerConfig(): void
    {
        $data = $this->blinkConfigDumper->dump([]);

        $configByClass = [];
        foreach ($data['rules'] as $rule) {
            $configByClass[$rule['class']] = $rule['config'];
        }

        $this->assertArrayHasKey(ArraySyntaxFixer::class, $configByClass);
        $this->assertSame([
            'syntax' => 'long',
        ], $configByClass[ArraySyntaxFixer::class]);
    }

    public function testDumpDropsOptionsLeftAtDefault(): void
    {
        $data = $this->blinkConfigDumper->dump([]);

        $configByClass = [];
        foreach ($data['rules'] as $rule) {
            $configByClass[$rule['class']] = $rule['config'];
        }

        // a configurable fixer left at its defaults dumps a config-less object,
        // not its resolved default options
        $this->assertArrayHasKey(FullyQualifiedStrictTypesFixer::class, $configByClass);
        $this->assertEquals(new stdClass(), $configByClass[FullyQualifiedStrictTypesFixer::class]);
    }

    public function testDumpReportsPathSkipAndClassSkip(): void
    {
        $data = $this->blinkConfigDumper->dump([]);

        $skipPaths = [];
        $skipClasses = [];
        foreach ($data['skips'] as $skip) {
            if (isset($skip['path'])) {
                $skipPaths[] = $skip['path'];
            } elseif (isset($skip['class']) && ! isset($skip['paths'])) {
                $skipClasses[] = $skip['class'];
            }
        }

        $this->assertContains(LowercaseKeywordsFixer::class, $skipClasses);

        $matchedGlob = array_filter($skipPaths, static fn (string $path): bool => str_contains($path, 'legacy'));
        $this->assertNotEmpty($matchedGlob, 'expected the legacy/* path skip to be dumped');
    }
}
