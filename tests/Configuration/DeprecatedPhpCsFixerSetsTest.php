<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Configuration;

use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Configuration\ECSConfigBuilder;

final class DeprecatedPhpCsFixerSetsTest extends TestCase
{
    public function testAcceptsLegacyNamedArgumentsWithoutCrashing(): void
    {
        $ecsConfigBuilder = new ECSConfigBuilder();

        ob_start();
        $result = $ecsConfigBuilder->withPhpCsFixerSets(perCS30: true);
        ob_end_clean();

        self::assertSame($ecsConfigBuilder, $result);
    }
}
