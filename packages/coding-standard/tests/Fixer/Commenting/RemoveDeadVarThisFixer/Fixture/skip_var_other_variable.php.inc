<?php

namespace Symplify\CodingStandard\Tests\Fixer\Commenting\RemoveDeadVarThisFixer\Fixture;

function skipFunction()
{
    /** @var SomeType $value */
    $value = create();
}
