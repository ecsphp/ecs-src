<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\DependencyInjection;

use Symplify\EasyCodingStandard\FixerRunner\Application\FixerFileProcessor;
use Symplify\EasyCodingStandard\SniffRunner\Application\SniffFileProcessor;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;

final class ConfigurationFileTest extends AbstractTestCase
{
    public function testEmptyConfig(): void
    {
        $this->createContainerWithConfigs([__DIR__ . '/ConfigurationFileSource/empty-config.php']);

        $fixerFileProcessor = $this->make(FixerFileProcessor::class);
        $this->assertCount(0, $fixerFileProcessor->getCheckers());

        $sniffFileProcessor = $this->make(SniffFileProcessor::class);
        $this->assertCount(0, $sniffFileProcessor->getCheckers());
    }

    public function testIncludeConfig(): void
    {
        $this->createContainerWithConfigs([__DIR__ . '/ConfigurationFileSource/include-another-config.php']);

        $fixerFileProcessor = $this->make(FixerFileProcessor::class);
        $this->assertCount(1, $fixerFileProcessor->getCheckers());

        $sniffFileProcessor = $this->make(SniffFileProcessor::class);
        $this->assertCount(1, $sniffFileProcessor->getCheckers());
    }

    public function testDeprecatedClosureConfig(): void
    {
        // the old closure config format is deprecated, but still loads
        $this->createContainerWithConfigs([__DIR__ . '/ConfigurationFileSource/deprecated-closure-config.php']);

        $fixerFileProcessor = $this->make(FixerFileProcessor::class);
        $this->assertCount(0, $fixerFileProcessor->getCheckers());
    }
}
