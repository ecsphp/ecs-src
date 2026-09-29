<?php

namespace Symplify\CodingStandard\Tests\Fixer\Spacing\MethodChainingNewlineFixer\Fixture;

use PHPStan\Type\Type;

class SkipShortMethodInIfCondition
{
    public function run(Type $paramType): bool
    {
        if ($paramType->isInteger()->yes()) {
            return true;
        }

        return false;
    }
}
