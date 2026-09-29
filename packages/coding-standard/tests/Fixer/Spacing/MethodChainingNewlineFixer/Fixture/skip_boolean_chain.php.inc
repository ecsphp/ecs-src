<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\MethodChainingNewlineFixer\Fixture;

use PHPStan\Type\Type;

class SkipBooleanChain
{
    public function run(Type $paramType): bool
    {
        if (
            $paramType->isInteger()->yes()
            || $paramType->isBoolean()->yes()
            || $paramType->isNumericString()->yes()
        ) {
            return true;
        }

        return false;
    }
}
