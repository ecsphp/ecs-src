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
        $arguments = $this->turboRunner->createArguments('reco', ['src', 'tests'], false);

        $this->assertSame(['reco', 'run', '--dry-run', 'src', 'tests'], $arguments);
    }

    public function testCreateArgumentsInFixModeRewritesInPlace(): void
    {
        $arguments = $this->turboRunner->createArguments('reco', ['src'], true);

        $this->assertSame(['reco', 'run', 'src'], $arguments);
    }
}
