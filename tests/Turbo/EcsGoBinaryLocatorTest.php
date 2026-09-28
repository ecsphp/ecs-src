<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Turbo;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Turbo\EcsGoBinaryLocator;
use Symplify\EasyCodingStandard\Turbo\Exception\EcsGoBinaryNotFoundException;

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

    public function testFindsBinaryOnPath(): void
    {
        putenv('ECS_TURBO_BIN');
        $originalPath = (string) getenv('PATH');
        putenv('PATH=' . __DIR__ . '/Source/bin');

        $this->assertSame(__DIR__ . '/Source/bin/ecs-go', $this->ecsGoBinaryLocator->locate());

        putenv('PATH=' . $originalPath);
    }

    public function testThrowsWhenNoBinaryFound(): void
    {
        putenv('ECS_TURBO_BIN');
        $originalPath = (string) getenv('PATH');
        putenv('PATH=');

        try {
            $this->expectException(EcsGoBinaryNotFoundException::class);
            $this->ecsGoBinaryLocator->locate();
        } finally {
            putenv('PATH=' . $originalPath);
        }
    }
}
