<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Turbo;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Turbo\RecoBinaryLocator;
use Symplify\EasyCodingStandard\Turbo\TurboRunner;

final class TurboRunnerTest extends TestCase
{
    private TurboRunner $turboRunner;

    protected function setUp(): void
    {
        $this->turboRunner = new TurboRunner(new RecoBinaryLocator());
    }

    public function testCreateArgumentsInCheckModeAddsDryRun(): void
    {
        $arguments = $this->turboRunner->createArguments('reco', '/tmp/ecs-turbo.json', false);

        $this->assertSame(['reco', 'run', '--ecs-config', '/tmp/ecs-turbo.json', '--dry-run'], $arguments);
    }

    public function testCreateArgumentsInFixModeRewritesInPlace(): void
    {
        $arguments = $this->turboRunner->createArguments('reco', '/tmp/ecs-turbo.json', true);

        $this->assertSame(['reco', 'run', '--ecs-config', '/tmp/ecs-turbo.json'], $arguments);
    }
}
