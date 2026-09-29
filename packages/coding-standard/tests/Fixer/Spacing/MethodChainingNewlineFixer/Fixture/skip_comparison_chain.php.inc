<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\MethodChainingNewlineFixer\Fixture;

use PHPStan\Reflection\MethodReflection;

class SkipComparisonChain
{
    public function isMethodSupported(MethodReflection $methodReflection): bool
    {
        return $methodReflection->getDeclaringClass()->getName() === ObjectDataMapper::class
            && $methodReflection->getName() === 'map';
    }
}
