<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Configuration;

use PHPUnit\Framework\TestCase;
use ReflectionMethod;
use Symplify\EasyCodingStandard\Configuration\ECSConfigBuilder;

final class PreparedSetsReadmeTest extends TestCase
{
    private const string README_PATH = __DIR__ . '/../../build/target-repository/README.md';

    public function testEveryPreparedSetIsDocumented(): void
    {
        $readmeContents = file_get_contents(self::README_PATH);
        self::assertNotFalse($readmeContents);

        $reflectionMethod = new ReflectionMethod(ECSConfigBuilder::class, 'withPreparedSets');

        foreach ($reflectionMethod->getParameters() as $reflectionParameter) {
            $setName = $reflectionParameter->getName();

            self::assertStringContainsString(
                $setName,
                $readmeContents,
                sprintf('Prepared set "%s" is missing in README, add it to keep docs in sync', $setName)
            );
        }
    }
}
