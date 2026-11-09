<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Configuration;

use PhpCsFixer\Fixer\FixerInterface;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;
use Symplify\EasyCodingStandard\Config\Level\ArrayLevel;
use Symplify\EasyCodingStandard\Config\Level\ControlStructuresLevel;
use Symplify\EasyCodingStandard\Config\Level\DocblockLevel;
use Symplify\EasyCodingStandard\Config\Level\SpacesLevel;

final class LevelRulesAreFixersTest extends TestCase
{
    /**
     * Levels must stay fixer-only: a single PHP_CodeSniffer sniff wakes the
     * sniff runner and tokenizes every file a second time.
     *
     * @param array<class-string> $rules
     */
    #[DataProvider('provideLevelRules')]
    public function testLevelRulesAreFixers(string $levelClass, array $rules): void
    {
        foreach ($rules as $rule) {
            self::assertTrue(
                is_a($rule, FixerInterface::class, true),
                sprintf('%s rule "%s" must be a fixer, not a sniff', $levelClass, $rule)
            );
        }
    }

    /**
     * @return iterable<array{class-string, array<class-string>}>
     */
    public static function provideLevelRules(): iterable
    {
        yield [SpacesLevel::class, SpacesLevel::RULES];
        yield [ArrayLevel::class, ArrayLevel::RULES];
        yield [ControlStructuresLevel::class, ControlStructuresLevel::RULES];
        yield [DocblockLevel::class, DocblockLevel::RULES];
    }
}
