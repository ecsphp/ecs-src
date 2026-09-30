<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Blink\BlinkRunner;
use Symplify\EasyCodingStandard\Blink\EcsGoBinaryLocator;

final class BlinkRunnerTest extends TestCase
{
    private BlinkRunner $blinkRunner;

    protected function setUp(): void
    {
        $this->blinkRunner = new BlinkRunner(new EcsGoBinaryLocator());
    }

    public function testCreateArgumentsInCheckModeReportsOnly(): void
    {
        $arguments = $this->blinkRunner->createArguments('ecs-go', '/tmp/ecs-blink.json', false);

        $this->assertSame(['ecs-go', '--ecs-config', '/tmp/ecs-blink.json'], $arguments);
    }

    public function testCreateArgumentsInFixModeRewritesInPlace(): void
    {
        $arguments = $this->blinkRunner->createArguments('ecs-go', '/tmp/ecs-blink.json', true);

        $this->assertSame(['ecs-go', '--ecs-config', '/tmp/ecs-blink.json', '--fix'], $arguments);
    }
}
