<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Set;

use Iterator;
use PHP_CodeSniffer\Sniffs\DeprecatedSniff;
use PhpCsFixer\Fixer\DeprecatedFixerInterface;
use PHPUnit\Framework\Attributes\DataProvider;
use ReflectionClass;
use Symplify\EasyCodingStandard\FixerRunner\Application\FixerFileProcessor;
use Symplify\EasyCodingStandard\SniffRunner\Application\SniffFileProcessor;
use Symplify\EasyCodingStandard\Testing\PHPUnit\AbstractTestCase;
use Symplify\EasyCodingStandard\ValueObject\Set\SetList;

final class NoDeprecatedRuleInSetTest extends AbstractTestCase
{
    #[DataProvider('provideSetPaths')]
    public function test(string $setName, string $setPath): void
    {
        $this->createContainerWithConfigs([$setPath]);

        $deprecatedRuleClasses = [];

        foreach ($this->make(FixerFileProcessor::class)->getCheckers() as $fixer) {
            if ($fixer instanceof DeprecatedFixerInterface) {
                $deprecatedRuleClasses[] = $fixer::class;
            }
        }

        foreach ($this->make(SniffFileProcessor::class)->getCheckers() as $sniff) {
            if ($sniff instanceof DeprecatedSniff) {
                $deprecatedRuleClasses[] = $sniff::class;
            }
        }

        $this->assertSame([], $deprecatedRuleClasses, sprintf(
            'Set "%s" registers deprecated rule(s): %s. Replace each with its successor.',
            $setName,
            implode(', ', $deprecatedRuleClasses)
        ));
    }

    public static function provideSetPaths(): Iterator
    {
        foreach (new ReflectionClass(SetList::class)->getConstants() as $name => $path) {
            yield $name => [$name, $path];
        }
    }
}
