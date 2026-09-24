<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Turbo;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Turbo\RecoBinaryLocator;

final class RecoBinaryLocatorTest extends TestCase
{
    private RecoBinaryLocator $recoBinaryLocator;

    protected function setUp(): void
    {
        $this->recoBinaryLocator = new RecoBinaryLocator();
    }

    public function testEnvironmentOverrideWins(): void
    {
        putenv('ECS_TURBO_BIN=' . __FILE__);

        $this->assertSame(__FILE__, $this->recoBinaryLocator->locate());

        putenv('ECS_TURBO_BIN');
    }

    public function testFallsBackToPathWhenNoBinaryFound(): void
    {
        putenv('ECS_TURBO_BIN');

        $this->assertSame('reco', $this->recoBinaryLocator->locate());
    }
}
