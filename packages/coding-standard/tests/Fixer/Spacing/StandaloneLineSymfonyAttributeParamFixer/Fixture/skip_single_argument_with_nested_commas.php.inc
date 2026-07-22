<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\StandaloneLineSymfonyAttributeParamFixer\Fixture;

use Symfony\Component\DependencyInjection\Attribute\Autowire;

final class SkipSingleArgumentWithNestedCommas
{
    public function __construct(
        #[Autowire(value: ['first', 'second', 'third'])]
        private readonly array $values
    ) {
    }
}
