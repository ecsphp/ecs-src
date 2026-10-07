<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLinePromotedPropertyFixer\Fixture;

final class SkipEmptyLinesBetweenProperties
{
    public function __construct(
        private int $age,

        #[SomeAttribute]
        private string $name = 'default',

        #[SomeAttribute]
        private array $items = [],
    ) {
    }
}
