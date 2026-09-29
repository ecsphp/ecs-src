<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\MethodChainingNewlineFixer\Fixture;

use Symplify\CodingStandard\Tests\Fixer\Spacing\MethodChainingNewlineFixer\Source\SomeClassWithChains;

class SkipMaybeTrailingMethod
{
    public function run(SomeClassWithChains $someClassWithChains)
    {
        return $someClassWithChains->one()->maybe();
    }
}
