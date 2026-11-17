<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use Override;
use Symplify\EasyCodingStandard\Blink\BlinkConfigDumper;
use Symplify\EasyCodingStandard\DependencyInjection\SimpleParameterProvider;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;
use Symplify\EasyCodingStandard\ValueObject\Option;

final class BlinkConfigDumperIndentTest extends AbstractTestCase
{
    #[Override]
    protected function tearDown(): void
    {
        SimpleParameterProvider::setParameter(Option::INDENTATION, Option::INDENTATION_SPACES);

        parent::tearDown();
    }

    public function testDumpDefaultIndent(): void
    {
        $this->createContainerWithConfigs([__DIR__ . '/Source/configured-ecs.php']);
        $blinkConfigDumper = $this->make(BlinkConfigDumper::class);

        $this->assertSame('    ', $blinkConfigDumper->dump([])['indent']);
    }

    public function testDumpTabIndent(): void
    {
        $this->createContainerWithConfigs([__DIR__ . '/Source/tab-indent-ecs.php']);
        $blinkConfigDumper = $this->make(BlinkConfigDumper::class);

        $this->assertSame("\t", $blinkConfigDumper->dump([])['indent']);
    }
}
