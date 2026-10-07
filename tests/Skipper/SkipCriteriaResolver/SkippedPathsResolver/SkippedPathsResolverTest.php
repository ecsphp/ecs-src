<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Skipper\SkipCriteriaResolver\SkippedPathsResolver;

use Override;
use Symplify\EasyCodingStandard\Skipper\SkipCriteriaResolver\SkippedCriteriaResolver;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;

final class SkippedPathsResolverTest extends AbstractTestCase
{
    private SkippedCriteriaResolver $skippedCriteriaResolver;

    #[Override]
    protected function setUp(): void
    {
        $this->createContainerWithConfigs([__DIR__ . '/config/config.php']);
        $this->skippedCriteriaResolver = $this->make(SkippedCriteriaResolver::class);
    }

    public function test(): void
    {
        $skippedPaths = $this->skippedCriteriaResolver->resolvePaths();
        $this->assertCount(2, $skippedPaths);

        $this->assertSame('*/Mask/*', $skippedPaths[1]);
    }
}
