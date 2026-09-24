<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Turbo;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Turbo\EcsGoBinaryLocator;
use Symplify\EasyCodingStandard\Turbo\TurboRunner;

final class TurboRunnerTest extends TestCase
{
    private TurboRunner $turboRunner;

    protected function setUp(): void
    {
        $this->turboRunner = new TurboRunner(new EcsGoBinaryLocator());
    }

    public function testCreateArgumentsInCheckModeReportsOnly(): void
    {
        $arguments = $this->turboRunner->createArguments('ecs-go', '/tmp/ecs-turbo.json', false);

        $this->assertSame(['ecs-go', '--ecs-config', '/tmp/ecs-turbo.json'], $arguments);
    }

    public function testCreateArgumentsInFixModeRewritesInPlace(): void
    {
        $arguments = $this->turboRunner->createArguments('ecs-go', '/tmp/ecs-turbo.json', true);

        $this->assertSame(['ecs-go', '--ecs-config', '/tmp/ecs-turbo.json', '--fix'], $arguments);
    }
}
