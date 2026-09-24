<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Turbo;

use Override;
use PhpCsFixer\Fixer\ArrayNotation\ArraySyntaxFixer;
use PhpCsFixer\Fixer\Casing\LowercaseKeywordsFixer;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;
use Symplify\EasyCodingStandard\Turbo\RecoConfigDumper;

final class RecoConfigDumperTest extends AbstractTestCase
{
    private RecoConfigDumper $recoConfigDumper;

    #[Override]
    protected function setUp(): void
    {
        parent::setUp();

        $this->createContainerWithConfigs([__DIR__ . '/Source/configured-ecs.php']);
        $this->recoConfigDumper = $this->make(RecoConfigDumper::class);
    }

    public function testDumpPassesPathsThrough(): void
    {
        $data = $this->recoConfigDumper->dump(['src', 'tests']);

        $this->assertSame(['src', 'tests'], $data['paths']);
    }

    public function testDumpExtractsConfiguredFixerConfig(): void
    {
        $data = $this->recoConfigDumper->dump([]);

        $configByClass = [];
        foreach ($data['rules'] as $rule) {
            $configByClass[$rule['class']] = $rule['config'];
        }

        $this->assertArrayHasKey(ArraySyntaxFixer::class, $configByClass);
        $this->assertSame([
            'syntax' => 'short',
        ], $configByClass[ArraySyntaxFixer::class]);
    }

    public function testDumpReportsPathSkipAndClassSkip(): void
    {
        $data = $this->recoConfigDumper->dump([]);

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
