<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Turbo;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Turbo\EcsGoBinaryLocator;

final class EcsGoBinaryLocatorTest extends TestCase
{
    private EcsGoBinaryLocator $ecsGoBinaryLocator;

    protected function setUp(): void
    {
        $this->ecsGoBinaryLocator = new EcsGoBinaryLocator();
    }

    public function testEnvironmentOverrideWins(): void
    {
        putenv('ECS_TURBO_BIN=' . __FILE__);

        $this->assertSame(__FILE__, $this->ecsGoBinaryLocator->locate());

        putenv('ECS_TURBO_BIN');
    }

    public function testFallsBackToPathWhenNoBinaryFound(): void
    {
        putenv('ECS_TURBO_BIN');

        $this->assertSame('ecs-go', $this->ecsGoBinaryLocator->locate());
    }
}
